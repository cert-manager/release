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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/cert-manager/release/pkg/sign"
)

const (
	signHelmCommand         = "helm"
	signHelmDescription     = "Manually sign a helm chart using a GCP KMS key"
	signHelmLongDescription = `The helm command signs a helm chart using a GCP KMS key rather than
a local PGP key.

Mostly, this command is provided in case of an issue with the signing process
elsewhere. cmrel should attempt to create signatures as part of the normal
release process, and it shouldn't be neccessary to create a manual signature.

Using this shim rather than "helm package --sign" allows us to use exactly one
signing key for all kinds of cert-manager artifact, with no member of the team
having access to the actual private key.`
)

var signHelmExample = fmt.Sprintf(`To sign a chart called "mychart.tgz":

%s %s %s --key "projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>" --chartpath mychart.tgz`, rootCommand, signCommand, signHelmCommand)

type signHelmOptions struct {
	// Key is the full name of the GCP KMS key to be used, e.g.
	// projects/<PROJECT_NAME>/locations/<LOCATION>/keyRings/<KEYRING_NAME>/cryptoKeys/<KEY_NAME>/cryptoKeyVersions/<KEY_VERSION>
	Key string

	// ChartPath is the path to the directory for the chart to sign
	ChartPath string
}

func (o *signHelmOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Key, "key", "", "Full name of the GCP KMS key to use for signing")
	fs.StringVar(&o.ChartPath, "chart-path", "", "Path to the directory of the helm chart to sign, similar to what would be passed into 'helm package'")
	markRequired("key")
	markRequired("chart-path")
}

func (o *signHelmOptions) print() {
	log.Printf("sign helm options:")
	log.Printf("        Key: %q", o.Key)
	log.Printf("  ChartPath: %q", o.ChartPath)
}

func signHelmCmd(rootOpts *rootOptions) *cobra.Command {
	o := &signHelmOptions{}
	cmd := &cobra.Command{
		Use:          signHelmCommand,
		Short:        signHelmDescription,
		Long:         signHelmLongDescription,
		Example:      signHelmExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSignHelm(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func runSignHelm(rootOpts *rootOptions, o *signHelmOptions) error {
	ctx := context.Background()

	parsedKey, err := sign.NewGCPKMSKey(o.Key)
	if err != nil {
		return err
	}

	signatureBytes, err := sign.HelmChart(ctx, parsedKey, o.ChartPath)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	provFile := filepath.Base(o.ChartPath) + ".prov"

	err = os.WriteFile(provFile, signatureBytes, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write %q: %w", provFile, err)
	}

	log.Printf("wrote signature successfully to %q", provFile)

	return nil
}
