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
)

const (
	bootstrapPGPCommand         = "bootstrap-pgp"
	bootstrapPGPDescription     = "Bootstrap PGP identity from GCP KMS key and print"
	bootstrapPGPLongDescription = `The bootstrap-pgp command uses a specified KMS key in GCP to create a PGP identity
which can be used for code signing. The public identity can then be
distributed for use in signature verification, while the KMS key itself can be
used to sign cert-manager artifacts.

The public key is written to stdout in "armored" format.

The raw PEM-encoded public key is also written to stdout.

This command starts a cloudbuild job, since the only IAM entity with the required
permissions to use the key should be cloudbuild.`
)

var (
	bootstrapPGPExample = fmt.Sprintf(`To create a public PGP identity for a key:

%s %s --key "projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>"`, rootCommand, bootstrapPGPCommand)
)

type bootstrapPGPOptions struct {
	// Key is the full name of the GCP KMS key to be used, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	Key string

	// The path to the cloudbuild.yaml file to be invoked
	CloudBuildFile string

	// Project names the GCP project in which the GCB job will be run
	Project string
}

func (o *bootstrapPGPOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Key, "key", "", "Full name of the GCP KMS key to use for bootstrapping")
	fs.StringVar(&o.CloudBuildFile, "cloudbuild", "./gcb/bootstrap-pgp/cloudbuild.yaml", "The path to the cloudbuild.yaml file to be invoked.")
	fs.StringVar(&o.Project, "project", release.DefaultReleaseProject, "GCP project in which to run the GCB build job.")
	markRequired("key")
}

func (o *bootstrapPGPOptions) print() {
	log.Printf("bootstrap-pgp options:")
	log.Printf("             Key: %q", o.Key)
	log.Printf("         Project: %q", o.Project)
	log.Printf("  CloudBuildFile: %q", o.CloudBuildFile)
}

func bootstrapPGPCmd(rootOpts *rootOptions) *cobra.Command {
	o := &bootstrapPGPOptions{}
	cmd := &cobra.Command{
		Use:          bootstrapPGPCommand,
		Short:        bootstrapPGPDescription,
		Long:         bootstrapPGPLongDescription,
		Example:      bootstrapPGPExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBootstrapPGP(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

func runBootstrapPGP(rootOpts *rootOptions, o *bootstrapPGPOptions) error {
	ctx := context.Background()

	log.Printf("Bootstrapping PGP identity from %s", o.Key)

	log.Printf("DEBUG: Loading cloudbuild.yaml file from %q", o.CloudBuildFile)

	build, err := gcb.LoadBuild(o.CloudBuildFile)
	if err != nil {
		return fmt.Errorf("error loading %q: %w", o.CloudBuildFile, err)
	}

	build.Substitutions["_KMS_KEY"] = o.Key

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
	log.Printf("Submitted build with name: %q", build.Id)
	log.Printf("  View logs at: %s", build.LogUrl)
	log.Printf("  Log bucket: %s", build.LogsBucket)
	log.Println("---")
	log.Printf("Waiting for build to complete...")
	build, err = gcb.WaitForBuild(svc, o.Project, build.Id)
	if err != nil {
		return fmt.Errorf("error waiting for cloud build to complete: %w", err)
	}

	if build.Status == gcb.Success {
		log.Printf("Completed cloud build job; check log stdout for keys: %s", build.LogUrl)
	} else {
		log.Printf("An error occurred bootstrapping the PGP identity. Check the log files for more information: %s", build.LogUrl)
		return fmt.Errorf("bootstrapping PGP identity failed")
	}

	return nil
}
