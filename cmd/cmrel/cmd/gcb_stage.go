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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/cert-manager/release/pkg/release"
)

const (
	gcbStageCommand         = "stage"
	gcbStageDescription     = "Stage release tarballs to a GCS release bucket"
	gcbStageLongDescription = `
The 'gcb stage' subcommand will run Bazel to produce a set of release artifacts
which are then uploaded (staged) to GCS.

This is the internal version of the 'stage' target. It is intended to be run by
a Google Cloud Build started via the 'stage' sub-command.
`
)

type gcbStageOptions struct {
	// The name of the GCS bucket to stage the release to.
	Bucket string

	// RepoPath is the path to a checked out copy of the cert-manager
	// repository at the desired ref to build for this release.
	RepoPath string

	// ReleaseVersion, if set, overrides the version git version tag used
	// during the build. This is used to force a build's version number to be
	// the final release tag before a tag has actually been created in the
	// repository.
	ReleaseVersion string

	// PublishedImageRepository is the docker repository that will be used for
	// built artifacts.
	// This must be set at the time a build is staged as parts of the release
	// incorporate this docker repository name.
	PublishedImageRepository string
}

func (o *gcbStageOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Bucket, "bucket", release.DefaultBucketName, "The name of the GCS bucket to stage the release to.")
	fs.StringVar(&o.RepoPath, "repo-path", "", "Path to the cert-manager repository stored in disk to be built and published. This must already be checked out at the appropriate revision.")
	fs.StringVar(&o.ReleaseVersion, "release-version", "", "Optional release version override used to force the version strings used during the release to a specific value.")
	fs.StringVar(&o.PublishedImageRepository, "published-image-repo", release.DefaultImageRepository, "The docker image repository set when building the release.")
}

func (o *gcbStageOptions) print() {
	log.Printf("GCB Stage options:")
	log.Printf("  Bucket: %q", o.Bucket)
	log.Printf("  RepoPath: %q", o.RepoPath)
	log.Printf("  ReleaseVersion: %q", o.ReleaseVersion)
}

func gcbStageCmd(rootOpts *rootOptions) *cobra.Command {
	o := &gcbStageOptions{}
	cmd := &cobra.Command{
		Use:          gcbStageCommand,
		Short:        gcbStageDescription,
		Long:         gcbStageLongDescription,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGCBStage(rootOpts, o)
		},
	}
	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))
	return cmd
}

func runGCBStage(rootOpts *rootOptions, o *gcbStageOptions) error {
	gitRef, err := readGitRef(o.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to read git ref from repository: %v", err)
	}

	if o.ReleaseVersion != "" {
		if err := runGit(o.RepoPath, "tag", "-f", o.ReleaseVersion); err != nil {
			return err
		}
		log.Printf("Tagged git repository at commit %q with version %q", gitRef, o.ReleaseVersion)
	}

	releaseVersion, err := readBazelVersion(o.RepoPath)
	if err != nil {
		return err
	}

	log.Printf("Building release artifacts with release version %q at ref %q", releaseVersion, gitRef)

	outputDir := ""
	// If --release-version is not explicitly set, we treat this build as a
	// 'devel' build and output into the development directory.
	if o.ReleaseVersion == "" {
		outputDir = release.BucketPathForRelease(release.DefaultBucketPathPrefix, release.BuildTypeDevel, releaseVersion, gitRef)
	} else {
		outputDir = release.BucketPathForRelease(release.DefaultBucketPathPrefix, release.BuildTypeRelease, releaseVersion, gitRef)
	}

	log.Printf("Built artifacts will be published to 'gs://%s/%s' once complete", o.Bucket, outputDir)

	// Create a list of built artifacts. For now this is pretty hardcoded and ugly.
	// In future, we may want to update cert-manager's build system to produce a 'manifest'
	// of all the artifacts that were built during a `bazel run` invocation.
	// This will mean we don't have to update this release tool whenever we add an additional
	// release artifact.
	allArchs := sets.NewString(append(release.ServerArchitectures, release.UBIArchitectures...)...)
	osVariant := "linux"
	for _, arch := range allArchs.List() {
		// hardcode 'os' to 'linux' for docker image builds
		log.Printf("Building %q target for %q architecture", release.TarsBazelTarget, arch)
		if err := runBazel(o.RepoPath, bazelBuildEnv(o), "build", "--stamp", platformFlagForOSArch(osVariant, arch), release.TarsBazelTarget); err != nil {
			return fmt.Errorf("failed building release artifacts for architecture %q: %w", arch, err)
		}
	}

	var artifacts []release.ArtifactMetadata
	for _, arch := range release.ServerArchitectures {
		// create an artifact for the arch specific 'server' release tarball
		artifactName := fmt.Sprintf("cert-manager-server-linux-%s.tar.gz", arch)
		// Add the arch-specific .tar.gz file to the list of artifacts
		if err := appendArtifact(&artifacts, o.RepoPath, artifactName, osVariant, arch); err != nil {
			return err
		}
	}
	for _, arch := range release.UBIArchitectures {
		// create an artifact for the arch specific 'server' release tarball
		artifactName := fmt.Sprintf("cert-manager-ubi-linux-%s.tar.gz", arch)
		// Add the arch-specific .tar.gz file to the list of artifacts
		if err := appendArtifact(&artifacts, o.RepoPath, artifactName, osVariant, arch); err != nil {
			return err
		}
	}
	// Add the 'manifests' file to the release output
	if err := appendArtifact(&artifacts, o.RepoPath, "cert-manager-manifests.tar.gz", "", ""); err != nil {
		return err
	}

	log.Printf("Built release artifacts for all architectures: %v", artifacts)

	// Build Google Cloud Storage API client for uploading artifacts
	ctx := context.Background()
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Upload all built release artifacts
	for _, artifact := range artifacts {
		filePath := buildArtifactPath(o.RepoPath, "build", "release-tars", artifact.Name)
		gcsPath := buildObjectName(outputDir, artifact.Name)
		log.Printf("Uploading artifact %q to GCS at path: %s", artifact, gcsPath)
		if err := func(filePath, gcsPath string) error {
			r, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer r.Close()

			w := gcs.Bucket(o.Bucket).Object(gcsPath).NewWriter(ctx)
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
			log.Printf("Uploaded artifact %q to GCS", artifact)

			return nil
		}(filePath, gcsPath); err != nil {
			return fmt.Errorf("failed to copy output artifact to GCS staging location: %w", err)
		}
	}

	meta, err := json.MarshalIndent(release.Metadata{
		ReleaseVersion: o.ReleaseVersion,
		GitCommitRef:   gitRef,
		Artifacts:      artifacts,
	}, "", " ")
	if err != nil {
		return fmt.Errorf("failed to encode metadata output: %w", err)
	}

	log.Printf("Uploading release metadata")
	w := gcs.Bucket(o.Bucket).Object(buildObjectName(outputDir, release.MetadataFileName)).NewWriter(ctx)
	if _, err := w.Write(meta); err != nil {
		return fmt.Errorf("failed to write release metadata to GCS staging location: %w", err)
	}
	if err := w.Close(); err != nil {
		return err
	}

	log.Printf("Successfully staged release with version %q", releaseVersion)

	return nil
}

