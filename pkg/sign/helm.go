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

package sign

import (
	"context"
	"fmt"
	"strings"

	helmsign "helm.sh/helm/v3/pkg/provenance"
)

// HelmChart signs a given packaged helm chart (usually a .tgz file) using the given
// KMS key, returning the human-readable signature bytes.
func HelmChart(ctx context.Context, key string, chartPath string) ([]byte, error) {
	signatory, err := signatoryFromKMS(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS signer: %w", err)
	}

	signature, err := signatory.ClearSign(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to sign %q: %w", chartPath, err)
	}

	// signatory.ClearSign doesn't necessarily return an error on failure; it can silently
	// return an empty signature and no error, so we check signature after checking the
	// error to confirm that a signature was actually made.

	// This can happen when an incorrect KMS key type is used, which helm can't handle
	if len(strings.TrimSpace(signature)) == 0 {
		return nil, fmt.Errorf("got empty signature from helm signing process; this can indicate a KMS key of an incorrect type")
	}

	return []byte(signature), nil
}

// signatoryFromKMS creates a Helm Signatory backed by a KMS key. The Signatory can then
// be used to sign helm charts, but won't also be usable for validating signatures.
func signatoryFromKMS(ctx context.Context, key string) (*helmsign.Signatory, error) {
	entity, _, err := deriveEntity(ctx, key)
	if err != nil {
		return nil, err
	}

	return &helmsign.Signatory{
		Entity: entity,
		// KeyRing is used for verification, which we don't support or care about
		// in cmrel since we expect end-users to use the helm CLI for verification
		KeyRing: nil,
	}, nil
}
