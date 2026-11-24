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
	"archive/tar"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	helmsign "helm.sh/helm/v4/pkg/provenance"
)

// fakeKMSSigner stands in for kmssigner: it is a crypto.Signer that is not one
// of the private key types go-crypto knows about, its Public returns a value
// like kmssigner's does, and like KMS it only accepts SHA-512 digests.
type fakeKMSSigner struct {
	key *rsa.PrivateKey
}

func (f *fakeKMSSigner) Public() crypto.PublicKey     { return f.key.PublicKey }
func (f *fakeKMSSigner) RSAPublicKey() *rsa.PublicKey { return &f.key.PublicKey }
func (f *fakeKMSSigner) CreationTime() time.Time      { return staticKeyCreationTime }
func (f *fakeKMSSigner) HashAlgo() crypto.Hash        { return crypto.SHA512 }

func (f *fakeKMSSigner) Sign(r io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	if len(digest) != crypto.SHA512.Size() {
		return nil, fmt.Errorf("expected a SHA-512 digest, got %d bytes", len(digest))
	}
	return rsa.SignPKCS1v15(r, f.key, crypto.SHA512, digest)
}

// TestBootstrapKeyVerifiesSignedChart runs the bootstrap-pgp and sign helm
// code paths with a stand-in KMS signer, then checks that helm accepts the
// chart signature using only the armored public key that bootstrap-pgp prints.
func TestBootstrapKeyVerifiesSignedChart(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	entity := entityFromSigner(&fakeKMSSigner{key: key})

	if err := addSelfSignedIdentity(entity, &packet.Config{DefaultHash: crypto.SHA512}); err != nil {
		t.Fatalf("addSelfSignedIdentity: %v", err)
	}
	publicKey, err := armoredPublicKey(entity)
	if err != nil {
		t.Fatalf("armoredPublicKey: %v", err)
	}

	dir := t.TempDir()
	chartPath := writeChart(t, dir)

	provData, err := signHelmChart(&helmsign.Signatory{Entity: entity}, chartPath)
	if err != nil {
		t.Fatalf("signHelmChart: %v", err)
	}

	keyringPath := filepath.Join(dir, "pubkey.asc")
	if err := os.WriteFile(keyringPath, []byte(publicKey), 0o600); err != nil {
		t.Fatal(err)
	}
	verifier, err := helmsign.NewFromKeyring(keyringPath, "")
	if err != nil {
		t.Fatalf("load armored public key: %v", err)
	}

	archiveData, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(archiveData, provData, filepath.Base(chartPath)); err != nil {
		t.Fatalf("helm does not accept the chart signature with the bootstrapped public key: %v", err)
	}
}

// writeChart writes a minimal packaged chart and returns its path.
func writeChart(t *testing.T, dir string) string {
	t.Helper()

	chartYAML := []byte("apiVersion: v2\nname: cert-manager\nversion: v0.0.0\n")
	path := filepath.Join(dir, "cert-manager-v0.0.0.tgz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "cert-manager/Chart.yaml", Mode: 0o644, Size: int64(len(chartYAML))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(chartYAML); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
