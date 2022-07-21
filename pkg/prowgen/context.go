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

package prowgen

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ProwContext holds jobs and information required to configure jobs for a given release channel.
type ProwContext struct {
	// Branches is the list of branches for which a given test should be added. The same test will be added
	// for every branch.
	Branches []string

	// PresubmitDashboardName is the name of the TestGrid dashboard to which presubmit jobs should
	// be added. If unset, no TestGrid annotations will be added to presubmit jobs.
	PresubmitDashboardName string

	// PeriodicDashboardName is the name of the TestGrid dashboard to which periodic jobs should
	// be added. If unset, no TestGrid annotations will be added to periodic jobs.
	PeriodicDashboardName string

	// Org is the GitHub organisation of the repository under test.
	Org string

	// Repo is the GitHub repository name of the repository under test.
	Repo string

	// Descriptor is a string which, if set, is inserted into periodic test names.
	// An example would be "previous" which would result in "my-test" having the name
	// "ci-<repo>-previous-my-test", where the test would've been called
	// "ci-<repo>-my-test" if the descriptor was not set
	Descriptor string

	presubmits []*PresubmitJob
	periodics  []*PeriodicJob

	minutesCounter time.Time
}

// RequiredPresubmit adds a presubmit which is run by default and required to pass before a PR can be merged
func (pc *ProwContext) RequiredPresubmit(job *Job) {
	pc.addPresubmit(job, true, false)
}

// RequiredPresubmits adds a list of jobs to the context
func (pc *ProwContext) RequiredPresubmits(jobs []*Job) {
	for _, job := range jobs {
		pc.addPresubmit(job, true, false)
	}
}

// RequiredPresubmit adds a presubmit which is not run by default and is optional
func (pc *ProwContext) OptionalPresubmit(job *Job) {
	pc.addPresubmit(job, false, true)
}

func (pc *ProwContext) addPresubmit(job *Job, alwaysRun bool, optional bool) {
	job.Name = pc.presubmitJobName(job.Name)

	if pc.PresubmitDashboardName != "" {
		addTestGridAnnotations(pc.PresubmitDashboardName)(job)
	}

	pc.presubmits = append(pc.presubmits, &PresubmitJob{
		Job:       *job,
		Branches:  pc.Branches,
		AlwaysRun: alwaysRun,
		Optional:  optional,
	})
}

// Periodic adds periodic jobs which will run every `periodicityHours` hours, at some minute
// within the hour, one job for each configured branch
func (pc *ProwContext) Periodics(job *Job, periodicityHours int) {
	for _, branch := range pc.Branches {
		job.Name = pc.periodicJobName(job.Name)

		if pc.PeriodicDashboardName != "" {
			addTestGridAnnotations(pc.PeriodicDashboardName)(job)
		}

		pc.periodics = append(pc.periodics, &PeriodicJob{
			Job: *job,
			ExtraRefs: []ExtraRef{
				{
					Org:     pc.Org,
					Repo:    pc.Repo,
					BaseRef: branch,
				},
			},
			Interval: strconv.Itoa(periodicityHours) + "h",
			// TODO: use Cron instead of Interval
			// Cron: pc.cronSchedule(periodicityHours),
		})
	}
}

func (pc *ProwContext) JobFile() *JobFile {
	// TODO: when using Cron instead of Interval for periodics, adjust all periodics
	// here to spread them evenly throughout the hour

	presubmitKey := fmt.Sprintf("%s/%s", pc.Org, pc.Repo)

	return &JobFile{
		Presubmits: map[string][]*PresubmitJob{
			presubmitKey: pc.presubmits,
		},
		Periodics: pc.periodics,
	}
}

func (pc *ProwContext) presubmitJobName(name string) string {
	return fmt.Sprintf("pull-%s-%s", pc.Repo, name)
}

func (pc *ProwContext) periodicJobName(name string) string {
	return fmt.Sprintf("ci-%s-%s%s", pc.Repo, pc.printableDescriptor(), name)
}

func (pc *ProwContext) printableDescriptor() string {
	if pc.Descriptor == "" {
		return pc.Descriptor
	}

	return strings.Trim(pc.Descriptor, "-") + "-"
}

func (pc *ProwContext) cronSchedule(periodicityHours int) string {
	minute := pc.minutesValue()

	return fmt.Sprintf("*/%d %d * * *", minute, periodicityHours)
}

// minutesValue returns a minute value (0 - 59) at which a test should be run and then
// increases the next value returned. This helps to prevent every test running at the same
// minute within the hour causing a spiky distribution of tests.
func (pc *ProwContext) minutesValue() int {
	minuteVal := pc.minutesCounter.Minute()

	pc.minutesCounter = pc.minutesCounter.Add(4 * time.Minute)

	return minuteVal
}
