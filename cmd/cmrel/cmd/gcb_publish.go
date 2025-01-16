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
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v35/github"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/release/docker"
	"github.com/cert-manager/release/pkg/release/helm"
	"github.com/cert-manager/release/pkg/release/publish/registry"
	"github.com/cert-manager/release/pkg/release/validation"
	"github.com/cert-manager/release/pkg/sign"
	"github.com/cert-manager/release/pkg/sign/cosign"
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

type publishAction func(context.Context, *gcbPublishOptions, *release.Unpacked) error

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

	// SkipSigning, if true, will skip trying to sign artifacts using KMS
	SkipSigning bool

	// SigningKMSKey is the full name of the GCP KMS key to be used for signing, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/versions/<KEY_VERSION>
	// This must be set if SkipSigning is not set to true
	SigningKMSKey string

	// PublishActions list of publishing actions to take
	PublishActions []string

	// CosignPath points to the location of the cosign binary
	CosignPath string

	// manualActionLogger logs to a buffer and is used by publish actions to log any manual
	// actions that must be taken by the user even after a successful publish is completed.
	// Get the log contents with ManualActionText()
	manualActionLogger *log.Logger

	manualActionBuffer bytes.Buffer
}

// NewGCBPublishOptions creates options and initializes loggers correctly
func NewGCBPublishOptions() *gcbPublishOptions {
	o := &gcbPublishOptions{}

	o.manualActionLogger = log.New(&o.manualActionBuffer, "* ", 0)

	return o
}

// ManualActionText returns a string containing any manual actions which have been logged using ManualActionLogger
func (o *gcbPublishOptions) ManualActionText() string {
	return o.manualActionBuffer.String()
}

// PublishActionList constructs a slice of artifact publishing functions based on the values
// listed in o.PublishActions.
func (o *gcbPublishOptions) PublishActionList() ([]publishAction, error) {
	actionNames, err := canonicalizeAndVerifyPublishActions(o.PublishActions)
	if err != nil {
		return nil, err
	}

	if len(actionNames) == 0 {
		return nil, fmt.Errorf("no artifacts to be published; nothing to do")
	}

	actionFuncs := make([]publishAction, len(actionNames))

	for i, action := range actionNames {
		// don't check if it's in map since we checked in canonicalizeAndVerifyPublishActions
		actionFuncs[i] = publishActionMap[action]
	}

	return actionFuncs, nil
}

func (o *gcbPublishOptions) GitHubClient(ctx context.Context) (*github.Client, error) {
	// construct the GitHub API client
	// The GITHUB_TOKEN must be a GitHub personal access token with at least
	// `repo` privileges and the associated user must have permission to create
	// branches and PRs at the Helm GitHub repository.
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set - a token is always required to create a release")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), nil
}

func (o *gcbPublishOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to stage the release to.")
	fs.StringVar(&o.ReleaseName, "release-name", "", "Name of the staged release to publish.")
	fs.BoolVar(&o.NoMock, "nomock", false, "Whether to actually publish the release. If false, the command will exit after preparing the release for pushing.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository to push the release images & manifest lists to.")
	fs.StringVar(&o.PublishedHelmChartGitHubOwner, "published-helm-chart-github-owner", release.DefaultHelmChartGitHubOwner, "The name of the owner of the GitHub repo for Helm charts.")
	fs.StringVar(&o.PublishedHelmChartGitHubRepo, "published-helm-chart-github-repo", release.DefaultHelmChartGitHubRepo, "The name of the GitHub repo for Helm charts.")
	fs.StringVar(&o.PublishedHelmChartGitHubBranch, "published-helm-chart-github-branch", release.DefaultHelmChartGitHubBranch, "The name of the main branch in the GitHub repository for Helm charts.")
	fs.StringVar(&o.PublishedGitHubOrg, "published-github-org", release.DefaultGitHubOrg, "The org of the repository where the release wil be published to.")
	fs.StringVar(&o.PublishedGitHubRepo, "published-github-repo", release.DefaultGitHubRepo, "The repo name in the provided org where the release will be published to.")
	fs.StringVar(&o.CosignPath, "cosign-path", "cosign", "Full path to the cosign binary. Defaults to searching in $PATH for a binary called 'cosign'")
	fs.StringVar(&o.SigningKMSKey, "signing-kms-key", defaultKMSKey, "Full name of the GCP KMS key to use for signing.")
	fs.BoolVar(&o.SkipSigning, "skip-signing", false, "Skip signing container images.")
	fs.StringSliceVar(&o.PublishActions, "publish-actions", []string{"*"}, fmt.Sprintf("Comma-separated list of actions to take, or '*' to do everything. Only meaningful if nomock is set. Operations are done in alphabetical order. Actions can be removed with a prefix of '-'. Options: %s", strings.Join(allPublishActionNames(), ", ")))
}

