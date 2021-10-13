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

import "testing"

func TestGCPKMSKey(t *testing.T) {
	tests := map[string]struct {
		input             string
		expectedCosignKey string
		shouldError       bool
	}{
		"parses GCP formatted key": {
			input:             "projects/cert-manager-release/locations/europe-west1/keyRings/cert-manager-release/cryptoKeys/cert-manager-release-signing-key/cryptoKeyVersions/1",
			expectedCosignKey: "gcpkms://projects/cert-manager-release/locations/europe-west1/keyRings/cert-manager-release/cryptoKeys/cert-manager-release-signing-key/versions/1",
			shouldError:       false,
		},
		"doesn't parse cosign formatted key": {
			input:       "projects/cert-manager-release/locations/europe-west1/keyRings/cert-manager-release/cryptoKeys/cert-manager-release-signing-key/versions/1",
			shouldError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			key, err := NewGCPKMSKey(test.input)

			if (err != nil) != test.shouldError {
				t.Errorf("shouldError=%v, err=%v", test.shouldError, err)
				return
			}

			if test.shouldError {
				return
			}

			// input format _is_ GCP format
			if key.GCPFormat() != test.input {
				t.Errorf("wanted GCP formatted key %q but got %q", test.input, key.GCPFormat())
			}

			if key.CosignFormat() != test.expectedCosignKey {
				t.Errorf("wanted cosign formatted key %q but got %q", test.expectedCosignKey, key.CosignFormat())
			}
		})
	}
}
