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

package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// tarEntry describes a single entry to write into a test tar.gz archive.
type tarEntry struct {
	name     string
	typeflag byte
	mode     int64
	linkname string
	body     string
}

// buildTarGz builds an in-memory gzipped tar archive from the given entries.
func buildTarGz(t *testing.T, entries []tarEntry) *bytes.Reader {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     e.mode,
			Linkname: e.linkname,
			Size:     int64(len(e.body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if len(e.body) > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("failed to write tar body: %v", err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

func TestUntarGz(t *testing.T) {
	tests := map[string]struct {
		entries   []tarEntry
		expectErr bool
	}{
		"extracts a well-formed archive": {
			entries: []tarEntry{
				{name: "server", typeflag: tar.TypeDir, mode: 0755},
				{name: "server/images", typeflag: tar.TypeDir, mode: 0755},
				{name: "server/images/controller.tar", typeflag: tar.TypeReg, mode: 0644, body: "image"},
			},
		},
		"rejects a parent-directory traversal entry": {
			entries: []tarEntry{
				{name: "../../go/bin/cosign", typeflag: tar.TypeReg, mode: 0755, body: "evil"},
			},
			expectErr: true,
		},
		"rejects an absolute-path entry": {
			entries: []tarEntry{
				{name: "/etc/cron.d/evil", typeflag: tar.TypeReg, mode: 0644, body: "evil"},
			},
			expectErr: true,
		},
		"rejects a symlink entry": {
			entries: []tarEntry{
				{name: "link", typeflag: tar.TypeSymlink, mode: 0777, linkname: "/etc/passwd"},
			},
			expectErr: true,
		},
		"rejects a hardlink entry": {
			entries: []tarEntry{
				{name: "link", typeflag: tar.TypeLink, mode: 0644, linkname: "../../go/bin/cosign"},
			},
			expectErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dst := t.TempDir()
			r := buildTarGz(t, test.entries)

			err := UntarGz(dst, r)
			if test.expectErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("got an unexpected error: %v", err)
			}

			if test.expectErr {
				// Ensure nothing escaped the destination directory.
				if _, statErr := os.Stat(filepath.Join(dst, "..", "..", "go", "bin", "cosign")); statErr == nil {
					t.Fatalf("traversal entry escaped the destination directory")
				}
			}
		})
	}
}
