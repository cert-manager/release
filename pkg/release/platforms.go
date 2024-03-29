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

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"k8s.io/apimachinery/pkg/util/sets"
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
func AllOSes() sets.String {
	// initialise set with server platforms
	platforms := sets.NewString(mapKeys(ServerPlatforms)...)

	// add client platforms
	platforms = platforms.Insert(mapKeys(ClientPlatforms)...)

	return platforms
}

// AllArchesForOSes returns all architectures targetted by any of the given OSes. Panics
// if any of the OSes are unknown
func AllArchesForOSes(osList sets.String) sets.String {
	knownArches := sets.String{}

	for _, os := range osList.List() {
		arches, ok := ArchitecturesPerOS[os]
		if !ok {
			panic("unknown OS when trying to get architectures: " + os)
		}

		knownArches = knownArches.Insert(arches...)
	}

	return knownArches
}

// AllOSes returns a slice of all known architectures which cert-manager targets
// including both server and client targets.

func mapKeys(in map[string][]string) []string {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}

	return keys
}

// OSListFromString parses and validates a comma-separated list of OSes, returning an error if any are invalid or a slice
// of valid OSes if all are OK.
func OSListFromString(targetOSes string) (sets.String, error) {
	allOSes := AllOSes()

	splitList := strings.Split(strings.TrimSpace(targetOSes), ",")

	if len(splitList) == 1 && splitList[0] == "*" {
		return allOSes, nil
	}

	osListOut := sets.String{}

	for _, rawOS := range splitList {
		os := strings.ToLower(strings.TrimSpace(rawOS))

		if len(os) == 0 {
			continue
		}

		if !allOSes.Has(os) {
			return nil, fmt.Errorf("unknown os %q", rawOS)
		}

		osListOut = osListOut.Insert(os)
	}

	if len(osListOut) == 0 {
		return nil, fmt.Errorf("invalid OS list; no OSes specified")
	}

	return osListOut, nil
}

// ArchListFromString parses and validates a comma-separated list of arches, returning a slice of valid arches if all are OK.
// Returns an error on an unknown architecture or an architecture which isn't a valid target for the given
// OSes for this invocation (e.g. will error if osList == []string{"windows"} and targetArches == "s390x")
// Panics if given an unknown OS
func ArchListFromString(targetArches string, osList sets.String) (sets.String, error) {
	allArches := AllArchesForOSes(osList)

	splitList := strings.Split(targetArches, ",")

	if len(splitList) == 1 && splitList[0] == "*" {
		return allArches, nil
	}

	archListOut := sets.String{}

	for _, rawArch := range splitList {
		arch := strings.ToLower(strings.TrimSpace(rawArch))

		if len(arch) == 0 {
			continue
		}

		if !allArches.Has(arch) {
			return nil, fmt.Errorf("unknown arch %q; if it's a valid arch, it might not be supported on any of the given OSes", rawArch)
		}

		archListOut = archListOut.Insert(arch)
	}

	if len(archListOut) == 0 {
		return nil, fmt.Errorf("invalid architecture list; no arches specified")
	}

	return archListOut, nil
}

// IsServerOS returns true if cert-manager can be deployed to the given OS on the server side
func IsServerOS(os string) bool {
	_, isServer := ServerPlatforms[os]
	return isServer
}

// IsClientOS returns true if cert-manager builds client binaries for the given platform
func IsClientOS(os string) bool {
	_, isClient := ClientPlatforms[os]
	return isClient
}

// Cmctl is only shipped with v1.14.X and below.
func CmctlIsShipped(releaseVersion string) bool {
	releaseVersion, _ = strings.CutPrefix(releaseVersion, "v")
	return semver.MustParse(releaseVersion).LT(semver.MustParse("1.15.0-alpha.0"))
}
