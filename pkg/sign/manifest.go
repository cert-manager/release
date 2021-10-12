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

package sign

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	tarball "github.com/cert-manager/release/pkg/release/tar"
)

const manifestLocation = "deploy/chart/cert-manager.tgz"

// CertManagerManifests takes a path to a cert-manager-manifests.tar.gz file, loads it into
// memory and signs anything inside the archive which is signable; currently,
// the helm chart located at "deploy/chart/cert-manager.tgz" is signed, and a
// signature "deploy/chart/cert-manager.tgz.prov" will be added.
// The cert-manifests.tar.gz file is changed in-place.
func CertManagerManifests(ctx context.Context, key GCPKMSKey, path string) error {
	// 1. Create temp dir for chart archive to be extracted to
	// (Helm signing requires a filename, not a reader, so we have to write to disk here)
	tmpDest, err := os.MkdirTemp("", "cmrel-extracted-manifests-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for extracting manifests: %w", err)
	}

	defer func() {
		err := os.RemoveAll(tmpDest)
		if err != nil {
			log.Printf("failed to remove temporary directory %q: %v", tmpDest, err)
		}
	}()

	// 2. Read + ungzip the manifests archive into memory
	tarData, originalMode, err := ungzipManifestArchive(path)
	if err != nil {
		return err
	}

	chartFileData, err := tarball.ReadSingleFile(manifestLocation, bytes.NewReader(tarData))
	if err != nil {
		return fmt.Errorf("failed to read %q from %q: %w", manifestLocation, path, err)
	}

	// Write file to temp location (for helm's sake)
	chartPath := filepath.Join(tmpDest, "cert-manager.tgz")
	err = os.WriteFile(chartPath, chartFileData, 0o644)
	if err != nil {
		return err
	}

	// 3. Sign chart
	signatureBytes, err := HelmChart(ctx, key, chartPath)
	if err != nil {
		return fmt.Errorf("failed to sign helm chart at %q: %w", chartPath, err)
	}

	// 4. Chmod the archive if needed so that it's writable, with a defer to reset its permissions
	// after we're done. This is required because bazel forces the mode to 0o555.
	modeResetFunc, err := ensureWritable(path, originalMode)
	if err != nil {
		return err
	}

	defer modeResetFunc()

	// 5. Append file to original tar archive

	// The tar spec requires that the end include two full empty blocks, so any valid tar file
	// will have exactly 512 * 2 bytes of empty space at the end.
	// We copy until the beginning of these empty blocks, then add our new header and our file,
	// and then close the tar.Writer to complete the tar archive.
	// See https://stackoverflow.com/a/18330903/1615417 for more details

	// NB: We can't just open the file as O_APPEND and seek back 1024 bytes because it's gzipped
	provPath := manifestLocation + ".prov"

	newTar, err := signatureToTar(signatureBytes, provPath, 0o644)
	if err != nil {
		return err
	}

	targzOut := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(targzOut)

	_, err = gzipWriter.Write(append(tarData[:len(tarData)-1024], newTar...))
	if err != nil {
		return fmt.Errorf("failed to compress tar output with helm signature: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return fmt.Errorf("couldn't finish writing gzip file: %w", err)
	}

	err = os.WriteFile(path, targzOut.Bytes(), originalMode)
	if err != nil {
		return fmt.Errorf("failed to write output tar file: %w", err)
	}

	log.Printf("successfully signed helm chart %q and added signature to %q as %q", chartPath, path, provPath)

	return nil
}

func setOwnerWritable(mode os.FileMode) os.FileMode {
	//      r  w  x
	// bits 2, 1, 0 are for world permissions
	// bits 5, 4, 3 are for group permissions
	// bits 8, 7, 6 are for owner permissions

	// we want to ensure that bit 7 is set, which is the write permission for owners
	return mode | 1<<7
}

func ensureWritable(path string, originalMode os.FileMode) (func() error, error) {
	newMode := setOwnerWritable(originalMode)

	err := os.Chmod(path, newMode)
	if err != nil {
		return nil, err
	}

	return func() error {
		return os.Chmod(path, originalMode)
	}, nil
}

func ungzipManifestArchive(path string) ([]byte, os.FileMode, error) {
	originalManifest, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open packaged manifest file %q: %w", path, err)
	}

	defer originalManifest.Close()

	manifestStat, err := originalManifest.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat original manifest file: %w", err)
	}

	mode := manifestStat.Mode()

	gzipReader, err := gzip.NewReader(originalManifest)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create gzip reader for %q: %w", path, err)
	}

	tarData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read tar data from %q: %w", path, err)
	}

	return tarData, mode, nil
}

func signatureToTar(signatureBytes []byte, filename string, mode os.FileMode) ([]byte, error) {
	newTar := &bytes.Buffer{}
	tarWriter := tar.NewWriter(newTar)

	err := tarWriter.WriteHeader(&tar.Header{
		Name: filename,
		Size: int64(len(signatureBytes)),
		Mode: int64(mode),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to write tar header for new prov file %q: %w", filename, err)
	}

	_, err = tarWriter.Write(signatureBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to add signature file %q to tar archive: %w", filename, err)
	}

	err = tarWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to finish writing and close tar archive data for %q: %w", filename, err)
	}

	return newTar.Bytes(), nil
}
