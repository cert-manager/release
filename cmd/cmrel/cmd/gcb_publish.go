/*
Copyright 2020 The Jetstack cert-manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v29/github"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/release/docker"
	"github.com/cert-manager/release/pkg/release/publish/registry"
	"github.com/cert-manager/release/pkg/release/validation"
)

const (
	gcbPublishCommand         = "publish"
	gcbPublishDescription     = "Publish a staged release to the public-facing artifact repositories"
	gcbPublishLongDescription = `
The 'gcb publish' subcommand will fetch a staged release from GCS, verify its
integrity and publish artifacts to public-facing artifact repositories (e.g.
Quay.io, GitHub releases and the Helm chart repostory).

It requires Docker to be installed and available.

The GitHub token to use to create the draft release should be set using the
GITHUB_TOKEN environment variable.
`
)

type gcbPublishOptions struct {
	// The name of the GCS bucket to stage the release to.
	Bucket string

	// Name of the staged release to publish
	ReleaseName string

	// NoMock controls whether release artifacts are actually published.
	// If false, the command will exit after preparing the release for pushing.
	NoMock bool

	// PublishedImageRepository is the image repository that images as part of
	// releases should be pushed to.
	// It is used as the repository for manifest lists created for artifacts.
	PublishedImageRepository string

	// PublishedHelmChartBucket is the name of the GCS bucket where published
	// Helm charts should be stored.
	PublishedHelmChartBucket string

	// PublishedGitHubOrg is the org of the repository where the release will
	// be published to.
	PublishedGitHubOrg string

	// PublishedGitHubRepo is the repo name in the provided org where the
	// release will be published to.
	PublishedGitHubRepo string
}

func (o *gcbPublishOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to stage the release to.")
	fs.StringVar(&o.ReleaseName, "release-name", "", "Name of the staged release to publish.")
	fs.BoolVar(&o.NoMock, "nomock", false, "Whether to actually publish the release. If false, the command will exit after preparing the release for pushing.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository to push the release images & manifest lists to.")
	fs.StringVar(&o.PublishedHelmChartBucket, "published-helm-chart-bucket", release.DefaultHelmChartBucket, "The name of the GCS bucket where published Helm charts should be stored.")
	fs.StringVar(&o.PublishedGitHubOrg, "published-github-org", release.DefaultGitHubOrg, "The org of the repository where the release wil be published to.")
	fs.StringVar(&o.PublishedGitHubRepo, "published-github-repo", release.DefaultGitHubRepo, "The repo name in the provided org where the release will be published to.")
}

func (o *gcbPublishOptions) print() {
	log.Printf("GCB Publish options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  ReleaseName: %q", o.ReleaseName)
	log.Printf("  NoMock: %t", o.NoMock)
	log.Printf("  PublishedImageRepo: %q", o.PublishedImageRepository)
	log.Printf("  PublishedHelmChartBucket: %q", o.PublishedHelmChartBucket)
	log.Printf("  PublishedGitHubOrg: %q", o.PublishedGitHubOrg)
	log.Printf("  PublishedGitHubRepo: %q", o.PublishedGitHubRepo)
}

func gcbPublishCmd(rootOpts *rootOptions) *cobra.Command {
	o := &gcbPublishOptions{}
	cmd := &cobra.Command{
		Use:          gcbPublishCommand,
		Short:        gcbPublishDescription,
		Long:         gcbPublishLongDescription,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGCBPublish(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

func runGCBPublish(rootOpts *rootOptions, o *gcbPublishOptions) error {
	ctx := context.Background()
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	bucket := release.NewBucket(gcs.Bucket(o.Bucket), release.DefaultBucketPathPrefix, release.BuildTypeRelease)
	staged, err := bucket.GetRelease(ctx, o.ReleaseName)
	if err != nil {
		return fmt.Errorf("failed to fetch release: %w", err)
	}

	log.Printf("Release with version %q (%s) will be published", staged.Metadata().ReleaseVersion, staged.Metadata().GitCommitRef)

	rel, err := release.Unpack(ctx, staged)
	if err != nil {
		return fmt.Errorf("failed to unpack staged release: %w", err)
	}

	// validate the release artifacts are roughly as expected
	validationOpts := validation.Options{
		ReleaseVersion:  staged.Metadata().ReleaseVersion,
		ImageRepository: o.PublishedImageRepository,
	}
	violations, err := validation.ValidateUnpackedRelease(validationOpts, rel)
	if err != nil {
		return fmt.Errorf("failed to validate unpacked release: %w", err)
	}
	if len(violations) > 0 {
		log.Printf("Release validation failed:")
		for _, v := range violations {
			log.Printf("  - %s", v)
		}
		return fmt.Errorf("release failed validation - refusing to publish")
	}
	log.Printf("Release validation succeeded!")

	for name, tars := range rel.ComponentImageBundles {
		log.Printf("Loading release images for component %q into local docker daemon...", name)
		for _, t := range tars {
			if err := docker.Load(t.Filepath()); err != nil {
				return err
			}
		}
	}

	for name, tars := range rel.UBIImageBundles {
		log.Printf("Loading UBI release images for component %q into local docker daemon...", name)
		for _, t := range tars {
			if err := docker.Load(t.Filepath()); err != nil {
				return err
			}
		}
	}

	if !o.NoMock {
		log.Printf("--nomock flag set to false, skipping actually publishing the release")
		return nil
	}

	// wrap errors from pushRelease to ensure we log a big warning message if
	// one is returned to inform users that a half-released version may be out.
	return errorDuringPublish(pushRelease(o, rel))
}

func pushRelease(o *gcbPublishOptions, rel *release.Unpacked) error {
	log.Printf("!!! Publishing release artifacts to public repositories !!!")
	ctx := context.TODO()

	// build required clients _first_ to try and avoid publishing partial
	// releases due to permissions issues
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error building GCS client: %w", err)
	}
	chartBucket := gcs.Bucket(o.PublishedHelmChartBucket)

	// TODO: perform check to ensure we have permission to write to the bucket

	// construct the GitHub API client
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable not set - a token is always required to create a release")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	// TODO: perform check to ensure we have permission to create releases

	// open manifest files ahead of time to ensure they are available on disk
	manifestsByName := map[string]*os.File{}
	for _, manifest := range rel.YAMLs {
		f, err := os.Open(manifest.Path())
		if err != nil {
			return fmt.Errorf("failed to open manifest file to be uploaded: %v", err)
		}
		defer f.Close()
		manifestsByName[filepath.Base(manifest.Path())] = f
	}

	// open ctl binary files ahead of time to ensure they are available on disk
	ctlBinariesByName := map[string]*os.File{}
	for _, files := range rel.CtlBinaries {
		for _, binary := range files {
			f, err := os.Open(binary.Filepath())
			if err != nil {
				return fmt.Errorf("failed to open manifest file to be uploaded: %v", err)
			}
			defer f.Close()
			manifestsByName[fmt.Sprintf("cert-manager-ctl-%s-%s", binary.OS(), binary.Architecture())] = f
		}
	}

	log.Printf("Pushing arch-specific docker images")
	for name, tars := range rel.ComponentImageBundles {
		log.Printf("Pushing release images for component %q", name)
		for _, t := range tars {
			if err := docker.Push(t.ImageName()); err != nil {
				return err
			}
			log.Printf("Pushed release image %q", t.ImageName())
			// Wait 2 seconds to avoid being rate limited by the registry.
			time.Sleep(time.Second * 2)
		}
	}

	// manifest lists can only be created using the docker CLI after the child
	// images have been pushed to the registry.
	// Build them all at once, and push them afterwards to avoid releasing an
	// incomplete set of manifest lists.
	var builtManifestLists []string
	log.Printf("Creating multi-arch manifest lists for image components")
	for name, tars := range rel.ComponentImageBundles {
		manifestListName := buildManifestListName(o.PublishedImageRepository, name, rel.ReleaseVersion)
		if err := registry.CreateManifestList(manifestListName, tars); err != nil {
			return err
		}
		builtManifestLists = append(builtManifestLists, manifestListName)
	}

	log.Printf("Pushing arch-specific UBI docker images")
	for name, tars := range rel.UBIImageBundles {
		log.Printf("Pushing UBI release images for component %q", name)
		for _, t := range tars {
			if err := docker.Push(t.ImageName()); err != nil {
				return err
			}
			log.Printf("Pushed UBI release image %q", t.ImageName())
			// Wait 2 seconds to avoid being rate limited by the registry.
			time.Sleep(time.Second * 2)
		}
	}

	log.Printf("Creating multi-arch manifest lists for UBI image components")
	for name, tars := range rel.UBIImageBundles {
		manifestListName := buildManifestListName(o.PublishedImageRepository, name, rel.ReleaseVersion+"-ubi")
		if err := registry.CreateManifestList(manifestListName, tars); err != nil {
			return err
		}
		builtManifestLists = append(builtManifestLists, manifestListName)
	}

	log.Printf("Pushing all multi-arch manifest lists")
	for _, manifestListName := range builtManifestLists {
		log.Printf("Pushing manifest list %q", manifestListName)
		if err := docker.Command("", "manifest", "push", manifestListName); err != nil {
			return err
		}
		log.Printf("Pushed multi-arch manifest list %q", manifestListName)
	}

	log.Printf("Pushing Helm chart(s) to release bucket")
	for _, chart := range rel.Charts {
		if err := func() error {
			chartFileName := chart.PackageFileName()
			log.Printf("Copying chart %q to release bucket gs://%s/", chartFileName, o.PublishedHelmChartBucket)
			r, err := os.Open(chart.Path())
			if err != nil {
				return err
			}
			defer r.Close()
			w := chartBucket.Object(chartFileName).NewWriter(ctx)
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
			log.Printf("Copied chart %q to release bucket", chartFileName)
			return nil
		}(); err != nil {
			return err
		}
	}

	log.Printf("Creating a draft GitHub release %q in repository %s/%s", rel.ReleaseVersion, o.PublishedGitHubOrg, o.PublishedGitHubRepo)
	trueBool := true
	defaultReleaseBody := "!!! Update this release note body before publishing this draft release!"
	githubRelease, resp, err := githubClient.Repositories.CreateRelease(ctx, o.PublishedGitHubOrg, o.PublishedGitHubRepo, &github.RepositoryRelease{
		TagName:         &rel.ReleaseVersion,
		TargetCommitish: &rel.GitCommitRef,
		Name:            &rel.ReleaseVersion,
		Body:            &defaultReleaseBody,
		Draft:           &trueBool,
		// TODO: determine whether this ReleaseVersion is a 'prerelease'
		Prerelease: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub release: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected response code when creating GitHub release %d", resp.StatusCode)
	}

	log.Printf("Uploading %d release manifests to GitHub release", len(rel.YAMLs))
	for name, f := range manifestsByName {
		asset, resp, err := githubClient.Repositories.UploadReleaseAsset(ctx, o.PublishedGitHubOrg, o.PublishedGitHubRepo, *githubRelease.ID, &github.UploadOptions{
			Name: name,
		}, f)
		if err != nil {
			return fmt.Errorf("failed to upload github release asset: %v", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("unexpected response code when uploading github release asset %d", resp.StatusCode)
		}
		log.Printf("Uploaded asset %q to GitHub release %q", *asset.Name, *githubRelease.Name)
	}

	for name, f := range ctlBinariesByName {
		asset, resp, err := githubClient.Repositories.UploadReleaseAsset(ctx, o.PublishedGitHubOrg, o.PublishedGitHubRepo, *githubRelease.ID, &github.UploadOptions{
			Name: name,
		}, f)
		if err != nil {
			return fmt.Errorf("failed to upload github release asset: %v", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("unexpected response code when uploading github release asset %d", resp.StatusCode)
		}
		log.Printf("Uploaded asset %q to GitHub release %q", *asset.Name, *githubRelease.Name)
	}

	log.Println()
	log.Printf("+++++++++ Publishing release completed successfully! Please update the GitHub release with release notes and hit PUBLISH! +++++++++")
	return nil
}

func buildManifestListName(repo, componentName, tag string) string {
	return fmt.Sprintf("%s/cert-manager-%s:%s", repo, componentName, tag)
}

func errorDuringPublish(err error) error {
	if err != nil {
		log.Printf("ERROR OCCURRED DURING PUBLISHING - INCOMPLETE RELEASE MAY BE PUBLISHED: %v", err)
	}
	return err
}
