/*
Copyright 2022 The cert-manager Authors.

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

// Note for developers:
// If you want to edit how tests are generated, change: ./pkg/prowgen/
// If you want to edit which tests are generated on each branch / k8s version, change: ./prowspecs/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	"github.com/cert-manager/release/prowspecs"
)

const (
	generateProwCommand         = "generate-prow"
	generateProwDescription     = "Generate YAML specifying prow tests for cert-manager"
	generateProwLongDescription = `generate-prow creates prow test specifications for a given cert-manager release branch. These specifications
define the Prow tests available to be run against a given branch.

Generated tests include both presubmit tests (tests that can be run against PRs) and periodic
tests (tests which are run on a schedule, independently of PRs).

By generating this config we avoid the need for humans to edit YAML manually
which is error-prone.`
)

var (
	generateProwExample = fmt.Sprintf(`
To generate tests for the a branch called foo:

	%s %s --branch=foo
`, rootCommand, generateProwCommand)
)

type generateProwOptions struct {
	// Branch specifies the name of the branch whose tests should be generated
	Branch string
}

func (o *generateProwOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Branch, "branch", "", fmt.Sprintf("Type of tests to generate; one of %s", prowspecs.KnownBranches()))

	markRequired("branch")
}

func generateProwCmd(rootOpts *rootOptions) *cobra.Command {
	o := &generateProwOptions{}

	cmd := &cobra.Command{
		Use:          generateProwCommand,
		Short:        generateProwDescription,
		Long:         generateProwLongDescription,
		Example:      generateProwExample,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateProw(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

// sanitizedArgs strips the path from the command which was used to invoke the script,
// so we don't include things like "/home/workspace/release/bin/cmrel"
func sanitizedArgs() []string {
	args := os.Args[:]
	args[0] = filepath.Base(args[0])

	return args
}

func runGenerateProw(rootOpts *rootOptions, o *generateProwOptions) error {
	spec, err := prowspecs.SpecForBranch(o.Branch)
	if err != nil {
		return err
	}

	jobFile := spec.GenerateJobFile()

	out, err := yaml.Marshal(jobFile)
	if err != nil {
		return err
	}

	prelude := fmt.Sprintf(
		`# THIS FILE HAS BEEN AUTOMATICALLY GENERATED
# Don't manually edit it; instead edit the "cmrel" tool which generated it
# Generated with: %s

`,
		strings.Join(sanitizedArgs(), " "),
	)

	fmt.Println(prelude + string(out))

	return nil
}
