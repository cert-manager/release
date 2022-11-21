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

// knownBranches specifies a BranchSpec for each possible branch to test against
// THIS IS WHAT YOU'RE MOST LIKELY TO NEED TO EDIT
// The branches and kubernetes versions below are likely to need to be updated after each cert-manager release!

// NB: There's at least one configurer (pkg/prowgen/configurers.go) which will changes its operations
// based on the k8s version it's being run against.

var knownBranches map[string]BranchSpec = map[string]BranchSpec{
	"release-1.9": {
		prowContext: &prowgen.ProwContext{
			Branch: "release-1.9",

			// Freeze test images used.
			Image: "eu.gcr.io/jetstack-build-infra-images/bazelbuild:20220512-b6ea825-4.2.1",

			// NB: we don't use a presubmit dashboard outside of "master", currently
			PresubmitDashboard: false,
			PeriodicDashboard:  true,

			Org:  "cert-manager",
			Repo: "cert-manager",
		},

		primaryKubernetesVersion: "1.24",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23"},

		skipTrivy: true,
	},
	"release-1.10": {
		prowContext: &prowgen.ProwContext{
			Branch: "release-1.10",

			// Freeze test images used.
			Image: "eu.gcr.io/jetstack-build-infra-images/bazelbuild:20220830-c65cd19-4.2.1",

			// NB: we don't use a presubmit dashboard outside of "master", currently
			PresubmitDashboard: false,
			PeriodicDashboard:  true,

			Org:  "cert-manager",
			Repo: "cert-manager",
		},

		primaryKubernetesVersion: "1.25",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23", "1.24"},

		skipTrivy: false,
	},
	"master": {
		prowContext: &prowgen.ProwContext{
			Branch: "master",

			// Use latest image.
			Image: prowgen.CommonTestImage,

			PresubmitDashboard: true,
			PeriodicDashboard:  true,

			Org:  "cert-manager",
			Repo: "cert-manager",
		},

		primaryKubernetesVersion: "1.25",
		otherKubernetesVersions:  []string{"1.20", "1.21", "1.22", "1.23", "1.24"},

		skipTrivy: false,
	},
}

// BranchSpec holds a specification of an entire test suite for a given branch, such as "master" or "release-1.9"
// That includes:
// - a ProwContext specifying things like the the repo, branch, dashboard names
// - the primary Kubernetes version (which is the version whose tests are always run for presubmits, among other uses)
// - the secondary Kubernetes versions, which are the rest of the supported versions for which tests should be generated
type BranchSpec struct {
	prowContext *prowgen.ProwContext

	primaryKubernetesVersion string
	otherKubernetesVersions  []string

	// skipUpgradeTest if set will cause the upgrade test to not be added to periodics or presubmits.
	// This is because the test is manually specified using bazel for release-1.8, and the test isn't implemented
	// in make. Efforts to backport things like tests have proven difficult, so let's make the change here
	// rather than trying to backport the upgrade test.
	skipUpgradeTest bool

	// skipTrivy skips generating tests relating to vulnerability scanning since this wasn't backported.
	skipTrivy bool
}

// GenerateJobFile will create a complete test file based on the BranchSpec. This
// assumes that all tests for all branches should be the same.
func (m *BranchSpec) GenerateJobFile() *prowgen.JobFile {
	m.prowContext.RequiredPresubmit(prowgen.MakeTest(m.prowContext))
	m.prowContext.RequiredPresubmit(prowgen.ChartTest(m.prowContext))

	for _, secondaryVersion := range m.otherKubernetesVersions {
		m.prowContext.OptionalPresubmit(prowgen.E2ETest(m.prowContext, secondaryVersion))
	}

	m.prowContext.RequiredPresubmit(prowgen.E2ETest(m.prowContext, m.primaryKubernetesVersion))

	if !m.skipUpgradeTest {
		// TODO: 1.8 is the last version which doesn't support make-based upgrade tests. This can be
		// done unconditionally when 1.8 is deprecated.
		m.prowContext.RequiredPresubmit(prowgen.UpgradeTest(m.prowContext, m.primaryKubernetesVersion))
	}

	m.prowContext.OptionalPresubmit(prowgen.E2ETestVenafiTPP(m.prowContext, m.primaryKubernetesVersion))
	m.prowContext.OptionalPresubmit(prowgen.E2ETestVenafiCloud(m.prowContext, m.primaryKubernetesVersion))
	m.prowContext.OptionalPresubmit(prowgen.E2ETestFeatureGatesDisabled(m.prowContext, m.primaryKubernetesVersion))

	allKubernetesVersions := append(m.otherKubernetesVersions, m.primaryKubernetesVersion)

	m.prowContext.Periodics(prowgen.MakeTest(m.prowContext), 2)

	// TODO: add chart periodic test?

	for _, kubernetesVersion := range allKubernetesVersions {
		m.prowContext.Periodics(prowgen.E2ETest(m.prowContext, kubernetesVersion), 2)

	}

	m.prowContext.Periodics(prowgen.E2ETestVenafiBoth(m.prowContext, m.primaryKubernetesVersion), 12)

	if !m.skipUpgradeTest {
		// TODO: 1.8 is the last version which doesn't support make-based upgrade tests. This can be
		// done unconditionally when 1.8 is deprecated.
		m.prowContext.Periodics(prowgen.UpgradeTest(m.prowContext, m.primaryKubernetesVersion), 8)
	}

	for _, kubernetesVersion := range allKubernetesVersions {
		// TODO: roll this into above for loop; we have two for loops here to preserve the
		// ordering of the tests in the output file, making it easier to review the
		// differences between generated tests and existing handwritten tests
		m.prowContext.Periodics(prowgen.E2ETestFeatureGatesDisabled(m.prowContext, kubernetesVersion), 24)
	}

	if !m.skipTrivy {
		for _, container := range []string{"controller", "acmesolver", "ctl", "cainjector", "webhook"} {
			m.prowContext.Periodics(prowgen.TrivyTest(m.prowContext, container), 24)
		}
	}

	return m.prowContext.JobFile()
}

// KnownBranches returns a list of all branches which have been configured here
func KnownBranches() []string {
	var availableBranches []string

	for branch, _ := range knownBranches {
		availableBranches = append(availableBranches, branch)
	}

	return availableBranches
}

// SpecForBranch returns a spec for the named branch, if it exists
func SpecForBranch(originalBranch string) (BranchSpec, error) {
	branch := strings.ToLower(originalBranch)

	spec, ok := knownBranches[branch]
	if !ok {
		return BranchSpec{}, fmt.Errorf("unknown branch %q; known branches are %q", originalBranch, KnownBranches())
	}

	return spec, nil
}
