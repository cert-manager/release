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

package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/release/manifests"
	"github.com/cert-manager/release/pkg/shell"
)

const (
	testKMSKey         = "projects/test-project/locations/test-location/keyRings/test-ring/cryptoKeys/test-key/cryptoKeyVersions/1"
	testKMSKeyCosign   = "gcpkms://projects/test-project/locations/test-location/keyRings/test-ring/cryptoKeys/test-key/versions/1"
	testOCIRegistry    = "quay.io/jetstack/charts"
	testHelmPath       = "/go/bin/helm"
	testCranePath      = "/go/bin/crane"
	testCosignPath     = "/go/bin/cosign"
	testReleaseVersion = "v1.99.0"
	testChartName      = "cert-manager"
	testChartVersion   = "v1.99.0"
)

// shellCall captures a single invocation made through the injected runner.
type shellCall struct {
	cmd  string
	args []string
}

// recorder is a fake shell.Runner that records every invocation. Callers can
// queue per-call errors via errs (one entry per expected call, nil for
// success). Calls past the length of errs return nil.
type recorder struct {
	calls []shellCall
	errs  []error
}

func (r *recorder) run(_ context.Context, _ string, cmd string, args ...string) error {
	idx := len(r.calls)
	r.calls = append(r.calls, shellCall{cmd: cmd, args: append([]string(nil), args...)})
	if idx < len(r.errs) {
		return r.errs[idx]
	}
	return nil
}

func (r *recorder) Runner() shell.Runner {
	return r.run
}

