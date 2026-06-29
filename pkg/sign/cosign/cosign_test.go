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
	"errors"
	"reflect"
	"testing"

	"github.com/cert-manager/release/pkg/sign"
)

const (
	testKeyRaw         = "projects/test-project/locations/test-location/keyRings/test-ring/cryptoKeys/test-key/cryptoKeyVersions/1"
	testKeyCosignValue = "gcpkms://projects/test-project/locations/test-location/keyRings/test-ring/cryptoKeys/test-key/versions/1"
	testContainer      = "quay.io/jetstack/charts/cert-manager:v1.99.0"
)

func mustKey(t *testing.T) sign.GCPKMSKey {
	t.Helper()
	k, err := sign.NewGCPKMSKey(testKeyRaw)
	if err != nil {
		t.Fatalf("failed to parse test key: %v", err)
	}
	return k
}

func TestSignArgs(t *testing.T) {
	key := mustKey(t)

	tests := map[string]struct {
		opts SignOptions
		want []string
	}{
		"all flags false (matches the legacy hack/push_and_sign_chart.sh)": {
			opts: SignOptions{
				TlogUpload:       false,
				NewBundleFormat:  false,
				UseSigningConfig: false,
			},
			want: []string{
				"sign",
				"--key", testKeyCosignValue,
				"--tlog-upload=false",
				"--new-bundle-format=false",
				"--use-signing-config=false",
				testContainer,
			},
		},
		"all flags true": {
			opts: SignOptions{
				TlogUpload:       true,
				NewBundleFormat:  true,
				UseSigningConfig: true,
			},
			want: []string{
				"sign",
				"--key", testKeyCosignValue,
				"--tlog-upload=true",
				"--new-bundle-format=true",
				"--use-signing-config=true",
				testContainer,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := SignArgs(testContainer, key, tc.opts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SignArgs mismatch\n got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

func TestVerifyArgs(t *testing.T) {
	key := mustKey(t)

	tests := map[string]struct {
		opts VerifyOptions
		want []string
	}{
		"sha512 + ignore tlog (matches the legacy hack/push_and_sign_chart.sh)": {
			opts: VerifyOptions{
				SignatureDigestAlgorithm: "sha512",
				InsecureIgnoreTlog:       true,
			},
			want: []string{
				"verify",
				"--key", testKeyCosignValue,
				"--signature-digest-algorithm", "sha512",
				"--insecure-ignore-tlog=true",
				testContainer,
			},
		},
		"no optional flags set": {
			opts: VerifyOptions{},
			want: []string{
				"verify",
				"--key", testKeyCosignValue,
				testContainer,
			},
		},
		"only digest algorithm set": {
			opts: VerifyOptions{
				SignatureDigestAlgorithm: "sha256",
			},
			want: []string{
				"verify",
				"--key", testKeyCosignValue,
				"--signature-digest-algorithm", "sha256",
				testContainer,
			},
		},
		"only ignore tlog set": {
			opts: VerifyOptions{
				InsecureIgnoreTlog: true,
			},
			want: []string{
				"verify",
				"--key", testKeyCosignValue,
				"--insecure-ignore-tlog=true",
				testContainer,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := VerifyArgs(testContainer, key, tc.opts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("VerifyArgs mismatch\n got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

// recordedCall captures the inputs of a single Runner invocation.
type recordedCall struct {
	cmd  string
	args []string
}

// recordingRunner returns a shell.Runner that records every invocation and
// returns the (optional) error supplied. It never executes anything.
func recordingRunner(calls *[]recordedCall, err error) func(ctx context.Context, workDir string, cmd string, args ...string) error {
	return func(ctx context.Context, workDir string, cmd string, args ...string) error {
		*calls = append(*calls, recordedCall{cmd: cmd, args: append([]string(nil), args...)})
		return err
	}
}

func TestSignWithOptionsInvokesRunner(t *testing.T) {
	key := mustKey(t)

	var calls []recordedCall
	runner := recordingRunner(&calls, nil)

	opts := SignOptions{}
	if err := SignWithOptions(context.Background(), runner, "/usr/bin/cosign", testContainer, key, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(calls))
	}
	if calls[0].cmd != "/usr/bin/cosign" {
		t.Errorf("expected cmd=/usr/bin/cosign, got %q", calls[0].cmd)
	}
	wantArgs := SignArgs(testContainer, key, opts)
	if !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("args mismatch\n got: %v\nwant: %v", calls[0].args, wantArgs)
	}
}

func TestSignWithOptionsPropagatesError(t *testing.T) {
	key := mustKey(t)

	wantErr := errors.New("cosign failed")
	var calls []recordedCall
	runner := recordingRunner(&calls, wantErr)

	err := SignWithOptions(context.Background(), runner, "cosign", testContainer, key, SignOptions{})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestVerifyWithOptionsInvokesRunner(t *testing.T) {
	key := mustKey(t)

	var calls []recordedCall
	runner := recordingRunner(&calls, nil)

	opts := VerifyOptions{
		SignatureDigestAlgorithm: "sha512",
		InsecureIgnoreTlog:       true,
	}
	if err := VerifyWithOptions(context.Background(), runner, "cosign", testContainer, key, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(calls))
	}
	wantArgs := VerifyArgs(testContainer, key, opts)
	if !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Errorf("args mismatch\n got: %v\nwant: %v", calls[0].args, wantArgs)
	}
}
