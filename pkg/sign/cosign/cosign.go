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

// SignWithOptions calls out to cosign to sign a container with specific options
func SignWithOptions(ctx context.Context, cosignPath string, container string, key sign.GCPKMSKey, opts SignOptions) error {
	args := []string{
		"sign",
		"--key", key.CosignFormat(),
		"--tlog-upload=" + fmt.Sprintf("%t", opts.TlogUpload),
		"--new-bundle-format=" + fmt.Sprintf("%t", opts.NewBundleFormat),
		"--use-signing-config=" + fmt.Sprintf("%t", opts.UseSigningConfig),
		container,
	}

	return shell.Command(ctx, "", cosignPath, args...)
}

// VerifyOptions contains options for verifying with cosign
type VerifyOptions struct {
	SignatureDigestAlgorithm string
	InsecureIgnoreTlog       bool
}

// VerifyWithOptions calls out to cosign to verify a container signature with specific options
func VerifyWithOptions(ctx context.Context, cosignPath string, container string, key sign.GCPKMSKey, opts VerifyOptions) error {
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

	return shell.Command(ctx, "", cosignPath, args...)
}