// writeChartTgz creates a minimal Helm chart tgz at the given path. The chart
// always contains a cert-manager/Chart.yaml with the supplied name/version. If
// withProv is true a sibling .prov file is also created.
func writeChartTgz(t *testing.T, dir, chartName, chartVersion string, withProv bool) string {
	t.Helper()

	chartPath := filepath.Join(dir, fmt.Sprintf("%s-%s.tgz", chartName, chartVersion))

	f, err := os.Create(chartPath)
	if err != nil {
		t.Fatalf("create chart tgz: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	chartYaml := fmt.Sprintf("name: %s\nversion: %s\nappVersion: %s\napiVersion: v1\n", chartName, chartVersion, chartVersion)
	if err := tw.WriteHeader(&tar.Header{
		Name: "cert-manager/Chart.yaml",
		Mode: 0o644,
		Size: int64(len(chartYaml)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write([]byte(chartYaml)); err != nil {
		t.Fatalf("write tar body: %v", err)
	}

	if withProv {
		if err := os.WriteFile(chartPath+".prov", []byte("fake-prov"), 0o644); err != nil {
			t.Fatalf("write prov: %v", err)
		}
	}

	return chartPath
}

// newTestRelease builds a release.Unpacked containing a single chart at the
// given version. The chart is written to a fresh tempdir owned by the test.
func newTestRelease(t *testing.T, releaseVersion string) *release.Unpacked {
	t.Helper()

	dir := t.TempDir()
	chartPath := writeChartTgz(t, dir, testChartName, testChartVersion, true)

	chart, err := manifests.NewChart(chartPath)
	if err != nil {
		t.Fatalf("load chart: %v", err)
	}

	return &release.Unpacked{
		ReleaseName:    "cert-manager-test",
		ReleaseVersion: releaseVersion,
		Charts:         []manifests.Chart{*chart},
	}
}

func newTestPublishOptions(runner shell.Runner) *gcbPublishOptions {
	o := NewGCBPublishOptions()
	o.PublishedHelmChartOCIRegistry = testOCIRegistry
	o.HelmPath = testHelmPath
	o.CranePath = testCranePath
	o.CosignPath = testCosignPath
	o.SigningKMSKey = testKMSKey
	o.Runner = runner
	return o
}

// assertCallEqual fails the test if the recorded call doesn't match wantCmd/wantArgs.
func assertCallEqual(t *testing.T, idx int, call shellCall, wantCmd string, wantArgs []string) {
	t.Helper()
	if call.cmd != wantCmd {
		t.Errorf("call %d: cmd = %q, want %q", idx, call.cmd, wantCmd)
	}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Errorf("call %d: args mismatch\n got: %v\nwant: %v", idx, call.args, wantArgs)
	}
}

func TestPushHelmChartOCI_VPrefixedVersion(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion) // "v1.99.0"

	if err := pushHelmChartOCI(context.Background(), o, rel); err != nil {
		t.Fatalf("pushHelmChartOCI failed: %v", err)
	}

	chartPath := rel.Charts[0].Path()
	vRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, testReleaseVersion)
	nonVRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, strings.TrimPrefix(testReleaseVersion, "v"))
	ociURL := "oci://" + testOCIRegistry

	wantCalls := []shellCall{
		// 1. helm version preflight
		{cmd: testHelmPath, args: []string{"version"}},
		// 2. crane version preflight
		{cmd: testCranePath, args: []string{"version"}},
		// 3. helm push
		{cmd: testHelmPath, args: []string{"push", chartPath, ociURL}},
		// 4. cosign sign v-prefixed
		{cmd: testCosignPath, args: []string{
			"sign",
			"--key", testKMSKeyCosign,
			"--tlog-upload=false",
			"--new-bundle-format=false",
			"--use-signing-config=false",
			vRef,
		}},
		// 5. cosign verify v-prefixed
		{cmd: testCosignPath, args: []string{
			"verify",
			"--key", testKMSKeyCosign,
			"--signature-digest-algorithm", "sha512",
			"--insecure-ignore-tlog=true",
			vRef,
		}},
		// 6. crane copy to non-v tag
		{cmd: testCranePath, args: []string{"copy", vRef, nonVRef}},
		// 7. cosign sign non-v
		{cmd: testCosignPath, args: []string{
			"sign",
			"--key", testKMSKeyCosign,
			"--tlog-upload=false",
			"--new-bundle-format=false",
			"--use-signing-config=false",
			nonVRef,
		}},
		// 8. cosign verify non-v
		{cmd: testCosignPath, args: []string{
			"verify",
			"--key", testKMSKeyCosign,
			"--signature-digest-algorithm", "sha512",
			"--insecure-ignore-tlog=true",
			nonVRef,
		}},
	}

	if len(rec.calls) != len(wantCalls) {
		t.Fatalf("got %d calls, want %d:\n got: %+v\nwant: %+v", len(rec.calls), len(wantCalls), rec.calls, wantCalls)
	}
	for i, want := range wantCalls {
		assertCallEqual(t, i, rec.calls[i], want.cmd, want.args)
	}
}

func TestPushHelmChartOCI_NonVPrefixedVersion(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, "1.99.0") // no v prefix

	if err := pushHelmChartOCI(context.Background(), o, rel); err != nil {
		t.Fatalf("pushHelmChartOCI failed: %v", err)
	}

	// Without a v prefix we should NOT see a crane copy or a second sign/verify.
	// Expect: helm version, crane version, helm push, cosign sign, cosign verify.
	if got, want := len(rec.calls), 5; got != want {
		t.Fatalf("got %d calls, want %d: %+v", got, want, rec.calls)
	}

	for _, c := range rec.calls {
		if c.cmd == testCranePath {
			for _, a := range c.args {
				if a == "copy" {
					t.Errorf("did not expect crane copy for non-v-prefixed version, got: %+v", c)
				}
			}
		}
	}
}

func TestPushHelmChartOCI_SkipSigning(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	o.SkipSigning = true
	rel := newTestRelease(t, testReleaseVersion)

	if err := pushHelmChartOCI(context.Background(), o, rel); err != nil {
		t.Fatalf("pushHelmChartOCI failed: %v", err)
	}

	// Expect: helm version, crane version, helm push. No sign/verify/copy.
	if got, want := len(rec.calls), 3; got != want {
		t.Fatalf("got %d calls, want %d: %+v", got, want, rec.calls)
	}
	for _, c := range rec.calls {
		if c.cmd == testCosignPath {
			t.Errorf("expected no cosign calls when skip-signing, got: %+v", c)
		}
	}
}

func TestPushHelmChartOCI_NoCharts(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := &release.Unpacked{
		ReleaseName:    "cert-manager-test",
		ReleaseVersion: testReleaseVersion,
	}

	err := pushHelmChartOCI(context.Background(), o, rel)
	if err == nil {
		t.Fatal("expected error when there are no charts, got nil")
	}
	if !strings.Contains(err.Error(), "no charts") {
		t.Errorf("expected error about missing charts, got: %v", err)
	}
}