func bazelBuildEnv(opts *gcbStageOptions) []string {
	return append(os.Environ(), "DOCKER_REGISTRY="+opts.PublishedImageRepository)
}

func appendArtifact(artifacts *[]release.ArtifactMetadata, repoPath, name, os, arch string) error {
	artifactPath := buildArtifactPath(repoPath, "build", "release-tars", name)
	artifactHash, err := sha256SumFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to compute sha256sum of release artifact %q: %w", artifactPath, err)
	}
	*artifacts = append(*artifacts, release.ArtifactMetadata{
		Name:         name,
		SHA256:       artifactHash,
		OS:           os,
		Architecture: arch,
	})
	return nil
}

func platformFlagForOSArch(os, arch string) string {
	return fmt.Sprintf("--platforms=@io_bazel_rules_go//go/toolchain:%s_%s", os, arch)
}

func runGit(wd string, args ...string) error {
	return runCmd(wd, "git", args...)
}

func runBazel(wd string, env []string, args ...string) error {
	return runCmdWithEnv(wd, env, "bazel", args...)
}

func runCmd(wd, cmd string, args ...string) error {
	return runCmdWithEnv(wd, nil, cmd, args...)
}

func runCmdWithEnv(wd string, env []string, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	// redirect all output
	// TODO: honour --debug flag
	c.Env = env
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Dir = wd
	return c.Run()
}

// readBazelVersion will build the //:version Bazel target and read the
// contents of the 'version' file generated.
func readBazelVersion(wd string) (string, error) {
	if err := runBazel(wd, nil, "build", "//:version"); err != nil {
		return "", err
	}

	vBytes, err := ioutil.ReadFile(buildArtifactPath(wd, "version"))
	if err != nil {
		return "", err
	}
	vers := strings.TrimSpace(string(vBytes))
	return vers, nil
}

func readGitRef(wd string) (string, error) {
	c := exec.Command("git", "rev-parse", "HEAD")
	b := &strings.Builder{}
	c.Stdout = b
	c.Stderr = os.Stderr
	c.Dir = wd
	if err := c.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}

func buildArtifactPath(repoRoot string, artifactPaths ...string) string {
	return filepath.Join(append([]string{repoRoot, "bazel-bin"}, artifactPaths...)...)
}

func buildObjectName(outputDir, name string) string {
	return fmt.Sprintf("%s/%s", outputDir, name)
}

func sha256SumFile(filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
