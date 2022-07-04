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

package testgen

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TestContext holds contextual information used to configure a test, such as a mapping
// of which branch names map to which TestGrid dashboard names
type TestContext struct {
	Branches []string

	// PresubmitDashboardName is the name of the TestGrid dashboard to which presubmit tests should
	// be added. If unset, no TestGrid annotations will be added to presubmit tests.
	PresubmitDashboardName string

	// PeriodicDashboardName is the name of the TestGrid dashboard to which periodic tests should
	// be added. If unset, no TestGrid annotations will be added to periodic tests.
	PeriodicDashboardName string

	Org  string
	Repo string

	// Descriptor is a string which, if set, is inserted into periodic test names.
	// An example would be "previous" which would result in "my-test" having the name
	// "ci-<repo>-previous-my-test", where the test would've been called
	// "ci-<repo>-my-test" if the descriptor was not set
	Descriptor string

	presubmits []*PresubmitTest
	periodics  []*PeriodicTest

	minutesCounter time.Time
}

// RequiredPresubmit adds a presubmit which is run by default and required to pass before a PR can be merged
func (tc *TestContext) RequiredPresubmit(test *Test) {
	tc.addPresubmit(test, true, false)
}

// RequiredPresubmit adds a presubmit which is not run by default and is optional
func (tc *TestContext) OptionalPresubmit(test *Test) {
	tc.addPresubmit(test, false, true)
}

func (tc *TestContext) addPresubmit(test *Test, alwaysRun bool, optional bool) {
	test.Name = tc.presubmitTestName(test.Name)

	if tc.PresubmitDashboardName != "" {
		addTestGridAnnotations(tc.PresubmitDashboardName)(test)
	}

	tc.presubmits = append(tc.presubmits, &PresubmitTest{
		Test:      *test,
		Branches:  tc.Branches,
		AlwaysRun: alwaysRun,
		Optional:  optional,
	})
}

// Periodic adds periodic tests which will run every `periodicityHours` hours, at some minute
// within the hour, one test for each configured branch
func (tc *TestContext) Periodics(test *Test, periodicityHours int) {
	for _, branch := range tc.Branches {
		test.Name = tc.periodicTestName(test.Name)

		if tc.PeriodicDashboardName != "" {
			addTestGridAnnotations(tc.PeriodicDashboardName)(test)
		}

		tc.periodics = append(tc.periodics, &PeriodicTest{
			Test: *test,
			ExtraRefs: []ExtraRef{
				{
					Org:     tc.Org,
					Repo:    tc.Repo,
					BaseRef: branch,
				},
			},
			Interval: strconv.Itoa(periodicityHours) + "h",
			// TODO: use Cron instead of Interval
			// Cron: tc.cronSchedule(periodicityHours),
		})
	}
}

func (tc *TestContext) TestFile() *TestFile {
	// TODO: when using Cron instead of Interval for periodics, adjust all periodics
	// here to spread them evenly throughout the hour

	presubmitKey := fmt.Sprintf("%s/%s", tc.Org, tc.Repo)

	return &TestFile{
		Presubmits: map[string][]*PresubmitTest{
			presubmitKey: tc.presubmits,
		},
		Periodics: tc.periodics,
	}
}

func (tc *TestContext) presubmitTestName(name string) string {
	return fmt.Sprintf("pull-%s-%s", tc.Repo, name)
}

func (tc *TestContext) periodicTestName(name string) string {
	return fmt.Sprintf("ci-%s-%s%s", tc.Repo, tc.printableDescriptor(), name)
}

func (tc *TestContext) printableDescriptor() string {
	if tc.Descriptor == "" {
		return tc.Descriptor
	}

	return strings.Trim(tc.Descriptor, "-") + "-"
}

func (tc *TestContext) cronSchedule(periodicityHours int) string {
	minute := tc.minutesValue()

	return fmt.Sprintf("*/%d %d * * *", minute, periodicityHours)
}

// minutesValue returns a minute value (0 - 59) at which a test should be run and then
// increases the next value returned. This helps to prevent every test running at the same
// minute within the hour causing a spiky distribution of tests.
func (tc *TestContext) minutesValue() int {
	minuteVal := tc.minutesCounter.Minute()

	tc.minutesCounter = tc.minutesCounter.Add(4 * time.Minute)

	return minuteVal
}
