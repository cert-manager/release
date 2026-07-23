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

// Sign calls out to cosign to sign a given container using the provided GCP key.
func Sign(ctx context.Context, cosignPath string, containers []string, key sign.GCPKMSKey) error {
	args := append([]string{
		"sign",
		"--key",
		key.CosignFormat(),
	}, containers...)

	return shell.Command(ctx, "", cosignPath, args...)
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
