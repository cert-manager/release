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

import "k8s.io/apimachinery/pkg/util/sets"

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
		"darwin":  []string{"amd64", "arm64"},
		"windows": []string{"amd64"},
	}

	// ArchitecturesPerOS is the list of OSes and architectures that we can build
	// This is used to drive the `--platforms` flag passed to 'bazel build'
	ArchitecturesPerOS = map[string][]string{
		"linux":   []string{"amd64", "arm", "arm64", "ppc64le", "s390x"},
		"darwin":  []string{"amd64", "arm64"},
		"windows": []string{"amd64"},
	}
)

// AllOSes returns a slice of all known operating systems which cert-manager targets
// including both server and client targets.
func AllOSes() []string {
	// initialise set with server platforms
	platforms := sets.NewString(mapKeys(ServerPlatforms)...)

	// add client platforms
	platforms = platforms.Insert(mapKeys(ClientPlatforms)...)

	return platforms.List()
}

func mapKeys(in map[string][]string) []string {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}

	return keys
}
