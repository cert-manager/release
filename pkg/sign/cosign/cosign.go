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
	"fmt"

	"github.com/cert-manager/release/pkg/shell"
	"github.com/cert-manager/release/pkg/sign"
)

// DefaultSignOptions are the cosign sign flags used for every artifact cmrel
// signs: container images, manifest lists and Helm charts. They keep the
// behaviour cosign v1 had for key-based signing, where nothing was uploaded to
// the transparency log unless COSIGN_EXPERIMENTAL was set.
//
// Why TlogUpload=false?
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
// disable it here to keep the same behaviour.
//
// cosign 3 also changes the default for "--new-bundle-format" to true, so we
// have to disable that too to keep the same behaviour as cosign 2, until we're
// able to verify that everything works with the new bundle format.
var DefaultSignOptions = SignOptions{
	TlogUpload:       false,
	NewBundleFormat:  false,
	UseSigningConfig: false,
}

// DefaultVerifyOptions are the cosign verify flags that match
// DefaultSignOptions: there is no tlog entry to check, and the KMS key signs
// with SHA-512.
var DefaultVerifyOptions = VerifyOptions{
	SignatureDigestAlgorithm: "sha512",
	InsecureIgnoreTlog:       true,
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

// SignOptions contains options for signing with cosign
type SignOptions struct {
	TlogUpload       bool
	NewBundleFormat  bool
	UseSigningConfig bool
}

// SignArgs builds the argument list for "cosign sign" with the given options.
// It is exported so callers (and tests) can inspect the exact CLI invocation
// without having to actually run cosign.
func SignArgs(container string, key sign.GCPKMSKey, opts SignOptions) []string {
	return []string{
		"sign",
		"--key", key.CosignFormat(),
		fmt.Sprintf("--tlog-upload=%t", opts.TlogUpload),
		fmt.Sprintf("--new-bundle-format=%t", opts.NewBundleFormat),
		fmt.Sprintf("--use-signing-config=%t", opts.UseSigningConfig),
		container,
	}
}

// SignWithOptions calls out to cosign to sign a container with specific options.
// The runner parameter allows tests to inject a fake. If runner is nil, the
// default real runner is used.
func SignWithOptions(ctx context.Context, runner shell.Runner, cosignPath string, container string, key sign.GCPKMSKey, opts SignOptions) error {
	if runner == nil {
		runner = shell.Default
	}
	return runner(ctx, "", cosignPath, SignArgs(container, key, opts)...)
}

// VerifyOptions contains options for verifying with cosign
type VerifyOptions struct {
	SignatureDigestAlgorithm string
	InsecureIgnoreTlog       bool
}

// VerifyArgs builds the argument list for "cosign verify" with the given options.
// It is exported so callers (and tests) can inspect the exact CLI invocation
// without having to actually run cosign.
func VerifyArgs(container string, key sign.GCPKMSKey, opts VerifyOptions) []string {
	args := []string{
		"verify",
		"--key", key.CosignFormat(),
	}

	if opts.SignatureDigestAlgorithm != "" {
		args = append(args, "--signature-digest-algorithm", opts.SignatureDigestAlgorithm)
	}

	if opts.InsecureIgnoreTlog {
		args = append(args, "--insecure-ignore-tlog=true")
	}

	args = append(args, container)
	return args
}

// VerifyWithOptions calls out to cosign to verify a container signature with specific options.
// The runner parameter allows tests to inject a fake. If runner is nil, the
// default real runner is used.
func VerifyWithOptions(ctx context.Context, runner shell.Runner, cosignPath string, container string, key sign.GCPKMSKey, opts VerifyOptions) error {
	if runner == nil {
		runner = shell.Default
	}
	return runner(ctx, "", cosignPath, VerifyArgs(container, key, opts)...)
}
