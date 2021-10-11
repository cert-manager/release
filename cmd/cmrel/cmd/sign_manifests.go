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

	"github.com/cert-manager/release/pkg/sign"
)

const (
	signManifestsCommand         = "manifests"
	signManifestsDescription     = "Manually sign helm charts in a cert-manager-manifests.tar.gz artifact in-place, using a GCP KMS key"
	signManifestsLongDescription = `The manifests command signs a helm chart inside a
cert-manager-manifests.tar.gz artifact, using KMS rather than a local PGP key.

Mostly, this command is provided in case of an issue with the signing process
elsewhere. cmrel should attempt to create signatures as part of the normal
release process, and it shouldn't be neccessary to create a manual signature.

Using this shim rather than "helm package --sign" allows us to use exactly one
signing key for all kinds of cert-manager artifact, with no member of the team
having access to the actual private key.

The cert-manager-manifests.tar.gz file has the signature appended to it, in-place.`
)

var signManifestsExample = fmt.Sprintf(`To sign a manifests bundle at "/tmp/cert-manager-manifests.tar.gz:

%s %s %s --key "projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>" --path /tmp/cert-manager-manifests.tar.gz`, rootCommand, signCommand, signManifestsCommand)

type signManifestsOptions struct {
	// Key is the full name of the GCP KMS key to be used for signing, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	Key string

	// Path is the path to the cert-manager-manifests.tar.gz file
	Path string
}

func (o *signManifestsOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Key, "key", "", "Full name of the GCP KMS key to use for signing")
	fs.StringVar(&o.Path, "path", "", "Path to cert-manager-manifests.tar.gz")
	markRequired("key")
	markRequired("path")
}

func (o *signManifestsOptions) print() {
	log.Printf("sign manifests options:")
	log.Printf("   Key: %q", o.Key)
	log.Printf("  Path: %q", o.Path)
}

func signManifestsCmd(rootOpts *rootOptions) *cobra.Command {
	o := &signManifestsOptions{}
	cmd := &cobra.Command{
		Use:          signManifestsCommand,
		Short:        signManifestsDescription,
		Long:         signManifestsLongDescription,
		Example:      signManifestsExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSignManifests(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func runSignManifests(rootOpts *rootOptions, o *signManifestsOptions) error {
	ctx := context.Background()

	err := sign.CertManagerManifests(ctx, o.Key, o.Path)
	if err != nil {
		return fmt.Errorf("failed to complete signing of %q: %w", o.Path, err)
	}

	log.Printf("appended signature successfully to %q", o.Path)

	return nil
}
