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
	gcbCommand         = "gcb"
	gcbDescription     = "Subcommands usually run within Google Cloud Build jobs"
	gcbDescriptionLong = ``
)

type gcbOptions struct {
}

func (o *gcbOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
}

func (o *gcbOptions) print() {
}

func gcbCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   gcbCommand,
		Short: gcbDescription,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			o.print()
		},
		Long: gcbDescriptionLong,
	}

	o.AddFlags(cmd.PersistentFlags(), mustMarkRequired(cmd.MarkPersistentFlagRequired))

	cmd.AddCommand(gcbStageCmd(o))
	cmd.AddCommand(gcbPublishCmd(o))
	cmd.AddCommand(gcbBootstrapPGPCmd(o))

	return cmd
}
