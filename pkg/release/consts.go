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

package release

import (
	"fmt"
)

const (
	// MetadataFileName is the name of the file in the root of a staged
	// release.
	MetadataFileName = "metadata.json"

	// TarsBazelTarget is the Bazel target used to build release tar files in
	// the cert-manager repository.
	TarsBazelTarget = "//build/release-tars"

	// DefaultBucketName is the default GCS bucket used to store release
	// artifacts. This is re-used throughout the cmd/ package and used as the
	// default value for flags.
	DefaultBucketName = "cert-manager-release"

	// DefaultReleaseProject is the default project to run Cloud Build jobs in
	// to stage and publish releases.
	DefaultReleaseProject = "cert-manager-release"

	// DefaultBucketPathPrefix is the default prefix prepended to paths written
	// to Google Cloud Storage.
	DefaultBucketPathPrefix = "stage/gcb"

	// DefaultImageRepository is the default image repository used for artifact
	// images.
	DefaultImageRepository = "quay.io/jetstack"

	// DefaultGitHubOrg is the default organisation containing the cert-manager
	// repository.
	DefaultGitHubOrg = "jetstack"

	// DefaultGitHubRepo is the default repository containing the cert-manager
	// code.
	DefaultGitHubRepo = "cert-manager"

	// DefaultHelmChartBucket is the name of the default Google Cloud Storage
	// bucket where Helm charts should be published to.
	DefaultHelmChartBucket = "jetstack-chart-museum"

	// BuildTypeRelease denotes that a build is targeting an actual named
	// release and is not just a development build that has been created using
	// the release tool.
	BuildTypeRelease = "release"

	// BuildTypeDevel denotes that a build did not explicitly set a
	// --release-version and so it is not suitable for being used as part of a
	// published release.
	BuildTypeDevel = "devel"
)

var (
	// ServerPlatforms is the list of OSes and architectures to build docker images
	// for during the release.
	// This is used to drive the `--platforms` flag passed to 'bazel build' as
	// well as to determine which image artifacts should be uploaded.
	ServerPlatforms = map[string][]string{
		"linux": []string{"amd64", "arm", "arm64", "ppc64le", "s390x"},
	}

	// ClientPlatforms is the list of OSes and architectures to build client CLI tools
	// for during the release.
	// This is used to determine which artifacts should be uploaded.
	ClientPlatforms = map[string][]string{
		"linux":   []string{"amd64", "arm", "arm64", "ppc64le", "s390x"},
		"darwin":  []string{"amd64"},
		"windows": []string{"amd64"},
	}

	// ArchitecturesPerOS is the list of OSes and architectures that we can build
	// This is used to drive the `--platforms` flag passed to 'bazel build'
	ArchitecturesPerOS = map[string][]string{
		"linux":   []string{"amd64", "arm", "arm64", "ppc64le", "s390x"},
		"darwin":  []string{"amd64"},
		"windows": []string{"amd64"},
	}
)

// BucketPathForRelease will assemble an output directory path for the given
// release parameters.
func BucketPathForRelease(bucketPrefix, buildType, releaseVersion, gitRef string) string {
	if buildType == BuildTypeRelease {
		return fmt.Sprintf("%s/%s/%s-%s", bucketPrefix, buildType, releaseVersion, gitRef)
	}
	return fmt.Sprintf("%s/%s/%s", bucketPrefix, buildType, gitRef)
}
