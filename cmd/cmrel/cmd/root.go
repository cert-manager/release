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
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

const (
	rootCommand         = "cmrel"
	rootDescription     = "cert-manager release management tool"
	rootDescriptionLong = `Use to prepare, build and publish cert-manager release artifacts.`
)

type rootOptions struct {
	// Debug configures whether output from subcommands should be directly
	// piped to stderr of the process.
	Debug bool
}

func (o *rootOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.BoolVar(&o.Debug, "debug", false, "If true, output from sub-commands will be directly piped to stderr.")
}

func (o *rootOptions) print() {
	log.Printf("Root options:")
	log.Printf("  Debug: %t", o.Debug)
}

func rootCmd(o *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootCommand,
		Short: rootDescription,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			o.print()
		},
		Long: rootDescriptionLong,
	}
	o.AddFlags(cmd.PersistentFlags(), mustMarkRequired(cmd.MarkPersistentFlagRequired))
	return cmd
}

// mustMarkRequired will return a func(string) that can be used to mark a flag
// as required.
// If the given MarkRequired func returns an error, it will print the error
// and call os.Exit(1).
func mustMarkRequired(markRequired func(string) error) func(string) {
	return func(s string) {
		if err := markRequired(s); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func Execute() {
	o := &rootOptions{}
	cmd := rootCmd(o)
	cmd.AddCommand(stagedCmd(o))
	cmd.AddCommand(stageCmd(o))
	cmd.AddCommand(gcbCmd(o))
	cmd.AddCommand(publishCmd(o))
	cmd.AddCommand(bootstrapPGPCmd(o))
	cmd.AddCommand(signCmd(o))
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
