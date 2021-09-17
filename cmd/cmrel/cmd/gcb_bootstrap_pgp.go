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
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"

	"github.com/cert-manager/release/pkg/sign"
)

const (
	// intentionally the same command + description between this and bootstrap-pgp
	gcbBootstrapPGPCommand         = bootstrapPGPCommand
	gcbBootstrapPGPDescription     = bootstrapPGPDescription
	gcbBootstrapPGPLongDescription = `The bootstrap-pgp command uses a specified KMS key in GCP to create a PGP identity
which can be used for code signing. The public identity can then be
distributed for use in signature verification, while the KMS key itself can be
used to sign cert-manager artifacts.

The public key is written to stdout in "armored" format.

The raw PEM-encoded public key is also written to stdout.

This is the internal version of the 'bootstrap-pgp' target. It is intended to be run by
a Google Cloud Build started via the 'bootstrap-pgp' sub-command.`
)

var gcbBootstrapPGPExample = fmt.Sprintf(`To create a public PGP identity for a key:

%s gcb %s --key "projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>"`, rootCommand, gcbBootstrapPGPCommand)

type gcbBootstrapPGPOptions struct {
	// Key is the full name of the GCP KMS key to be used, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	Key string
}

func (o *gcbBootstrapPGPOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Key, "key", "", "Full name of the GCP KMS key to use for bootstrapping")
	markRequired("key")
}

func (o *gcbBootstrapPGPOptions) print() {
	log.Printf("bootstrap-pgp options:")
	log.Printf("         Key: %q", o.Key)
}

func gcbBootstrapPGPCmd(rootOpts *rootOptions) *cobra.Command {
	o := &gcbBootstrapPGPOptions{}
	cmd := &cobra.Command{
		Use:          gcbBootstrapPGPCommand,
		Short:        gcbBootstrapPGPDescription,
		Long:         gcbBootstrapPGPLongDescription,
		Example:      gcbBootstrapPGPExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGCBBootstrapPGP(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func getPEMPubkey(ctx context.Context, key string) (string, error) {
	oauthClient, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return "", fmt.Errorf("could not create GCP OAuth2 client: %w", err)
	}

	svc, err := cloudkms.NewService(ctx, option.WithHTTPClient(oauthClient))
	if err != nil {
		return "", fmt.Errorf("could not create GCP KMS client: %w", err)
	}

	res, err := svc.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.GetPublicKey(key).Do()
	if err != nil {
		return "", fmt.Errorf("could not get public key from Google Cloud KMS API: %w", err)
	}

	return res.Pem, nil
}

func runGCBBootstrapPGP(rootOpts *rootOptions, o *gcbBootstrapPGPOptions) error {
	ctx := context.Background()

	armoredKey, err := sign.BootstrapPGPFromGCP(ctx, o.Key)
	if err != nil {
		return fmt.Errorf("failed to bootstrap PGP identity using %q: %w", o.Key, err)
	}

	pemPubkey, err := getPEMPubkey(ctx, o.Key)
	if err != nil {
		return fmt.Errorf("failed to get pubkey for %q: %w", o.Key, err)
	}

	fmt.Printf("armored signed PGP public identity:\n%s\n", armoredKey)

	fmt.Printf("PEM formatted raw public key:\n%s\n", pemPubkey)

	return nil
}
