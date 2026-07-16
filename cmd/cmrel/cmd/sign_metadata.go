/*
Copyright 2026 The cert-manager Authors.

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
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/sign"
)

const (
	signMetadataCommand         = "metadata"
	signMetadataDescription     = "Sign a staged release metadata.json file using a GCP KMS key"
	signMetadataLongDescription = `The metadata command signs a release metadata.json file using a GCP KMS
key, writing a detached signature to metadata.json.sig alongside it.

metadata.json is the root of trust for the publish step: it lists the release
artifacts and their checksums. It is read from the same staging bucket that the
artifacts are written to, so on its own it provides no guarantee of who produced
it. Signing it with a KMS key that only the staging pipeline can use, and
verifying that signature at publish time, binds a published release to metadata
that was genuinely produced by the pipeline - regardless of who can write to the
bucket.

This command is intended to be run as part of staging a release, immediately
before metadata.json (and its signature) are uploaded to the staging bucket.`
)

var signMetadataExample = fmt.Sprintf(`To sign a metadata file at "/tmp/metadata.json":

%s %s %s --key "projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>" --path /tmp/metadata.json`, rootCommand, signCommand, signMetadataCommand)

type signMetadataOptions struct {
	// Key is the full name of the GCP KMS key to be used for signing, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	Key string

	// Path is the path to the metadata.json file to sign.
	Path string
}

func (o *signMetadataOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Key, "key", "", "Full name of the GCP KMS key to use for signing")
	fs.StringVar(&o.Path, "path", "", "Path to the metadata.json file to sign")
	markRequired("key")
	markRequired("path")
}

func (o *signMetadataOptions) print() {
	log.Printf("sign metadata options:")
	log.Printf("   Key: %q", o.Key)
	log.Printf("  Path: %q", o.Path)
}

func signMetadataCmd(rootOpts *rootOptions) *cobra.Command {
	o := &signMetadataOptions{}
	cmd := &cobra.Command{
		Use:          signMetadataCommand,
		Short:        signMetadataDescription,
		Long:         signMetadataLongDescription,
		Example:      signMetadataExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSignMetadata(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func runSignMetadata(rootOpts *rootOptions, o *signMetadataOptions) error {
	ctx := context.Background()

	parsedKey, err := sign.NewGCPKMSKey(o.Key)
	if err != nil {
		return err
	}

	metadata, err := os.ReadFile(o.Path)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %q: %w", o.Path, err)
	}

	signature, err := sign.SignMetadata(ctx, parsedKey, metadata)
	if err != nil {
		return fmt.Errorf("failed to sign metadata file %q: %w", o.Path, err)
	}

	// The signature is base64-encoded so that it is safe to move around as a
	// text file (e.g. through make, gsutil) without any risk of byte mangling.
	encoded := base64.StdEncoding.EncodeToString(signature)

	sigPath := o.Path + release.MetadataSignatureExtension
	if err := os.WriteFile(sigPath, []byte(encoded), 0o644); err != nil {
		return fmt.Errorf("failed to write signature file %q: %w", sigPath, err)
	}

	log.Printf("wrote metadata signature successfully to %q", sigPath)

	return nil
}
