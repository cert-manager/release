/*
Copyright 2026 The cert-manager Authors.

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
	"fmt"
	"runtime/debug"
)

// releaseRepoRef returns the git ref of the cert-manager/release repository that
// GCB build jobs should install cmrel from, i.e. the value of the
// _RELEASE_REPO_REF substitution.
func releaseRepoRef() (string, error) {
	info, ok := debug.ReadBuildInfo()
	return releaseRepoRefFromBuildInfo(info, ok)
}

// releaseRepoRefFromBuildInfo implements releaseRepoRef against an explicit
// build.BuildInfo so the ref-resolution logic can be unit tested.
func releaseRepoRefFromBuildInfo(info *debug.BuildInfo, ok bool) (string, error) {
	if !ok || info == nil {
		return "", fmt.Errorf("unable to read build info to determine the cmrel commit; build cmrel with `go build`/`go install` from a clean checkout of cert-manager/release")
	}

	var revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}

	// Built from a VCS working tree (e.g. `make build` / `go build ./cmd/cmrel`).
	if revision != "" {
		if modified == "true" {
			return "", fmt.Errorf("cmrel was built from a modified working tree (commit %s); GCB cannot install an unpushed build - rebuild from a clean, pushed commit", revision)
		}
		return revision, nil
	}

	// Installed as a module (e.g. `go install .../cmrel@<ref>`); Main.Version is a
	// tag or pseudo-version that `go install` can resolve. "(devel)" means the
	// binary was built from source without VCS stamping, which we cannot pin.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v, nil
	}

	return "", fmt.Errorf("unable to determine the commit cmrel was built from; build cmrel with `go build`/`go install` from cert-manager/release so its version is stamped")
}
