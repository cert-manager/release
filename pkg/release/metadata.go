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

package release

// Metadata about a staged release.
type Metadata struct {
	// ReleaseVersion, if set, is an explicit version used to build the release
	// artifacts.
	// By default, the release version will be computed by the Bazel release
	// process and the build type will be 'devel' instead of 'release'.
	ReleaseVersion string `json:"releaseVersion"`

	// Required git commit reference to build.
	GitCommitRef string `json:"gitCommitRef"`

	// GCS URIs of the artifacts that are a part of this build.
	Artifacts []ArtifactMetadata `json:"artifacts"`
}

type ArtifactMetadata struct {
	// Name of the artifact within the release directory.
	Name string `json:"name"`

	// SHA256 is a hash of the artifact, computed during the staging process.
	SHA256 string `json:"sha256"`

	// OS, if specified, is the OS parameter that this artifact was built for.
	// This could be 'linux', 'darwin', 'windows' etc.
	OS string `json:"os,omitempty"`

	// Architecture, if specified, is the architecture that this artifact was
	// built for.
	// This could be 'amd64', 'arm', 'arm64' etc.
	Architecture string `json:"architecture,omitempty"`
}
