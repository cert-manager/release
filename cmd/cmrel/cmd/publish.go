/*
Copyright 2021 The cert-manager Authors.

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
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/cloudbuild/v1"

	"github.com/cert-manager/release/pkg/gcb"
	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/sign"
)

const (
	publishCommand         = "publish"
	publishDescription     = "Publish a release from a staged tarball on GCS"
	publishLongDescription = `The publish command will build and publish a cert-manager release to a
public release locations. It will create a Google Cloud Build job
which will consume a staged build, run pre-release checks, push docker images,
Helm charts, generated static manifests and create a release tag on GitHub.

It can only be run by specifying a previously staged build.
`
	publishExample = ""
)

type publishOptions struct {
	// The name of the GCS bucket to publish the release to
	Bucket string

	// Name of the staged release to publish
	ReleaseName string

	// The path to the cloudbuild.yaml file used to perform the cert-manager crossbuild
	CloudBuildFile string

	// Project to run the GCB job in
	Project string

	// Name of the GitHub org to fetch cert-manager sources from
	Org string

	// Name of the GitHub repo to fetch cert-manager sources from
	Repo string

	// NoMock controls whether release artifacts are actually published.
	// If false, the command will exit after preparing the release for pushing.
	NoMock bool

	// PublishedImageRepository is the image repository that images as part of
	// releases should be pushed to.
	// It is used as the repository for manifest lists created for artifacts.
	PublishedImageRepository string

	// PublishedHelmChartGitHubOwner is the name of the owner of the GitHub repo
	// for Helm charts.
	PublishedHelmChartGitHubOwner string

	// PublishedHelmChartGitHubRepo is the name of the GitHub repository for
	// Helm charts.
	PublishedHelmChartGitHubRepo string

	// PublishedHelmChartGitHubBranch is the name of the main branch in the
	// GitHub repository for Helm Charts.
	PublishedHelmChartGitHubBranch string

	// PublishedGitHubOrg is the org of the repository where the release will
	// be published to.
	PublishedGitHubOrg string

	// PublishedGitHubRepo is the repo name in the provided org where the
	// release will be published to.
	PublishedGitHubRepo string

	// PublishActions is a list of publishing actions which should be taken,
	// or else "*" - the default - to mean "all actions"
	PublishActions []string

	// SkipSigning, if true, will skip trying to sign artifacts using KMS
	SkipSigning bool

	// SigningKMSKey is the full name of the GCP KMS key to be used for signing, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/versions/<KEY_VERSION>
	// This must be set if SkipSigning is not set to true
	SigningKMSKey string
}

func (o *publishOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to publish the release to.")
	fs.StringVar(&o.ReleaseName, "release-name", "", "Name of the staged release to publish.")
	fs.StringVar(&o.CloudBuildFile, "cloudbuild", "./gcb/publish/cloudbuild.yaml", "The path to the cloudbuild.yaml file used to publish the release. "+
		"The default value assumes that this tool is run from the root of the release repository.")
	fs.StringVar(&o.Project, "project", release.DefaultReleaseProject, "The GCP project to run the GCB build jobs in.")
	fs.BoolVar(&o.NoMock, "nomock", false, "Whether to actually publish the release. If false, the command will exit after preparing the release for pushing.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository to push the release images & manifest lists to.")
	fs.StringVar(&o.PublishedHelmChartGitHubOwner, "published-helm-chart-github-owner", release.DefaultHelmChartGitHubOwner, "The name of the owner of the GitHub repo for Helm charts.")
	fs.StringVar(&o.PublishedHelmChartGitHubRepo, "published-helm-chart-github-repo", release.DefaultHelmChartGitHubRepo, "The name of the GitHub repo for Helm charts.")
	fs.StringVar(&o.PublishedHelmChartGitHubBranch, "published-helm-chart-github-branch", release.DefaultHelmChartGitHubBranch, "The name of the main branch in the GitHub repository for Helm charts.")
	fs.StringVar(&o.PublishedGitHubOrg, "published-github-org", release.DefaultGitHubOrg, "The org of the repository where the release wil be published to.")
	fs.StringVar(&o.PublishedGitHubRepo, "published-github-repo", release.DefaultGitHubRepo, "The repo name in the provided org where the release will be published to.")
	fs.StringVar(&o.SigningKMSKey, "signing-kms-key", defaultKMSKey, "Full name of the GCP KMS key to use for signing.")
	fs.BoolVar(&o.SkipSigning, "skip-signing", false, "Skip signing container images.")
	fs.StringSliceVar(&o.PublishActions, "publish-actions", []string{"*"}, fmt.Sprintf("Comma-separated list of actions to take, or '*' to do everything. Only meaningful if nomock is set. Order of operations is preserved if given, or is alphabetical by default. Actions can be removed with a prefix of '-'. Options: %s", strings.Join(allPublishActionNames(), ", ")))
}

func (o *publishOptions) print() {
	log.Printf("Publish options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  ReleaseName: %q", o.ReleaseName)
	log.Printf("  CloudBuildFile: %q", o.CloudBuildFile)
	log.Printf("  Project: %q", o.Project)
	log.Printf("  NoMock: %t", o.NoMock)
	log.Printf("  PublishedImageRepo: %q", o.PublishedImageRepository)
	log.Printf("  PublishedHelmChartGitHubRepo: %q", o.PublishedHelmChartGitHubRepo)
	log.Printf("  PublishedHelmChartGitHubOwner: %q", o.PublishedHelmChartGitHubOwner)
	log.Printf("  PublishedHelmChartGitHubBranch: %q", o.PublishedHelmChartGitHubBranch)
	log.Printf("  PublishedGitHubOrg: %q", o.PublishedGitHubOrg)
	log.Printf("  PublishedGitHubRepo: %q", o.PublishedGitHubRepo)
	log.Printf("  PublishActions: %q", strings.Join(o.PublishActions, ","))
}

func publishCmd(rootOpts *rootOptions) *cobra.Command {
	o := &publishOptions{}
	cmd := &cobra.Command{
		Use:          publishCommand,
		Short:        publishDescription,
		Long:         publishLongDescription,
		Example:      publishExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

func runPublish(rootOpts *rootOptions, o *publishOptions) error {
	ctx := context.Background()

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	if o.SigningKMSKey != "" {
		if _, err := sign.NewGCPKMSKey(o.SigningKMSKey); err != nil {
			return err
		}
	}

	bucket := release.NewBucket(gcs.Bucket(o.Bucket), release.DefaultBucketPathPrefix, release.BuildTypeRelease)
	rel, err := bucket.GetRelease(ctx, o.ReleaseName)
	if err != nil {
		return fmt.Errorf("failed to fetch release: %w", err)
	}
	log.Printf("Release with version %q (%s) will be published", rel.Metadata().ReleaseVersion, rel.Metadata().GitCommitRef)

	log.Printf("DEBUG: Loading cloudbuild.yaml file from %q", o.CloudBuildFile)
	build, err := gcb.LoadBuild(o.CloudBuildFile)
	if err != nil {
		return fmt.Errorf("error loading cloudbuild.yaml file: %w", err)
	}

	// make sure that publish-actions is valid
	_, err = canonicalizeAndVerifyPublishActions(o.PublishActions)
	if err != nil {
		return fmt.Errorf("invalid publish-actions: %w", err)
	}

	build.Substitutions["_RELEASE_NAME"] = o.ReleaseName
	build.Substitutions["_RELEASE_BUCKET"] = o.Bucket
	build.Substitutions["_NO_MOCK"] = fmt.Sprintf("%t", o.NoMock)
	build.Substitutions["_PUBLISHED_GITHUB_ORG"] = o.PublishedGitHubOrg
	build.Substitutions["_PUBLISHED_GITHUB_REPO"] = o.PublishedGitHubRepo
	build.Substitutions["_PUBLISHED_HELM_CHART_GITHUB_OWNER"] = o.PublishedHelmChartGitHubOwner
	build.Substitutions["_PUBLISHED_HELM_CHART_GITHUB_REPO"] = o.PublishedHelmChartGitHubRepo
	build.Substitutions["_PUBLISHED_HELM_CHART_GITHUB_BRANCH"] = o.PublishedHelmChartGitHubBranch
	build.Substitutions["_PUBLISHED_IMAGE_REPO"] = o.PublishedImageRepository
	build.Substitutions["_PUBLISH_ACTIONS"] = strings.Join(o.PublishActions, ",")
	build.Substitutions["_SKIP_SIGNING"] = fmt.Sprintf("%v", o.SkipSigning)
	build.Substitutions["_KMS_KEY"] = o.SigningKMSKey

	log.Printf("DEBUG: building google cloud build API client")
	svc, err := cloudbuild.NewService(ctx)
	if err != nil {
		return fmt.Errorf("error building google cloud build API client: %w", err)
	}

	log.Printf("Submitting GCB build job...")
	build, err = gcb.SubmitBuild(svc, o.Project, build)
	if err != nil {
		return fmt.Errorf("error submitting build to cloud build: %w", err)
	}

	log.Println("---")
	log.Printf("Submitted publish job with name: %q", build.Id)
	log.Printf("  View logs at: %s", build.LogUrl)
	log.Printf("  Log bucket: %s", build.LogsBucket)
	log.Println("---")
	log.Printf("Waiting for publish job to complete, this may take a while...")
	build, err = gcb.WaitForBuild(svc, o.Project, build.Id)
	if err != nil {
		return fmt.Errorf("error waiting for cloud build to complete: %w", err)
	}

	if build.Status == gcb.Success {
		log.Printf("Release %q published!", rel.Metadata().ReleaseVersion)
	} else {
		log.Printf("An error occurred while publishing the release. Check the log files for more information: %s", build.LogUrl)
		return fmt.Errorf("publishing release failed")
	}

	return nil
}