func (o *gcbPublishOptions) print() {
	log.Printf("GCB Publish options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  ReleaseName: %q", o.ReleaseName)
	log.Printf("  NoMock: %t", o.NoMock)
	log.Printf("  PublishedImageRepo: %q", o.PublishedImageRepository)
	log.Printf("  PublishedHelmChartGitHubRepo: %q", o.PublishedHelmChartGitHubRepo)
	log.Printf("  PublishedHelmChartGitHubOwner: %q", o.PublishedHelmChartGitHubOwner)
	log.Printf("  PublishedHelmChartGitHubBranch: %q", o.PublishedHelmChartGitHubBranch)
	log.Printf("  PublishedGitHubOrg: %q", o.PublishedGitHubOrg)
	log.Printf("  PublishedGitHubRepo: %q", o.PublishedGitHubRepo)
	log.Printf("  CosignPath: %q", o.CosignPath)
	log.Printf("  SkipSigning: %v", o.SkipSigning)
	log.Printf("  SigningKMSKey: %q", o.SigningKMSKey)
	log.Printf("  PublishActions: %q", strings.Join(o.PublishActions, ","))
}

func allPublishActionNames() []string {
	names := make([]string, len(publishActionMap))
	i := 0
	for k := range publishActionMap {
		names[i] = k
		i++
	}

	sort.Strings(names)
	return names
}

// canonicalizeAndVerifyPublishActions converts a list of raw actions into
// a slice of canonical action names (whitespace removed, lowercased), returning an error
// if any of the actions don't correspond to known actions. Supports removing actions via a prefix of "-"
// Actions are returned in alphabetical order
func canonicalizeAndVerifyPublishActions(rawActions []string) ([]string, error) {
	actions := sets.NewString()

	for _, rawAction := range rawActions {
		action := strings.ToLower(strings.TrimSpace(rawAction))

		if len(action) == 0 {
			continue
		}

		if action == "*" {
			actions = actions.Insert(allPublishActionNames()...)
			continue
		}

		_, ok := publishActionMap[strings.TrimPrefix(action, "-")]
		if !ok {
			return nil, fmt.Errorf("unknown action %q", rawAction)
		}

		if strings.HasPrefix(action, "-") {
			actions = actions.Delete(strings.TrimPrefix(action, "-"))
		} else {
			actions = actions.Insert(action)
		}
	}

	return actions.List(), nil
}

var publishActionMap map[string]publishAction = map[string]publishAction{
	"helmchartpr":         pushHelmChartPR,
	"githubrelease":       pushGitHubRelease,
	"pushcontainerimages": pushContainerImages,
}

func gcbPublishCmd(rootOpts *rootOptions) *cobra.Command {
	o := NewGCBPublishOptions()

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

	if o.SigningKMSKey != "" {
		if _, err := sign.NewGCPKMSKey(o.SigningKMSKey); err != nil {
			return err
		}

		log.Printf("getting cosign version information")
		if err := cosign.Version(ctx, o.CosignPath); err != nil {
			return fmt.Errorf("failed to query cosign version: %w", err)
		}
	}

	// fetch the staged release from GCS
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
			if err := docker.Load(ctx, t.Filepath()); err != nil {
				return err
			}
		}
	}

	if !o.NoMock {
		log.Printf("--nomock flag set to false, skipping actually publishing the release")
		return nil
	}

	log.Printf("!!! Publishing release artifacts to public repositories !!!")

	// TODO: perform check to ensure we have permission to create releases

	publishFuncs, err := o.PublishActionList()
	if err != nil {
		return fmt.Errorf("failed to parse published artifacts list: %w", err)
	}

	for _, publishFunc := range publishFuncs {
		err = publishFunc(ctx, o, rel)

		if err != nil {
			return errorDuringPublish(err)
		}
	}

	log.Println()
	log.Printf("+++++++++ Publishing release completed successfully! +++++++++")
	log.Printf("You MUST now perform the following manual tasks:\n%s", o.ManualActionText())

	return nil
}

func pushHelmChartPR(ctx context.Context, o *gcbPublishOptions, rel *release.Unpacked) error {
	githubClient, err := o.GitHubClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create github client for pushing helm chart PR: %w", err)
	}

	helmRepo := helm.NewGitHubRepositoryManager(
		&helm.GitHubClient{
			GitClient:          githubClient.Git,
			PullRequestClient:  githubClient.PullRequests,
			RepositoriesClient: githubClient.Repositories,
			UsersClient:        githubClient.Users,
		},
		o.PublishedHelmChartGitHubOwner,
		o.PublishedHelmChartGitHubRepo,
		o.PublishedHelmChartGitHubBranch,
	)
	if err := helmRepo.Check(ctx); err != nil {
		return fmt.Errorf("error in preflight checks for Helm GitHub repository: %v", err)
	}

	log.Printf("Pushing Helm chart(s)")

	prURLForHelmCharts, err := helmRepo.Publish(ctx, rel.ReleaseName, rel.Charts...)
	if err != nil {
		return err
	}

	o.manualActionLogger.Printf("Review and merge the GitHub PR containing the Helm charts: %s", prURLForHelmCharts)

	return nil
}

