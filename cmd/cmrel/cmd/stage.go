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

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/cloudbuild/v1"

	"github.com/cert-manager/release/pkg/gcb"
	"github.com/cert-manager/release/pkg/release"

	_ "embed"
)

const (
	stageCommand         = "stage"
	stageDescription     = "Stage release tarballs to a GCS release bucket"
	stageLongDescription = `The stage command will build and stage a cert-manager release to a
Google Cloud Storage bucket. It will create a Google Cloud Build job
which will run a full cross-build and publish the artifacts to the
staging release bucket.
`
)

var (
	stageExample = fmt.Sprintf(`
To stage a release of the 'master' branch to the default staging bucket at 'devel' path, run:

	%s %s --branch=master

To stage a release of the 'release-0.14' branch to the default staging bucket at 'release' path,
overriding the release version as 'v0.14.0', run:

	%s %s --branch=release-0.14 --release-version=v0.14.0`, rootCommand, stageCommand, rootCommand, stageCommand)
)

type stageOptions struct {
	// The name of the GCS bucket to stage the release to
	Bucket string

	// Name of the GitHub org to fetch cert-manager sources from
	Org string

	// Name of the GitHub repo to fetch cert-manager sources from
	Repo string

	// Name of the branch in the GitHub repo to build cert-manager sources from
	Branch string

	// Optional commit ref of cert-manager that should be staged
	GitRef string

	// Project is the name of the GCP project to run the GCB job in
	Project string

	// ReleaseVersion, if set, overrides the version git version tag used
	// during the build. This is used to force a build's version number to be
	// the final release tag before a tag has actually been created in the
	// repository.
	ReleaseVersion string

	// PublishedImageRepository is the docker repository that will be used for
	// built artifacts.
	// This must be set at the time a build is staged as parts of the release
	// incorporate this docker repository name.
	PublishedImageRepository string
}

func (o *stageOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to stage the release to.")
	fs.StringVar(&o.Org, "org", "jetstack", "Name of the GitHub org to fetch cert-manager sources from.")
	fs.StringVar(&o.Repo, "repo", "cert-manager", "Name of the GitHub repo to fetch cert-manager sources from.")
	fs.StringVar(&o.Branch, "branch", "master", "The git branch to build the release from. If --git-ref is not specified, the HEAD of this branch will be looked up on GitHub.")
	fs.StringVar(&o.GitRef, "git-ref", "", "The git commit ref of cert-manager that should be staged.")
	fs.StringVar(&o.Project, "project", release.DefaultReleaseProject, "The GCP project to run the GCB build jobs in.")
	fs.StringVar(&o.ReleaseVersion, "release-version", "", "Optional release version override used to force the version strings used during the release to a specific value. If not set, build is treated as development build and artifacts staged to 'devel' path.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository set when building the release.")
	markRequired("branch")
}

func (o *stageOptions) print() {
	log.Printf("Stage options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  Org: %q", o.Org)
	log.Printf("  Repo: %q", o.Repo)
	log.Printf("  Branch: %q", o.Branch)
	log.Printf("  GitRef: %q", o.GitRef)
	log.Printf("  Project: %q", o.Project)
	log.Printf("  ReleaseVersion: %q", o.ReleaseVersion)
	log.Printf("  PublishedImageRepo: %q", o.PublishedImageRepository)
}

func stageCmd(rootOpts *rootOptions) *cobra.Command {
	o := &stageOptions{}
	cmd := &cobra.Command{
		Use:          stageCommand,
		Short:        stageDescription,
		Long:         stageLongDescription,
		Example:      stageExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStage(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

//go:embed stage_cloudbuild.yaml
var cloudbuildStage []byte

func runStage(_ *rootOptions, o *stageOptions) error {
	if o.GitRef == "" {
		log.Printf("git-ref flag not specified, looking up git commit ref for %s/%s@%s", o.Org, o.Repo, o.Branch)
		ref, err := release.LookupBranchRef(o.Org, o.Repo, o.Branch)
		if err != nil {
			return fmt.Errorf("error looking up git commit ref: %w", err)
		}
		o.GitRef = ref
	}
	log.Printf("Staging build for %s/%s@%s", o.Org, o.Repo, o.GitRef)

	build, err := gcb.LoadCloudBuild(cloudbuildStage)
	if err != nil {
		return fmt.Errorf("error loading cloudbuild.yaml file: %w", err)
	}

	if build.Options == nil {
		build.Options = &cloudbuild.BuildOptions{MachineType: "n1-highcpu-32"}
	}

	build.Substitutions["_CM_REPO"] = fmt.Sprintf("https://github.com/%s/%s.git", o.Org, o.Repo)
	build.Substitutions["_CM_REF"] = o.GitRef
	build.Substitutions["_RELEASE_VERSION"] = o.ReleaseVersion
	build.Substitutions["_RELEASE_BUCKET"] = o.Bucket
	build.Substitutions["_TAG_RELEASE_BRANCH"] = o.Branch
	build.Substitutions["_PUBLISHED_IMAGE_REPO"] = o.PublishedImageRepository

	outputDir := ""
	// If --release-version is not explicitly set, we treat this build as a
	// 'devel' build and output into the development directory.
	if o.ReleaseVersion == "" {
		outputDir = release.BucketPathForRelease(release.DefaultBucketPathPrefix, release.BuildTypeDevel, "", o.GitRef)
	} else {
		outputDir = release.BucketPathForRelease(release.DefaultBucketPathPrefix, release.BuildTypeRelease, o.ReleaseVersion, o.GitRef)
	}

	log.Printf("DEBUG: building google cloud build API client")
	ctx := context.Background()
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
	log.Printf("Submitted build with name: %q", build.Id)
	log.Printf("  View logs at: %s", build.LogUrl)
	log.Printf("  Log bucket: %s", build.LogsBucket)
	log.Printf("  Once complete, view artifacts at: gs://%s/%s", o.Bucket, outputDir)
	log.Println("---")
	log.Printf("Waiting for build to complete, this may take a while...")
	build, err = gcb.WaitForBuild(svc, o.Project, build.Id)
	if err != nil {
		return fmt.Errorf("error waiting for cloud build to complete: %w", err)
	}

	if build.Status == gcb.Success {
		log.Printf("Release build complete - artifacts available at: gs://%s/%s", o.Bucket, outputDir)
	} else {
		log.Printf("An error occurred building the release. Check the log files for more information: %s", build.LogUrl)
		return fmt.Errorf("building release tarballs failed")
	}

	return nil
}
