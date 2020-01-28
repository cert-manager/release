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

	"github.com/cert-manager/release/pkg/release"
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
	for _, tars := range rel.ComponentImageBundles {
		for _, tar := range tars {
			if !strings.HasPrefix(tar.ImageName(), opts.ImageRepository+"/") {
				violations = append(violations, fmt.Sprintf("Image %q does not have prefix %q", tar.ImageName(), opts.ImageRepository+"/"))
			}
			if tar.ImageTag() != opts.ReleaseVersion {
				violations = append(violations, fmt.Sprintf("Image %q does not have expected tag %q", tar.ImageName(), opts.ReleaseVersion))
			}
		}
	}
	for _, ch := range rel.Charts {
		if ch.Version() != opts.ReleaseVersion {
			violations = append(violations, fmt.Sprintf("Helm chart sets 'version' to %q, expected %q", ch.Version(), opts.ReleaseVersion))
		}
		if ch.AppVersion() != opts.ReleaseVersion {
			violations = append(violations, fmt.Sprintf("Helm chart sets 'appVersion' to %q, expected %q", ch.AppVersion(), opts.ReleaseVersion))
		}
	}
	return violations, nil
}
