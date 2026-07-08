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

package zip

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// zipEntry describes a single entry to write into a test zip archive.
type zipEntry struct {
	name string
	mode os.FileMode
	body string
}

// buildZip writes a zip archive containing the given entries to a temp file and
// returns the opened *os.File, seeked to the start.
func buildZip(t *testing.T, entries []zipEntry) *os.File {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "archive-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp zip: %v", err)
	}

	zw := zip.NewWriter(f)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name}
		hdr.SetMode(e.mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatalf("failed to write zip body: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("failed to seek zip: %v", err)
	}

	return f
}

func TestUnzip(t *testing.T) {
	tests := map[string]struct {
		entries   []zipEntry
		expectErr bool
	}{
		"extracts a well-formed archive": {
			entries: []zipEntry{
				{name: "cmctl", mode: 0755, body: "binary"},
				{name: "LICENSE", mode: 0644, body: "license"},
			},
		},
		"rejects a parent-directory traversal entry": {
			entries: []zipEntry{
				{name: "../../go/bin/cosign", mode: 0755, body: "evil"},
			},
			expectErr: true,
		},
		"rejects an absolute-path entry": {
			entries: []zipEntry{
				{name: "/etc/cron.d/evil", mode: 0644, body: "evil"},
			},
			expectErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dst := t.TempDir()
			f := buildZip(t, test.entries)
			defer f.Close()

			err := Unzip(dst, f)
			if test.expectErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("got an unexpected error: %v", err)
			}

			if test.expectErr {
				if _, statErr := os.Stat(filepath.Join(dst, "..", "..", "go", "bin", "cosign")); statErr == nil {
					t.Fatalf("traversal entry escaped the destination directory")
				}
			}
		})
	}
}
