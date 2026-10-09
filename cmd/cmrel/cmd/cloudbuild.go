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
	"log"

	"google.golang.org/api/cloudbuild/v1"

	"github.com/cert-manager/release/pkg/gcb"
)

const cloudBuildFlagHelp = "Path to a cloudbuild.yaml to use instead of the one embedded in this binary. " +
	"For cmrel development only: a release must use the embedded config so that it matches the pinned cmrel commit."

// loadCloudBuild returns the embedded config, or the file at path if one was
// given with --cloudbuild.
func loadCloudBuild(path string, embedded []byte) (*cloudbuild.Build, error) {
	if path == "" {
		return gcb.ParseBuild(embedded)
	}
	log.Printf("WARNING: --cloudbuild set; loading cloudbuild.yaml from %q instead of the embedded config", path)
	return gcb.LoadBuild(path)
}
