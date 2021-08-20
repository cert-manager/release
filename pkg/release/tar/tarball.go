/*
Copyright 2020 The Jetstack cert-manager contributors.

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
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// UntarGz takes a destination path and a reader; a tar reader loops over the
// tarfile creating the file structure at 'dst' along the way, and writing any
// files.
func UntarGz(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

// ReadSingleFile will read a single file from a tar archive and return with
// its contents as a []byte.
// This should only be used to read small files from tar archives.
func ReadSingleFile(filename string, r io.Reader) ([]byte, error) {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		// if no more files are found, break
		if err == io.EOF {
			break
		}
		// return any other error
		if err != nil {
			return nil, err
		}
		// if the header is nil, just skip it (not sure how this happens)
		if header == nil {
			continue
		}
		// if this isn't the file we're looking for, continue
		if filename != header.Name {
			continue
		}
		if header.Typeflag == tar.TypeDir {
			return nil, fmt.Errorf("expected path %q to be a file, but it was a directory", filename)
		}
		return io.ReadAll(tr)
	}
	return nil, fmt.Errorf("could not find file %q in tar input", filename)
}
