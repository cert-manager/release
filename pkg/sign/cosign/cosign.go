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

	"github.com/cert-manager/release/pkg/shell"
	"github.com/cert-manager/release/pkg/sign"
)

// Sign calls out to cosign to sign the image or chart at ref using the provided
// GCP KMS key.
func Sign(ctx context.Context, runner shell.Runner, cosignPath, ref string, key sign.GCPKMSKey) error {
	return runner(ctx, "", cosignPath, signArgs(ref, key)...)
}

// signArgs returns the "cosign sign" arguments used for every artifact cmrel
// signs: container images, manifest lists and Helm charts. They keep the
// behavior cosign v1 had for key-based signing, where nothing was uploaded to
// the transparency log unless COSIGN_EXPERIMENTAL was set.
//
// Why --tlog-upload=false?
// This flag prevents us creating a tlog entry for the signature, which is
// usually a good thing to do. Unfortunately, as well as creating the tlog
// entry, cosign also attempts to verify the tlog entry, which is the issue
// we run into - our KMS key uses SHA-512 as the signature digest algorithm,
// but there's no option to specify the digest algorithm for the tlog entry,
// so verification fails. We solved this for "cosign verify" with a cosign PR[0]
// a while back, but this problem hasn't been solved for tlog verification.
// [0]: https://github.com/sigstore/cosign/pull/1071
//
// As of cosign 3, --tlog-upload=false is deprecated and we'll eventually have
// to migrate to using "--signing-config". "--tlog-upload" is incompatible with
// "--use-signing-config=true". The default in cosign 2 is "--use-signing-config=false".
// The default in cosign 3 is "--use-signing-config=true", so we have to manually
// disable it here to keep the same behavior.
//
// cosign 3 also changes the default for "--new-bundle-format" to true, so we
// have to disable that too to keep the same behavior as cosign 2, until we're
// able to verify that everything works with the new bundle format.
func signArgs(ref string, key sign.GCPKMSKey) []string {
	return []string{
		"sign",
		"--key", key.CosignFormat(),
		"--tlog-upload=false",
		"--new-bundle-format=false",
		"--use-signing-config=false",
		ref,
	}
}

// Verify calls out to cosign to check the signature that Sign made for ref.
func Verify(ctx context.Context, runner shell.Runner, cosignPath, ref string, key sign.GCPKMSKey) error {
	return runner(ctx, "", cosignPath, verifyArgs(ref, key)...)
}

// verifyArgs returns the "cosign verify" arguments that match signArgs: there
// is no tlog entry to check, and the KMS key signs with SHA-512.
func verifyArgs(ref string, key sign.GCPKMSKey) []string {
	return []string{
		"verify",
		"--key", key.CosignFormat(),
		"--signature-digest-algorithm", "sha512",
		"--insecure-ignore-tlog=true",
		ref,
	}
}

// VerifyBlob calls out to cosign to verify that the detached signature at
// signaturePath is a valid signature of the file at blobPath, made with the
// provided GCP KMS key. It returns a non-nil error if verification fails.
func VerifyBlob(ctx context.Context, cosignPath, blobPath, signaturePath string, key sign.GCPKMSKey) error {
	return shell.Command(ctx, "", cosignPath, verifyBlobArgs(key, blobPath, signaturePath)...)
}

func verifyBlobArgs(key sign.GCPKMSKey, blobPath, signaturePath string) []string {
	return []string{
		"verify-blob",
		"--key",
		key.CosignFormat(),
		"--signature",
		signaturePath,
		// Staging signs metadata.json with "cosign sign-blob --tlog-upload=false"
		// (see make/release.mk in cert-manager), so there is no tlog entry to
		// look up. Without this flag cosign v2+ fails the verification when it
		// cannot find one.
		"--insecure-ignore-tlog=true",
		blobPath,
	}
}

// Version calls "cosign version", both for informational purposes and as a check that the binary exists
func Version(ctx context.Context, cosignPath string) error {
	return shell.Command(ctx, "", cosignPath, []string{"version"}...)
}