func pushGitHubRelease(ctx context.Context, o *gcbPublishOptions, rel *release.Unpacked) error {
	githubClient, err := o.GitHubClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create github client for creating github release: %w", err)
	}

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

	log.Printf("Creating a draft GitHub release %q in repository %s/%s", rel.ReleaseVersion, o.PublishedGitHubOrg, o.PublishedGitHubRepo)

	defaultReleaseBody := "!!! Update this release note body before publishing this draft release!"
	githubRelease, resp, err := githubClient.Repositories.CreateRelease(ctx, o.PublishedGitHubOrg, o.PublishedGitHubRepo, &github.RepositoryRelease{
		TagName:         &rel.ReleaseVersion,
		TargetCommitish: &rel.GitCommitRef,
		Name:            &rel.ReleaseVersion,
		Body:            &defaultReleaseBody,
		Draft:           pointer.Bool(true),
		// TODO: determine whether this ReleaseVersion is a 'prerelease'
		Prerelease: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub release: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected response code when creating GitHub release %d", resp.StatusCode)
	}

	log.Printf("Uploading %d release manifests to GitHub release", len(manifestsByName))
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

	if release.CmctlIsShipped(rel.ReleaseVersion) {
		// Open ctl binary tar files ahead of time to ensure they are available
		// on disk.
		ctlBinariesByName := map[string]*os.File{}
		for _, ctlBinary := range rel.CtlBinaryBundles {
			f, err := os.Open(ctlBinary.Filepath())
			if err != nil {
				return fmt.Errorf("failed to open manifest file to be uploaded: %v", err)
			}

			defer f.Close()

			ctlBinariesByName[ctlBinary.ArtifactFilename()] = f
		}

		log.Printf("Uploading %d release binary tars to GitHub release", len(ctlBinariesByName))
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

	}

	o.manualActionLogger.Printf("Update the GitHub release with release notes and hit PUBLISH!")
	return nil
}

const registryWaitTime = time.Second * 2

func pushContainerImages(ctx context.Context, o *gcbPublishOptions, rel *release.Unpacked) error {
	log.Printf("Pushing arch-specific docker images")

	if o.SigningKMSKey == "" && !o.SkipSigning {
		return fmt.Errorf("must set signing-kms-key or skip-signing in order to sign images")
	}

	var pushedContent []string

	for name, tars := range rel.ComponentImageBundles {
		log.Printf("Pushing release images for component %q", name)
		for _, t := range tars {
			imageTag := buildImageTag(o.PublishedImageRepository, name, t.Architecture(), rel.ReleaseVersion)

			log.Printf("Tagging %q with new name %q", t.RawImageName(), imageTag)

			if err := docker.Tag(ctx, t.RawImageName(), imageTag); err != nil {
				return err
			}

			if err := docker.Push(ctx, imageTag); err != nil {
				return err
			}

			// PublishedTag will be used later to refer to the image under the tag we
			// actually pushed it under
			t.PublishedTag = imageTag

			log.Printf("Pushed release image %q", imageTag)
			pushedContent = append(pushedContent, imageTag)

			// Wait to avoid being rate limited by the registry
			time.Sleep(registryWaitTime)
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
		if err := registry.CreateManifestList(ctx, manifestListName, tars); err != nil {
			return err
		}

		builtManifestLists = append(builtManifestLists, manifestListName)
	}

	log.Printf("Pushing all multi-arch manifest lists")
	for _, manifestListName := range builtManifestLists {
		log.Printf("Pushing manifest list %q", manifestListName)
		if err := docker.PushManifestList(ctx, manifestListName); err != nil {
			return err
		}

		pushedContent = append(pushedContent, manifestListName)
		log.Printf("Pushed multi-arch manifest list %q", manifestListName)

		// Wait to avoid being rate limited by the registry
		time.Sleep(registryWaitTime)
	}

	if err := signRegistryContent(ctx, o, pushedContent); err != nil {
		return fmt.Errorf("failed to sign images: %w", err)
	}

	return nil
}

func signRegistryContent(ctx context.Context, o *gcbPublishOptions, allContentToSign []string) error {
	if o.SkipSigning {
		log.Println("Skipping signing container images / manifest lists as skip-signing is set")
		return nil
	}

	log.Println("Signing container images")

	parsedKey, err := sign.NewGCPKMSKey(o.SigningKMSKey)
	if err != nil {
		return err
	}

	for _, toSign := range allContentToSign {
		log.Printf("Signing %q", toSign)
		if err := cosign.Sign(ctx, o.CosignPath, []string{toSign}, parsedKey); err != nil {
			return fmt.Errorf("failed to sign container image / manifest list %q: %w", toSign, err)
		}

		// Wait to avoid being rate limited by the registry
		time.Sleep(registryWaitTime)
	}

	log.Printf("Finished signing: %s", strings.Join(allContentToSign, ", "))

	return nil
}

func buildManifestListName(repo, componentName, tag string) string {
	return fmt.Sprintf("%s/cert-manager-%s:%s", repo, componentName, tag)
}

func buildImageTag(repo, componentName, arch, tag string) string {
	return fmt.Sprintf("%s/cert-manager-%s-%s:%s", repo, componentName, arch, tag)
}

func errorDuringPublish(err error) error {
	if err != nil {
		log.Printf("ERROR OCCURRED DURING PUBLISHING - INCOMPLETE RELEASE MAY BE PUBLISHED: %v", err)
	}
	return err
}
