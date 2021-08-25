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
	"sort"
	"text/tabwriter"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/mod/semver"

	"github.com/cert-manager/release/pkg/release"
)

const (
	stagedCommand     = "staged"
	stagedDescription = "List existing staged releases in the GCS bucket, sorted by version."
)

var (
	stagedExample = fmt.Sprint(`
Imagine that you just ran 'cmrel stage', and you now want to run 'cmrel publish',
which requires you to know the "release name" (--release-name).

    v1.0.0-alpha.1-ae6a747fd4495a24db00ce4c1522c6eac72bc5a4

The "staged" command will help you find this release name. To list the existing
staged releases, run:

    cmrel staged

The output is sorted lexicographically using the version string:

    NAME                                                      VERSION
    v1.0.2-219b7934ac499c7818526597cf635a922bddd22e           v1.0.2
    v1.0.3-cbd52ed6e9c296012bab87d3877d31e1f1295fa5           v1.0.3
    v1.0.4-4d870e49b43960fad974487a262395e65da1373e           v1.0.4
    v1.1.0-7fbdd6487646e812fe74c0c05503805b5d9d4751           v1.1.0
    v1.1.0-alpha.0-09f043d2c96da68ed8d4f2c71a868fe0846d3669   v1.1.0-alpha.0
    v1.1.0-alpha.1-fda1c091e3f37046c378bbf832e603284b6db531   v1.1.0-alpha.1
    v1.1.1-3ac7418070e22c87fae4b22603a6b952f797ae96           v1.1.1
    v1.2.0-969b678f330c68a6429b7a71b271761c59651a85           v1.2.0
    v1.2.0-alpha.0-7cef4582ec8e33ff2f3b8dcf15b3f293f6ef82cc   v1.2.0-alpha.0
    v1.2.0-alpha.1-33f18811909bdd08d39fd8aa3f016734d1393d18   v1.2.0-alpha.1
    v1.2.0-alpha.2-35febb171706826f27d71af466c624c25733c135   v1.2.0-alpha.2
    v1.3.0-9c42eeebfd3978531b517277a21e28e3cf90b876           v1.3.0
    v1.3.0-alpha.0-77b045d159bd20ce0ec454cd79a5edce9187bdd9   v1.3.0-alpha.0
    v1.3.0-alpha.1-c2c0fdd78131493707050ffa4a7454885d041b08   v1.3.0-alpha.1
    v1.3.0-beta.0-9f612f0c2eee8390fb730b1aafa592b88d768d15    v1.3.0-beta.0
    v1.3.1-614438aed00e1060870b273f2238794ef69b60ab           v1.3.1
    v1.4.0-alpha.1-0ff2b8778c51e6cebe140a6b196e7a9a28cbee87   v1.4.0-alpha.1
    v1.4.0-alpha.0-8d794c6bcf3bb02b9961bbd40f5b821f5636cceb   v1.4.0-wallrj.1
    v1.4.0-wallrj.2-0ff2b8778c51e6cebe140a6b196e7a9a28cbee87  v1.4.0-wallrj.2

If you already know the release version (and since you have run 'cmrel stage',
you probably do), you can select just these versions:

	cmrel staged --release-version=v1.3.1

which will only show the releases that you are interested in:

    NAME                                                      VERSION
    v1.3.1-614438aed00e1060870b273f2238794ef69b60ab           v1.3.1

The "release name" that you need to pass as --release-name to 'cmrel publish'
is the string:

    v1.3.1-614438aed00e1060870b273f2238794ef69b60ab

Note that by default, the command will only show the "release" type, not the
"devel" ones. To see the "devel" staged releases, you need to run:

	cmrel staged --release-type=devel

This time, no version will be shown, just the git commit hash:

    NAME                                     VERSION
    29406bfaa25c33661ff31b4d60a74f7b04ab6f2d
    3c43140e9e7a6fc04e0e7ba0d50faeaa6aea97df
    b95836421f7f3d2bbbebaa4fa3cca7128e3a97ad
    dfafd10391b00d65315624dbbdc840d21735b240
    ece63038d00e62711443a5abbc0e87b15a1367c1
`)
)

type stagedOptions struct {
	// The name of the GCS bucket containing the staged releases
	Bucket string

	// Optional commit ref of cert-manager that should be stagedd
	GitRef string

	// ReleaseVersion, if set, overrides the version git version tag used
	// during the build. This is used to force a build's version number to be
	// the final release tag before a tag has actually been created in the
	// repository.
	ReleaseVersion string

	// The type of release to list - usually one of 'release' or 'devel'
	ReleaseType string
}

func (o *stagedOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket containing the staged releases.")
	fs.StringVar(&o.GitRef, "git-ref", "", "Optional specific git reference to list staged releases for - if specified, --release-version must also be specified.")
	fs.StringVar(&o.ReleaseVersion, "release-version", "", "Optional release version override used to force the version strings used during the release to a specific value.")
	fs.StringVar(&o.ReleaseType, "release-type", "release", "The type of release to list, usually one of 'release' or 'devel'")
}

func (o *stagedOptions) print() {
	log.Printf("Staged options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  GitRef: %q", o.GitRef)
	log.Printf("  ReleaseVersion: %q", o.ReleaseVersion)
	log.Printf("  ReleaseType: %q", o.ReleaseType)
}

func stagedCmd(rootOpts *rootOptions) *cobra.Command {
	o := &stagedOptions{}
	cmd := &cobra.Command{
		Use:          stagedCommand,
		Short:        stagedDescription,
		Example:      stagedExample,
		SilenceUsage: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStaged(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

func runStaged(_ *rootOptions, o *stagedOptions) error {
	if o.ReleaseVersion == "" && o.GitRef != "" {
		return fmt.Errorf("cannot specify --git-ref without --release-version")
	}
	ctx := context.Background()
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	bucket := release.NewBucket(gcs.Bucket(o.Bucket), release.DefaultBucketPathPrefix, o.ReleaseType)
	stagedReleases, err := bucket.ListReleases(ctx, o.ReleaseVersion, o.GitRef)
	if err != nil {
		return fmt.Errorf("failed listing staged releases: %w", err)
	}

	lines := []string{"NAME\tVERSION"}
	sort.Sort(ByVersion(stagedReleases))
	for _, rel := range stagedReleases {
		vers := rel.Metadata().ReleaseVersion
		lines = append(lines, fmt.Sprintf("%s\t%s", rel.Name(), vers))
	}

	logTable(lines...)

	return nil
}

func logTable(lines ...string) {
	// Observe how the b's and the d's, despite appearing in the
	// second cell of each line, belong to different columns.
	w := tabwriter.NewWriter(log.Writer(), 0, 0, 1, ' ', tabwriter.TabIndent)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	w.Flush()
}

type ByVersion []release.Staged

func (a ByVersion) Len() int      { return len(a) }
func (a ByVersion) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByVersion) Less(i, j int) bool {
	return semver.Compare(a[i].Metadata().ReleaseVersion, a[j].Metadata().ReleaseVersion) < 0
}
