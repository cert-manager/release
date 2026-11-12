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
	"For cmrel development only: a release must use the embedded config, which is the one this binary was built with."

// loadCloudBuild returns the embedded config, or the file at path if one was
// given with --cloudbuild. The returned build always has a non-nil
// Substitutions map, so callers can assign to it.
func loadCloudBuild(path string, embedded string) (*cloudbuild.Build, error) {
	var (
		build *cloudbuild.Build
		err   error
	)
	if path == "" {
		log.Printf("Using the cloudbuild.yaml embedded in this binary")
		build, err = gcb.ParseBuild([]byte(embedded))
	} else {
		log.Printf("WARNING: --cloudbuild set; loading cloudbuild.yaml from %q instead of the embedded config", path)
		build, err = gcb.LoadBuild(path)
	}
	if err != nil {
		return nil, err
	}
	if build.Substitutions == nil {
		build.Substitutions = map[string]string{}
	}
	return build, nil
}
