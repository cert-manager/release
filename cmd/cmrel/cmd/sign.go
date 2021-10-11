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
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

const (
	signCommand         = "sign"
	signDescription     = "Subcommands for signing artifacts manually"
	signDescriptionLong = `sign contains commands for signing cert-manager artifacts manually.

Ideally, these commands should never need to be run manually; signing should
be done by GCB builds automatically as part of staging a release.

These commands are provided as an escape-hatch for when things go wrong.`
)

type signOptions struct {
}

func (o *signOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
}

func (o *signOptions) print() {
}

func signCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   signCommand,
		Short: signDescription,
		Long:  signDescriptionLong,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			o.print()
		},
	}

	o.AddFlags(cmd.PersistentFlags(), mustMarkRequired(cmd.MarkPersistentFlagRequired))

	cmd.AddCommand(signHelmCmd(o))
	cmd.AddCommand(signManifestsCmd(o))

	return cmd
}