func TestPushHelmChartOCI_HelmPreflightFails(t *testing.T) {
	rec := &recorder{errs: []error{errors.New("helm not found")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion)

	err := pushHelmChartOCI(context.Background(), o, rel)
	if err == nil {
		t.Fatal("expected error when helm preflight fails")
	}
	// We should fail immediately - just one call should have been made.
	if len(rec.calls) != 1 {
		t.Errorf("expected 1 call before short-circuiting, got %d: %+v", len(rec.calls), rec.calls)
	}
}

func TestPushHelmChartOCI_CranePreflightFails(t *testing.T) {
	rec := &recorder{errs: []error{nil, errors.New("crane not found")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion)

	err := pushHelmChartOCI(context.Background(), o, rel)
	if err == nil {
		t.Fatal("expected error when crane preflight fails")
	}
	if len(rec.calls) != 2 {
		t.Errorf("expected 2 calls before short-circuiting, got %d: %+v", len(rec.calls), rec.calls)
	}
}

func TestPushHelmChartOCI_PushFails(t *testing.T) {
	// helm version OK, crane version OK, helm push fails.
	rec := &recorder{errs: []error{nil, nil, errors.New("push failed")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion)

	err := pushHelmChartOCI(context.Background(), o, rel)
	if err == nil {
		t.Fatal("expected error when helm push fails")
	}
	if !strings.Contains(err.Error(), "push") {
		t.Errorf("expected error mentioning push, got: %v", err)
	}
	// We should NOT proceed to cosign after a push failure.
	for _, c := range rec.calls {
		if c.cmd == testCosignPath {
			t.Errorf("did not expect cosign calls after push failure, got: %+v", c)
		}
	}
}

func TestPushHelmChartOCI_InvalidKMSKey(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	o.SigningKMSKey = "not-a-valid-key"
	rel := newTestRelease(t, testReleaseVersion)

	err := pushHelmChartOCI(context.Background(), o, rel)
	if err == nil {
		t.Fatal("expected error for invalid KMS key, got nil")
	}
	// We should have failed before any cosign invocation.
	for _, c := range rec.calls {
		if c.cmd == testCosignPath {
			t.Errorf("did not expect cosign to be invoked with an invalid KMS key, got: %+v", c)
		}
	}
}

func TestPushHelmChartOCI_ChartWithoutProv(t *testing.T) {
	dir := t.TempDir()
	chartPath := writeChartTgz(t, dir, testChartName, testChartVersion, false)
	chart, err := manifests.NewChart(chartPath)
	if err != nil {
		t.Fatalf("load chart: %v", err)
	}
	rel := &release.Unpacked{
		ReleaseName:    "cert-manager-test",
		ReleaseVersion: testReleaseVersion,
		Charts:         []manifests.Chart{*chart},
	}
	if chart.ProvPath() != nil {
		t.Fatalf("test setup: expected chart without prov, got prov path %q", *chart.ProvPath())
	}

	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())

	if err := pushHelmChartOCI(context.Background(), o, rel); err != nil {
		t.Fatalf("pushHelmChartOCI failed: %v", err)
	}

	// Same command count as the v-prefixed test: missing .prov is a warning, not a hard failure.
	if got, want := len(rec.calls), 8; got != want {
		t.Fatalf("got %d calls, want %d: %+v", got, want, rec.calls)
	}
}

func TestNewGCBPublishOptionsHasDefaultRunner(t *testing.T) {
	o := NewGCBPublishOptions()
	if o.Runner == nil {
		t.Error("expected NewGCBPublishOptions to populate a default Runner so production callers don't need to set one")
	}
}

func TestHelmChartOCIIsRegisteredPublishAction(t *testing.T) {
	if _, ok := publishActionMap["helmchartoci"]; !ok {
		t.Error("expected 'helmchartoci' to be a registered publish action")
	}
	if _, ok := publishActionMap["helmchartpr"]; ok {
		t.Error("expected 'helmchartpr' to no longer be registered after deprecation")
	}
}
