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

package cosign

import (
	"reflect"
	"testing"

	"github.com/cert-manager/release/pkg/sign"
)

const testKeyResource = "projects/proj/locations/loc/keyRings/ring/cryptoKeys/key/cryptoKeyVersions/1"

func testKey(t *testing.T) sign.GCPKMSKey {
	t.Helper()
	key, err := sign.NewGCPKMSKey(testKeyResource)
	if err != nil {
		t.Fatalf("failed to parse test key: %v", err)
	}
	return key
}

func TestVerifyBlobArgs(t *testing.T) {
	got := verifyBlobArgs(testKey(t), "/tmp/metadata.json", "/tmp/metadata.json.sig")
	want := []string{
		"verify-blob",
		"--key",
		"gcpkms://projects/proj/locations/loc/keyRings/ring/cryptoKeys/key/versions/1",
		"--signature",
		"/tmp/metadata.json.sig",
		"/tmp/metadata.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("verifyBlobArgs() = %v, want %v", got, want)
	}
}
