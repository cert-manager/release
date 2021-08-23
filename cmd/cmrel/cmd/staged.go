/*
Copyright 2020 The Jetstack cert-manager contributors.

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
	"text/tabwriter"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/cert-manager/release/pkg/release"
)

const (
	stagedCommand         = "staged"
	stagedDescription     = "Staged release tarballs to a GCS release bucket"
	stagedLongDescription = `The staged command will build and staged a cert-manager release to a
Google Cloud Storage bucket. It will create a Google Cloud Build job
which will run a full cross-build and publish the artifacts to the
staging release bucket.
`
)

var (
	stagedExample = fmt.Sprintf(`
To staged a release of the 'master' branch to the default staging bucket, run:

	%s %s --git-ref=master

To staged a release of the 'release-0.14' branch to the default staging bucket,
overriding the release version as 'v0.14.0', run:

	%s %s --git-ref=release-0.14 --release-version=v0.14.0`, rootCommand, stagedCommand, rootCommand, stagedCommand)
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
	fs.StringVar(&o.ReleaseType, "release-type", "release", "The type of release to list - usually one of 'release' or 'devel'")
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
		Long:         stagedLongDescription,
		Example:      stagedExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
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

func runStaged(rootOpts *rootOptions, o *stagedOptions) error {
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

	lines := []string{"NAME\tVERSION\tDATE"}
	for _, rel := range stagedReleases {
		vers := rel.Metadata().ReleaseVersion
		lines = append(lines, fmt.Sprintf("%s\t%s\tUNKNOWN", rel.Name(), vers))
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
