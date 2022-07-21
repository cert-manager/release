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

package prowspecs

import (
	"fmt"
	"strings"

	"github.com/cert-manager/release/pkg/prowgen"
)

// modes specifies a ModeSpec for each possible mode
// THIS IS WHAT YOU'RE MOST LIKELY TO NEED TO EDIT
// The branches and kubernetes versions below are likely to need to be updated after each
// cert-manager release!

// NB: There's at least one configurer (pkg/prowgen/configurers.go) which will changes its operations
// based on the k8s version it's being run against.
var modes map[string]ModeSpec = map[string]ModeSpec{
	"previous": ModeSpec{
		prowContext: &prowgen.ProwContext{
			Branches: []string{"release-1.8"},

			// NB: we don't use a presubmit dashboard on previous currently
			PresubmitDashboardName: "",
			PeriodicDashboardName:  "jetstack-cert-manager-previous",

			Org:  "cert-manager",
			Repo: "cert-manager",

			// Descriptor inserts "previous" into periodic names
			Descriptor: "previous",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.19", "1.20", "1.21", "1.22", "1.23"},
	},
	"current": ModeSpec{
		prowContext: &prowgen.ProwContext{
			Branches: []string{"master"},

			PresubmitDashboardName: "jetstack-cert-manager-presubmits-blocking",
			PeriodicDashboardName:  "jetstack-cert-manager-master",

			Org:  "cert-manager",
			Repo: "cert-manager",

			// Descriptor is what's added to periodic test names; for master we don't add anything
			// so we have "ci-cert-manager-e2e-v1-20" and not "ci-cert-manager-current-e2e-v1-20"
			Descriptor: "",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23"},
	},
	"next": ModeSpec{
		prowContext: &prowgen.ProwContext{
			Branches: []string{"release-1.9"},

			// NB: we don't use a presubmit dashboard on next currently
			PresubmitDashboardName: "",
			PeriodicDashboardName:  "jetstack-cert-manager-next",

			Org:  "cert-manager",
			Repo: "cert-manager",

			// Descriptor inserts "next" into periodic names
			Descriptor: "next",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23"},
	},
}

// ModeSpec holds a specification of an entire test suite for a given mode or "channel", such as "previous",
// "current" or "next". That includes:
// - a ProwContext specifying things like the repo, branch(es), dashboard names
// - the primary Kubernetes version (which is the version whose tests are always run for presubmits, among other uses)
// - the secondary Kubernetes versions, which are the rest of the supported versions for which tests should be generated
type ModeSpec struct {
	prowContext *prowgen.ProwContext

	primaryKubernetesVersion string
	otherKubernetesVersions  []string
}

// GenerateJobFile will create a complete test file based on the ModeSpec `m`. This
// assumes that all tests for all of `previous`, `current` and `next` should be the same.
func (m *ModeSpec) GenerateJobFile() *prowgen.JobFile {
	m.prowContext.RequiredPresubmit(prowgen.MakeTest())
	m.prowContext.RequiredPresubmit(prowgen.ChartTest())

	for _, secondaryVersion := range m.otherKubernetesVersions {
		m.prowContext.OptionalPresubmit(prowgen.E2ETest(secondaryVersion))
	}

	m.prowContext.RequiredPresubmit(prowgen.E2ETest(m.primaryKubernetesVersion))
	m.prowContext.RequiredPresubmit(prowgen.UpgradeTest(m.primaryKubernetesVersion))

	m.prowContext.OptionalPresubmit(prowgen.E2ETestVenafiTPP(m.primaryKubernetesVersion))
	m.prowContext.OptionalPresubmit(prowgen.E2ETestVenafiCloud(m.primaryKubernetesVersion))
	m.prowContext.OptionalPresubmit(prowgen.E2ETestFeatureGatesDisabled(m.primaryKubernetesVersion))

	allKubernetesVersions := append(m.otherKubernetesVersions, m.primaryKubernetesVersion)

	m.prowContext.Periodics(prowgen.MakeTest(), 2)

	// TODO: add chart periodic test?

	for _, kubernetesVersion := range allKubernetesVersions {
		m.prowContext.Periodics(prowgen.E2ETest(kubernetesVersion), 2)

	}

	m.prowContext.Periodics(prowgen.E2ETestVenafiBoth(m.primaryKubernetesVersion), 12)
	m.prowContext.Periodics(prowgen.UpgradeTest(m.primaryKubernetesVersion), 8)

	// TODO: roll this into above for loop; we have two for loops here to preserve the
	// ordering of the tests in the output file, making it easier to review the
	// differences between generated tests and existing handwritten tests

	for _, kubernetesVersion := range allKubernetesVersions {
		m.prowContext.Periodics(prowgen.E2ETestFeatureGatesDisabled(kubernetesVersion), 24)
	}

	return m.prowContext.JobFile()
}

// ValidModes returns a string containing the names of each recognised valid mode
func ValidModes() string {
	var availableModes []string

	for mode, _ := range modes {
		availableModes = append(availableModes, mode)
	}

	return strings.Join(availableModes, ", ")
}

// SpecForMode returns a spec for the named mode, if it exists
func SpecForMode(originalMode string) (ModeSpec, error) {
	mode := strings.ToLower(originalMode)

	spec, ok := modes[mode]
	if !ok {
		return ModeSpec{}, fmt.Errorf("unknown mode %q; valid modes are %q", originalMode, ValidModes())
	}

	return spec, nil
}
