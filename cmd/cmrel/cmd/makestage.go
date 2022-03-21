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
	"github.com/cert-manager/release/pkg/sign"
)

const (
	makeStageCommand         = "makestage"
	makeStageDescription     = "Build a staged release using make, then copy artifacts to GCS"
	makeStageLongDescription = `makestage builds a staged release using make, then copies artifacts to GCS`
)

var (
	makeStageExample = fmt.Sprintf(`
To stage a release of the 'release-1.8' branch to the default bucket under the path 'release/':

	%s %s --ref=release-0.14

To stage a release of the 'v1.8.0' tag to the default bucket under the path 'release/':

	%s %s --ref=v1.8.0`, rootCommand, makeStageCommand, rootCommand, makeStageCommand)
)

type makeStageOptions struct {
	// The name of the GCS bucket to stage the release to
	Bucket string

	// Name of the GitHub org to fetch cert-manager sources from
	Org string

	// Name of the GitHub repo to fetch cert-manager sources from
	Repo string

	// Ref is the git ref to check out when building
	Ref string

	// The path to the cloudbuild.yaml file to be used
	CloudBuildFile string

	// Project is the name of the GCP project to run the GCB job in
	Project string

	// PublishedImageRepository is the docker repository that will be used for
	// built artifacts.
	// This must be set at the time a build is staged as parts of the release
	// incorporate this docker repository name.
	PublishedImageRepository string

	// SigningKMSKey is the full name of the GCP KMS key to be used for signing, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	// This must be set if SkipSigning is not set to true
	SigningKMSKey string
}

func (o *makeStageOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to stage the release to.")
	fs.StringVar(&o.Org, "org", "cert-manager", "Name of the GitHub org to fetch cert-manager sources from.")
	fs.StringVar(&o.Repo, "repo", "cert-manager", "Name of the GitHub repo to fetch cert-manager sources from.")
	fs.StringVar(&o.Ref, "ref", "master", "The git ref to build the release from.")
	fs.StringVar(&o.CloudBuildFile, "cloudbuild", "./gcb/makestage/cloudbuild.yaml", "The path to the cloudbuild.yaml file used to perform the cert-manager crossbuild. "+
		"The default value assumes that this tool is run from the root of the release repository.")
	fs.StringVar(&o.Project, "project", release.DefaultReleaseProject, "The GCP project to run the GCB build jobs in.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository set when building the release.")
	fs.StringVar(&o.SigningKMSKey, "signing-kms-key", defaultKMSKey, "Full name of the GCP KMS key to use for signing")

	markRequired("ref")
}

func (o *makeStageOptions) print() {
	log.Printf("Stage options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  Org: %q", o.Org)
	log.Printf("  Repo: %q", o.Repo)
	log.Printf("  Ref: %q", o.Ref)
	log.Printf("  CloudBuildFile: %q", o.CloudBuildFile)
	log.Printf("  Project: %q", o.Project)
	log.Printf("  PublishedImageRepo: %q", o.PublishedImageRepository)
	log.Printf("  SigningKMSKey: %q", o.SigningKMSKey)
}

func makeStageCmd(rootOpts *rootOptions) *cobra.Command {
	o := &makeStageOptions{}
	cmd := &cobra.Command{
		Use:          makeStageCommand,
		Short:        makeStageDescription,
		Long:         makeStageLongDescription,
		Example:      makeStageExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeStage(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func runMakeStage(rootOpts *rootOptions, o *makeStageOptions) error {
	if o.SigningKMSKey != "" {
		if _, err := sign.NewGCPKMSKey(o.SigningKMSKey); err != nil {
			return err
		}
	}

	log.Printf("Staging build for %s/%s@%s", o.Org, o.Repo, o.Ref)

	log.Printf("DEBUG: Loading cloudbuild.yaml file from %q", o.CloudBuildFile)
	build, err := gcb.LoadBuild(o.CloudBuildFile)
	if err != nil {
		return fmt.Errorf("error loading cloudbuild.yaml file: %w", err)
	}

	if build.Options == nil {
		build.Options = &cloudbuild.BuildOptions{MachineType: "n1-highcpu-32"}
	}

	build.Substitutions["_CM_REF"] = o.Ref
	build.Substitutions["_CM_REPO"] = fmt.Sprintf("https://github.com/%s/%s.git", o.Org, o.Repo)
	build.Substitutions["_RELEASE_BUCKET"] = o.Bucket
	build.Substitutions["_KMS_KEY"] = o.SigningKMSKey

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
	log.Println("---")
	log.Printf("Waiting for build to complete, this may take a while...")

	build, err = gcb.WaitForBuild(svc, o.Project, build.Id)
	if err != nil {
		return fmt.Errorf("error waiting for cloud build to complete: %w", err)
	}

	if build.Status != gcb.Success {
		log.Printf("An error occurred building the release. Check the log files for more information: %s", build.LogUrl)
		return fmt.Errorf("building release tarballs failed")
	}

	log.Printf("Release build complete for ref %s", o.Ref)

	return nil
}
