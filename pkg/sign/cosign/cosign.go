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

package cosign

import (
	"context"
	"strings"

	"github.com/cert-manager/release/pkg/shell"
)

// Sign calls out to cosign to sign a given container using the provided GCP key.
func Sign(ctx context.Context, containers []string, key string) error {
	if !strings.HasPrefix(key, "gcpkms://") {
		key = "gcpkms://" + key
	}

	args := append([]string{
		"sign",
		"-key",
		key,
	}, containers...)

	return shell.Command(ctx, "", "cosign", args...)
}
