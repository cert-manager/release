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

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	"github.com/cert-manager/release/pkg/testgen"
)

const (
	generateTestsCommand         = "generate-tests"
	generateTestsDescription     = "Generate YAML specifying tests for cert-manager"
	generateTestsLongDescription = `generate-tests creates test specifications for a given cert-manager release 'channel', which
define the Prow tests available to be run for that 'channel'.

That includes both presubmit tests (tests that can be run against PRs) and periodic
tests (tests which are run regularly).

By generating this YAML we avoid the need for humans to edit YAML manually
which is error-prone.`
)

var (
	generateTestsExample = fmt.Sprintf(`
To generate tests for the previous release:

	%s %s --mode=previous
`, rootCommand, generateTestsCommand)
)

type generateTestsOptions struct {
	// Mode specifies which type of tests to generate, e.g. "previous" or "master"
	Mode string
}

func (o *generateTestsOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Mode, "mode", "", fmt.Sprintf("Type of tests to generate; one of %s", validModes()))

	markRequired("mode")
}

func generateTestsCmd(rootOpts *rootOptions) *cobra.Command {
	o := &generateTestsOptions{}

	cmd := &cobra.Command{
		Use:          generateTestsCommand,
		Short:        generateTestsDescription,
		Long:         generateTestsLongDescription,
		Example:      generateTestsExample,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateTests(rootOpts, o)
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

func runGenerateTests(rootOpts *rootOptions, o *generateTestsOptions) error {
	spec, ok := modes[strings.ToLower(o.Mode)]
	if !ok {
		return fmt.Errorf("unknown mode %q; valid modes are %s", o.Mode, validModes())
	}

	testFile := spec.GenerateTestFile()

	out, err := yaml.Marshal(testFile)
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

type modeSpec struct {
	testContext *testgen.TestContext

	primaryKubernetesVersion string
	otherKubernetesVersions  []string
}

// GenerateTestFile will create a complete test file based on the modeSpec `m`. This
// function assumes that all tests for all of `previous`, `current` and `next` should
// be broadly the same.
func (m *modeSpec) GenerateTestFile() *testgen.TestFile {
	m.testContext.RequiredPresubmit(testgen.MakeTest())
	m.testContext.RequiredPresubmit(testgen.ChartTest())

	for _, secondaryVersion := range m.otherKubernetesVersions {
		m.testContext.OptionalPresubmit(testgen.E2ETest(secondaryVersion))
	}

	m.testContext.RequiredPresubmit(testgen.E2ETest(m.primaryKubernetesVersion))
	m.testContext.RequiredPresubmit(testgen.UpgradeTest(m.primaryKubernetesVersion))

	m.testContext.OptionalPresubmit(testgen.E2ETestVenafiTPP(m.primaryKubernetesVersion))
	m.testContext.OptionalPresubmit(testgen.E2ETestVenafiCloud(m.primaryKubernetesVersion))
	m.testContext.OptionalPresubmit(testgen.E2ETestFeatureGatesDisabled(m.primaryKubernetesVersion))

	allKubernetesVersions := append(m.otherKubernetesVersions, m.primaryKubernetesVersion)

	m.testContext.Periodics(testgen.MakeTest(), 2)

	// TODO: add chart periodic test?

	for _, kubernetesVersion := range allKubernetesVersions {
		m.testContext.Periodics(testgen.E2ETest(kubernetesVersion), 2)

	}

	m.testContext.Periodics(testgen.E2ETestVenafiBoth(m.primaryKubernetesVersion), 12)
	m.testContext.Periodics(testgen.UpgradeTest(m.primaryKubernetesVersion), 8)

	// TODO: roll this into above for loop; we have two for loops here to preserve the
	// ordering of the tests in the output file, making it easier to review the
	// differences between generated tests and existing handwritten tests

	for _, kubernetesVersion := range allKubernetesVersions {
		m.testContext.Periodics(testgen.E2ETestFeatureGatesDisabled(kubernetesVersion), 24)
	}

	return m.testContext.TestFile()
}

var modes map[string]modeSpec = map[string]modeSpec{
	"previous": modeSpec{
		testContext: &testgen.TestContext{
			Branches: []string{"release-1.8"},

			PresubmitDashboardName: "",
			PeriodicDashboardName:  "jetstack-cert-manager-previous",

			Org:  "cert-manager",
			Repo: "cert-manager",

			Descriptor: "previous",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.19", "1.20", "1.21", "1.22", "1.23"},
	},
	"current": modeSpec{
		testContext: &testgen.TestContext{
			Branches: []string{"master"},

			PresubmitDashboardName: "jetstack-cert-manager-presubmits-blocking",
			PeriodicDashboardName:  "jetstack-cert-manager-master",

			Org:  "cert-manager",
			Repo: "cert-manager",

			// descriptor is what's added to periodic test names; for master we don't add anything
			// so we have "ci-cert-manager-e2e-v1-20"
			// and not "ci-cert-manager-current-e2e-v1-20"
			Descriptor: "",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23"},
	},
	"next": modeSpec{
		testContext: &testgen.TestContext{
			Branches: []string{"release-1.9"},

			PresubmitDashboardName: "",
			PeriodicDashboardName:  "jetstack-cert-manager-next",

			Org:  "cert-manager",
			Repo: "cert-manager",

			Descriptor: "next",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23"},
	},
}

func validModes() string {
	var availableModes []string

	for mode, _ := range modes {
		availableModes = append(availableModes, mode)
	}

	return strings.Join(availableModes, ", ")
}
