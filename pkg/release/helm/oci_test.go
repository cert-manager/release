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

package helm

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordedCall struct {
	workDir string
	cmd     string
	args    []string
}

func newRecorder(err error) (*[]recordedCall, func(ctx context.Context, workDir string, cmd string, args ...string) error) {
	calls := &[]recordedCall{}
	runner := func(ctx context.Context, workDir string, cmd string, args ...string) error {
		*calls = append(*calls, recordedCall{
			workDir: workDir,
			cmd:     cmd,
			args:    append([]string(nil), args...),
		})
		return err
	}
	return calls, runner
}

func TestPushChartToOCI(t *testing.T) {
	calls, runner := newRecorder(nil)

	err := PushChartToOCI(
		context.Background(),
		runner,
		"/tmp/cert-manager-v1.99.0.tgz",
		"oci://quay.io/jetstack/charts",
		"/usr/local/bin/helm",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}

	call := (*calls)[0]
	if call.cmd != "/usr/local/bin/helm" {
		t.Errorf("expected helm path, got %q", call.cmd)
	}
	wantArgs := []string{"push", "/tmp/cert-manager-v1.99.0.tgz", "oci://quay.io/jetstack/charts"}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Errorf("args mismatch\n got: %v\nwant: %v", call.args, wantArgs)
	}
}

func TestPushChartToOCIPropagatesError(t *testing.T) {
	wantErr := errors.New("helm push failed")
	_, runner := newRecorder(wantErr)

	err := PushChartToOCI(context.Background(), runner, "/x", "oci://r", "helm")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestCopyChartTag(t *testing.T) {
	calls, runner := newRecorder(nil)

	err := CopyChartTag(
		context.Background(),
		runner,
		"quay.io/jetstack/charts/cert-manager:v1.99.0",
		"quay.io/jetstack/charts/cert-manager:1.99.0",
		"/usr/local/bin/crane",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}

	call := (*calls)[0]
	if call.cmd != "/usr/local/bin/crane" {
		t.Errorf("expected crane path, got %q", call.cmd)
	}
	wantArgs := []string{
		"copy",
		"quay.io/jetstack/charts/cert-manager:v1.99.0",
		"quay.io/jetstack/charts/cert-manager:1.99.0",
	}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Errorf("args mismatch\n got: %v\nwant: %v", call.args, wantArgs)
	}
}

func TestCopyChartTagPropagatesError(t *testing.T) {
	wantErr := errors.New("crane copy failed")
	_, runner := newRecorder(wantErr)

	err := CopyChartTag(context.Background(), runner, "src", "dest", "crane")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}
