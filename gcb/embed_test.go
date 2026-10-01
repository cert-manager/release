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

package gcb_test

import (
	"testing"

	gcbconfig "github.com/cert-manager/release/gcb"
	"github.com/cert-manager/release/pkg/gcb"
)

// TestEmbeddedConfigsParse fails if any embedded cloudbuild.yaml no longer
// decodes, so a broken YAML is caught in CI rather than on release day.
func TestEmbeddedConfigsParse(t *testing.T) {
	for name, data := range map[string][]byte{
		"stage":         gcbconfig.Stage,
		"makestage":     gcbconfig.MakeStage,
		"publish":       gcbconfig.Publish,
		"bootstrap-pgp": gcbconfig.BootstrapPGP,
	} {
		if _, err := gcb.ParseBuild(data); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestPublishPinPlaceholder fails if the publish config stops failing closed
// when cmrel does not substitute _RELEASE_REPO_REF.
func TestPublishPinPlaceholder(t *testing.T) {
	build, err := gcb.ParseBuild(gcbconfig.Publish)
	if err != nil {
		t.Fatal(err)
	}
	if got := build.Substitutions["_RELEASE_REPO_REF"]; got != "PINNED_CMREL_COMMIT_SHA_REQUIRED" {
		t.Errorf("_RELEASE_REPO_REF = %q, want the fail-closed placeholder", got)
	}
}
