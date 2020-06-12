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

package validation

import (
	"fmt"
	"strings"

	"github.com/blang/semver"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/release/images"
)

type Options struct {
	// ReleaseVersion is used to ensure that the artifacts in a staged release
	// all specify the same image tag and define a consistent version.
	ReleaseVersion string

	// ImageRepository is used to ensure that the artifacts in a staged release
	// all use the specified image repository prefix.
	ImageRepository string
}

func ValidateUnpackedRelease(opts Options, rel *release.Unpacked) ([]string, error) {
	var violations []string
	if err := validateSemver(rel.ReleaseVersion); err != nil {
		violations = append(violations, fmt.Sprintf("Release version %q is not semver compliant: %v", rel.ReleaseVersion, err))
	}
	violations = append(violations, validateImageBundles(rel.ComponentImageBundles, opts)...)
	for _, ch := range rel.Charts {
		if ch.Version() != opts.ReleaseVersion {
			violations = append(violations, fmt.Sprintf("Helm chart sets 'version' to %q, expected %q", ch.Version(), opts.ReleaseVersion))
		}
		if ch.AppVersion() != opts.ReleaseVersion {
			violations = append(violations, fmt.Sprintf("Helm chart sets 'appVersion' to %q, expected %q", ch.AppVersion(), opts.ReleaseVersion))
		}
	}
	if len(rel.CtlBinaryBundles) == 0 {
		violations = append(violations, fmt.Sprintf("No kubectl plugin binaries found in release - this is probably an error!"))
	}
	return violations, nil
}

func validateSemver(v string) error {
	if v[0] != 'v' {
		return fmt.Errorf("version number must have a leading 'v' character")
	}
	// trim v prefix as the semver library only offers ParseTolerant
	// which is not sufficient for us
	v = strings.TrimPrefix(v, "v")
	_, err := semver.Parse(v)
	return err
}

func validateImageBundles(bundles map[string][]images.Tar, opts Options) []string {
	var violations []string
	for componentName, tars := range bundles {
		for _, tar := range tars {
			expectedName := fmt.Sprintf("%s/cert-manager-%s-%s", opts.ImageRepository, componentName, tar.Architecture())
			actualNameWithoutTag := strings.Split(tar.ImageName(), ":")[0]
			if expectedName != actualNameWithoutTag {
				violations = append(violations, fmt.Sprintf("Image %q does not match expected name %q", actualNameWithoutTag, expectedName))
			}
			if tar.ImageTag() != opts.ReleaseVersion {
				violations = append(violations, fmt.Sprintf("Image %q does not have expected tag %q", tar.ImageName(), opts.ReleaseVersion))
			}

			expectedArch := tar.Architecture()
			actualArch := tar.ImageArchitecture()
			if expectedArch != actualArch {
				violations = append(violations, fmt.Sprintf("Image architecture %q does not match expected architecture %q", actualArch, expectedArch))
			}
		}
	}
	return violations
}
