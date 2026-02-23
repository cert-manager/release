// +skip_license_check

// Copyright © 2018 Heptio

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package kmssigner implements a crypto.Signer backed by Google Cloud KMS.
package kmssigner

// This has been modified to suit cert-manager's use case; we add support for SHA512
// digests, which are forced by helm. This is copied across to the cert-manager/release
// project unless/until the changes are merged upstream in:
// https://github.com/heptiolabs/google-kms-pgp/

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"time"

	cloudkms "google.golang.org/api/cloudkms/v1"
)

// Signer extends crypto.Signer to provide more key metadata.
type Signer interface {
	crypto.Signer

	RSAPublicKey() *rsa.PublicKey

	CreationTime() time.Time

	HashAlgo() crypto.Hash
}

// NewWithExplicitMetadata returns a crypto.Signer backed by the named Google Cloud KMS key,
// but doesn't need the "cloudkms.cryptoKeyVersions.get" permission which would otherwise be required
// for fetching the hash algorithm and creation time.
func NewWithExplicitMetadata(api *cloudkms.Service, name string, hashAlgo crypto.Hash, creationTime time.Time) (Signer, error) {
	res, err := api.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.GetPublicKey(name).Do()
	if err != nil {
		return nil, fmt.Errorf("could not get public key from Google Cloud KMS API: %w", err)
	}

	block, _ := pem.Decode([]byte(res.Pem))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("could not decode public key PEM")
	}

	pubkey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse public key: %w", err)
	}

	pubkeyRSA, ok := pubkey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key was not an RSA key as expected; got type %T", pubkey)
	}

	return &kmsSigner{
		api:  api,
		name: name,

		pubkey:        *pubkeyRSA,
		creationTime:  creationTime,
		pgpDigestAlgo: hashAlgo,
	}, nil
}

// New returns a crypto.Signer backed by the named Google Cloud KMS key.
func New(api *cloudkms.Service, name string) (Signer, error) {
	metadata, err := api.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("could not get key version from Google Cloud KMS API: %w", err)
	}

	var hashAlgo crypto.Hash

	switch metadata.Algorithm {
	case "RSA_SIGN_PKCS1_2048_SHA256":
		hashAlgo = crypto.SHA256
	case "RSA_SIGN_PKCS1_3072_SHA256":
		hashAlgo = crypto.SHA256
	case "RSA_SIGN_PKCS1_4096_SHA256":
		hashAlgo = crypto.SHA256
	case "RSA_SIGN_PKCS1_4096_SHA512":
		hashAlgo = crypto.SHA512

	default:
		return nil, fmt.Errorf("unsupported key algorithm %q", metadata.Algorithm)
	}

	creationTime, err := time.Parse(time.RFC3339Nano, metadata.CreateTime)
	if err != nil {
		return nil, fmt.Errorf("could not parse key creation timestamp: %w", err)
	}

	return NewWithExplicitMetadata(api, name, hashAlgo, creationTime)
}

type kmsSigner struct {
	api          *cloudkms.Service
	name         string
	pubkey       rsa.PublicKey
	creationTime time.Time

	pgpDigestAlgo crypto.Hash
}

func (k *kmsSigner) Public() crypto.PublicKey {
	return k.pubkey
}

func (k *kmsSigner) RSAPublicKey() *rsa.PublicKey {
	return &k.pubkey
}

func (k *kmsSigner) CreationTime() time.Time {
	return k.creationTime
}

func (k *kmsSigner) HashAlgo() crypto.Hash {
	return k.pgpDigestAlgo
}

// KMSDigest returns a Digest corresponding to the given digest algorithm, or an error
// if the digest is the incorrect size
func (k *kmsSigner) KMSDigest(digest []byte) (*cloudkms.Digest, error) {
	if len(digest) != k.pgpDigestAlgo.Size() {
		return nil, fmt.Errorf("expected digest to have length %d but got %d", k.pgpDigestAlgo.Size(), len(digest))
	}

	encodedDigest := base64.StdEncoding.EncodeToString(digest)

	kmsDigest := new(cloudkms.Digest)
	switch k.pgpDigestAlgo {
	case crypto.SHA256:
		kmsDigest.Sha256 = encodedDigest

	case crypto.SHA512:
		kmsDigest.Sha512 = encodedDigest

	default:
		panic("unknown digest type")
	}

	return kmsDigest, nil
}

func (k *kmsSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	kmsDigest, err := k.KMSDigest(digest)
	if err != nil {
		return nil, fmt.Errorf("input digest must be valid size for given key type: %w", err)
	}

	sig, err := k.api.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.AsymmetricSign(
		k.name,
		&cloudkms.AsymmetricSignRequest{
			Digest: kmsDigest,
		},
	).Do()
	if err != nil {
		return nil, fmt.Errorf("error signing with Google Cloud KMS: %w", err)
	}

	res, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid Base64 response from Google Cloud KMS AsymmetricSign endpoint: %w", err)
	}

	return res, nil
}
