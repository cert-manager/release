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
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Unzip unzips an archive held in `r` to the destination directory `dst`
func Unzip(dst string, r *os.File) error {
	fileInfo, err := r.Stat()
	if err != nil {
		return err
	}

	unzipper, err := zip.NewReader(r, fileInfo.Size())
	if err != nil {
		return err
	}

	for _, f := range unzipper.File {
		err = writeFileFromZipArchive(dst, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeFileFromZipArchive(dst string, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer rc.Close()

	// reject any entry whose name would escape the destination directory
	// (e.g. "../../go/bin/cosign" or an absolute path).
	if !filepath.IsLocal(f.Name) {
		return fmt.Errorf("refusing to extract %q: entry path escapes the destination directory", f.Name)
	}

	targetFilename := filepath.Join(dst, f.Name)

	// O_EXCL ensures we never follow into or overwrite an existing file, and we
	// mask to permission bits so a malicious archive cannot set setuid/setgid/sticky bits.
	outFile, err := os.OpenFile(targetFilename, os.O_CREATE|os.O_RDWR|os.O_EXCL, f.Mode().Perm())
	if err != nil {
		return err
	}

	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return err
	}

	return nil
}
