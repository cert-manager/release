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

// Sign calls out to cosign to sign a given container using the provided GCP key.
func Sign(ctx context.Context, cosignPath string, containers []string, key sign.GCPKMSKey) error {
	args := append([]string{
		"sign",
		"--key",
		key.CosignFormat(),
	}, containers...)

	return shell.Command(ctx, "", cosignPath, args...)
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
		blobPath,
	}
}

// Version calls "cosign version", both for informational purposes and as a check that the binary exists
func Version(ctx context.Context, cosignPath string) error {
	return shell.Command(ctx, "", cosignPath, []string{"version"}...)
}
