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
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	chartYaml := fmt.Sprintf("name: %s\nversion: %s\nappVersion: %s\napiVersion: v1\n", chartName, chartVersion, chartVersion)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "cert-manager/Chart.yaml",
		Mode: 0o644,
		Size: int64(len(chartYaml)),
	}))
	_, err = tw.Write([]byte(chartYaml))
	require.NoError(t, err)

	if withProv {
		require.NoError(t, os.WriteFile(chartPath+".prov", []byte("fake-prov"), 0o644))
	}

	return chartPath
}

// newTestRelease builds a release.Unpacked containing a single chart at the
// given version. The chart is written to a fresh tempdir owned by the test.
// The chart version matches the release version, as release validation
// requires for a real release.
func newTestRelease(t *testing.T, releaseVersion string, withProv bool) *release.Unpacked {
	t.Helper()

	chart, err := manifests.NewChart(writeChartTgz(t, t.TempDir(), testChartName, releaseVersion, withProv))
	require.NoError(t, err)

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

func TestPushHelmChartOCI_VPrefixedVersion(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion, true) // "v1.99.0"

	require.NoError(t, pushHelmChartOCI(t.Context(), o, rel))

	chartPath := rel.Charts[0].Path()
	vRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, testReleaseVersion)
	nonVRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, strings.TrimPrefix(testReleaseVersion, "v"))
	ociURL := "oci://" + testOCIRegistry

	assert.Equal(t, []shellCall{
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
	}, rec.calls)
}

func TestPushHelmChartOCI_NonVPrefixedVersion(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, "1.99.0", true) // no v prefix

	require.NoError(t, pushHelmChartOCI(t.Context(), o, rel))

	// Without a v prefix we should NOT see a crane copy or a second sign/verify.
	// Expect: helm version, crane version, helm push, cosign sign, cosign verify.
	require.Len(t, rec.calls, 5)
	for _, c := range rec.calls {
		assert.NotContains(t, c.args, "copy")
	}
}

func TestPushHelmChartOCI_SkipSigning(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	o.SkipSigning = true
	rel := newTestRelease(t, testReleaseVersion, true)

	require.NoError(t, pushHelmChartOCI(t.Context(), o, rel))

	// Skipping signing must not skip publishing: the non-v tag is still copied.
	vRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, testReleaseVersion)
	nonVRef := fmt.Sprintf("%s/cert-manager:%s", testOCIRegistry, strings.TrimPrefix(testReleaseVersion, "v"))
	assert.Equal(t, []shellCall{
		{cmd: testHelmPath, args: []string{"version"}},
		{cmd: testCranePath, args: []string{"version"}},
		{cmd: testHelmPath, args: []string{"push", rel.Charts[0].Path(), "oci://" + testOCIRegistry}},
		{cmd: testCranePath, args: []string{"copy", vRef, nonVRef}},
	}, rec.calls)
}

func TestPushHelmChartOCI_NoCharts(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	rel := &release.Unpacked{
		ReleaseName:    "cert-manager-test",
		ReleaseVersion: testReleaseVersion,
	}

	require.ErrorContains(t, pushHelmChartOCI(t.Context(), o, rel), "no charts")
	assert.Empty(t, rec.calls)
}

func TestPushHelmChartOCI_HelmPreflightFails(t *testing.T) {
	rec := &recorder{errs: []error{errors.New("helm not found")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion, true)

	require.Error(t, pushHelmChartOCI(t.Context(), o, rel))
	// We should fail immediately - just one call should have been made.
	assert.Len(t, rec.calls, 1)
}

func TestPushHelmChartOCI_CranePreflightFails(t *testing.T) {
	rec := &recorder{errs: []error{nil, errors.New("crane not found")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion, true)

	require.Error(t, pushHelmChartOCI(t.Context(), o, rel))
	assert.Len(t, rec.calls, 2)
}

func TestPushHelmChartOCI_PushFails(t *testing.T) {
	// helm version OK, crane version OK, helm push fails.
	rec := &recorder{errs: []error{nil, nil, errors.New("push failed")}}
	o := newTestPublishOptions(rec.Runner())
	rel := newTestRelease(t, testReleaseVersion, true)

	require.ErrorContains(t, pushHelmChartOCI(t.Context(), o, rel), "push")
	// We should NOT proceed to cosign after a push failure.
	assert.Len(t, rec.calls, 3)
}

func TestPushHelmChartOCI_InvalidKMSKey(t *testing.T) {
	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())
	o.SigningKMSKey = "not-a-valid-key"
	rel := newTestRelease(t, testReleaseVersion, true)

	require.Error(t, pushHelmChartOCI(t.Context(), o, rel))
	// We should have failed before any cosign invocation.
	for _, c := range rec.calls {
		assert.NotEqual(t, testCosignPath, c.cmd)
	}
}

func TestPushHelmChartOCI_ChartWithoutProv(t *testing.T) {
	rel := newTestRelease(t, testReleaseVersion, false)
	require.Nil(t, rel.Charts[0].ProvPath(), "test setup: expected chart without prov")

	rec := &recorder{}
	o := newTestPublishOptions(rec.Runner())

	require.NoError(t, pushHelmChartOCI(t.Context(), o, rel))

	// Same command count as the v-prefixed test: missing .prov is a warning, not a hard failure.
	assert.Len(t, rec.calls, 8)
}

func TestNewGCBPublishOptionsHasDefaultRunner(t *testing.T) {
	assert.NotNil(t, NewGCBPublishOptions().Runner, "production callers should not need to set a Runner")
}

// TestPublishActionListOrder checks that the chart is published after the
// images it references, whatever order the actions are given in.
func TestPublishActionListOrder(t *testing.T) {
	for name := range publishActionMap {
		assert.Contains(t, publishActionOrder, name)
	}

	o := NewGCBPublishOptions()
	o.PublishActions = []string{"helmchartoci", "pushcontainerimages"}
	funcs, err := o.PublishActionList()
	require.NoError(t, err)
	var got []string
	for _, f := range funcs {
		got = append(got, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
	}
	assert.Equal(t, []string{
		"github.com/cert-manager/release/cmd/cmrel/cmd.pushContainerImages",
		"github.com/cert-manager/release/cmd/cmrel/cmd.pushHelmChartOCI",
	}, got)
}

func TestHelmChartOCIIsRegisteredPublishAction(t *testing.T) {
	assert.Contains(t, publishActionMap, "helmchartoci")
	assert.NotContains(t, publishActionMap, "helmchartpr", "helmchartpr was removed")
}
