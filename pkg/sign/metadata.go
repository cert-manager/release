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
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha512"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"

	"github.com/cert-manager/release/pkg/sign/internal/kmssigner"
)

// metadataSigningHash is the hash algorithm used when signing and verifying
// release metadata with a GCP KMS key.
const metadataSigningHash = crypto.SHA512

// SignMetadata signs the given release metadata with the named GCP KMS key and
// returns a detached RSA PKCS#1 v1.5 signature over its SHA-512 digest.
func SignMetadata(ctx context.Context, key GCPKMSKey, metadata []byte) ([]byte, error) {
	signer, err := metadataSigner(ctx, key)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(nil, metadataDigest(metadata), metadataSigningHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign metadata with KMS key %q: %w", key, err)
	}

	return signature, nil
}

// VerifyMetadata verifies a detached signature previously produced by
// SignMetadata against the given metadata, using the public key of the named
// GCP KMS key. It returns a non-nil error if the signature is invalid.
func VerifyMetadata(ctx context.Context, key GCPKMSKey, metadata []byte, signature []byte) error {
	signer, err := metadataSigner(ctx, key)
	if err != nil {
		return err
	}

	return verifyMetadataSignature(signer.RSAPublicKey(), metadata, signature)
}

// verifyMetadataSignature verifies signature against metadata using the given
// RSA public key. It is separated from VerifyMetadata so the verification logic
// can be unit tested without access to GCP KMS.
func verifyMetadataSignature(pub *rsa.PublicKey, metadata []byte, signature []byte) error {
	if err := rsa.VerifyPKCS1v15(pub, metadataSigningHash, metadataDigest(metadata), signature); err != nil {
		return fmt.Errorf("metadata signature verification failed: %w", err)
	}

	return nil
}

// metadataDigest returns the SHA-512 digest of the given metadata, which is what
// is actually signed and verified.
func metadataDigest(metadata []byte) []byte {
	digest := sha512.Sum512(metadata)
	return digest[:]
}

// metadataSigner constructs a KMS-backed signer for the given key using an
// explicit SHA-512 hash, avoiding the need for the cloudkms.cryptoKeyVersions.get permission.
// Both signing and verification fetch the key's public key from KMS.
func metadataSigner(ctx context.Context, key GCPKMSKey) (kmssigner.Signer, error) {
	oauthClient, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("could not create GCP OAuth2 client: %w", err)
	}

	svc, err := cloudkms.NewService(ctx, option.WithHTTPClient(oauthClient))
	if err != nil {
		return nil, fmt.Errorf("could not create GCP KMS client: %w", err)
	}

	// The creation time is only relevant when the signer is used to build a PGP
	// entity (where it forms part of the key ID); it has no effect on raw
	// AsymmetricSign, so we reuse the same static value used elsewhere.
	signer, err := kmssigner.NewWithExplicitMetadata(svc, key.GCPFormat(), metadataSigningHash, staticKeyCreationTime)
	if err != nil {
		return nil, fmt.Errorf("could not create KMS signer: %w", err)
	}

	return signer, nil
}
