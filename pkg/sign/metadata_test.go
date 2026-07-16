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

package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// signTestMetadata signs metadata with key the same way SignMetadata does, but
// using a locally-held private key rather than KMS, so the verification logic
// can be exercised without access to GCP.
func signTestMetadata(t *testing.T, key *rsa.PrivateKey, metadata []byte) []byte {
	t.Helper()

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, metadataDigest(metadata))
	if err != nil {
		t.Fatalf("failed to sign test metadata: %v", err)
	}

	return signature
}

func TestVerifyMetadataSignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	metadata := []byte(`{"releaseVersion":"v1.2.3","gitCommitRef":"abc123"}`)
	signature := signTestMetadata(t, key, metadata)

	t.Run("valid signature verifies", func(t *testing.T) {
		if err := verifyMetadataSignature(&key.PublicKey, metadata, signature); err != nil {
			t.Errorf("expected valid signature to verify, got error: %v", err)
		}
	})

	t.Run("tampered metadata is rejected", func(t *testing.T) {
		tampered := []byte(`{"releaseVersion":"v9.9.9","gitCommitRef":"abc123"}`)
		if err := verifyMetadataSignature(&key.PublicKey, tampered, signature); err == nil {
			t.Error("expected tampered metadata to fail verification, got nil error")
		}
	})

	t.Run("tampered signature is rejected", func(t *testing.T) {
		tampered := make([]byte, len(signature))
		copy(tampered, signature)
		tampered[0] ^= 0xff
		if err := verifyMetadataSignature(&key.PublicKey, metadata, tampered); err == nil {
			t.Error("expected tampered signature to fail verification, got nil error")
		}
	})

	t.Run("signature from a different key is rejected", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("failed to generate second test key: %v", err)
		}
		if err := verifyMetadataSignature(&otherKey.PublicKey, metadata, signature); err == nil {
			t.Error("expected signature from a different key to fail verification, got nil error")
		}
	})
}
