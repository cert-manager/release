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

package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cert-manager/release/pkg/release/binaries"
	"github.com/cert-manager/release/pkg/release/images"
	"github.com/cert-manager/release/pkg/release/manifests"
	"github.com/cert-manager/release/pkg/release/tar"
	"github.com/cert-manager/release/pkg/release/zip"
)

// Unpacked wraps a staged release that has been fetched and unpacked locally.
// It provides methods to interact with the release in order to prepare it for
// publishing.
type Unpacked struct {
	ReleaseName           string
	ReleaseVersion        string
	GitCommitRef          string
	Charts                []manifests.Chart
	YAMLs                 []manifests.YAML
	CtlBinaryBundles      []binaries.Archive
	ComponentImageBundles map[string][]*images.Tar
}

// Unpack takes a staged release, inspects its metadata, fetches referenced
// artifacts and extracts them to disk.
func Unpack(ctx context.Context, s *Staged) (*Unpacked, error) {
	log.Printf("Unpacking staged release %q", s.Name())

	log.Printf("Unpacking 'manifests' type artifact")
	manifestsA, err := manifestArtifactForStaged(s)
	if err != nil {
		return nil, err
	}
	manifestsDir, err := extractStagedArtifactToTempDir(ctx, manifestsA)
	if err != nil {
		return nil, err
	}
	log.Printf("Unpacked 'manifests' artifact to directory: %s", manifestsDir)

	// chart packages have a .tgz file extension
	chartPaths, err := recursiveFindWithExt(manifestsDir, ".tgz")
	if err != nil {
		return nil, err
	}
	var charts []manifests.Chart
	for _, path := range chartPaths {
		c, err := manifests.NewChart(path)
		if err != nil {
			return nil, err
		}
		charts = append(charts, *c)
	}
	log.Printf("Extracted %d Helm charts from manifests archive", len(charts))

	// static manifests have a .yaml file extension
	yamlPaths, err := recursiveFindWithExt(manifestsDir, ".yaml")
	if err != nil {
		return nil, err
	}
	var yamls []manifests.YAML
	for _, path := range yamlPaths {
		yamls = append(yamls, *manifests.NewYAML(path))
	}
	log.Printf("Extracted %d YAML manifests from manifests archive", len(yamls))

	bundles, err := unpackServerImagesFromRelease(ctx, s)
	if err != nil {
		return nil, err
	}
	log.Printf("Extracted %d component bundles from images archive", len(bundles))

	ctlBinaryBundles, err := unpackCtlFromRelease(ctx, s)
	if err != nil {
		return nil, err
	}
	log.Printf("Extracted %d multi arch ctl bundles from cmctl and kubectl-cert_manager archives", len(ctlBinaryBundles))

	return &Unpacked{
		ReleaseName:           s.Name(),
		ReleaseVersion:        s.Metadata().ReleaseVersion,
		GitCommitRef:          s.Metadata().GitCommitRef,
		YAMLs:                 yamls,
		Charts:                charts,
		CtlBinaryBundles:      ctlBinaryBundles,
		ComponentImageBundles: bundles,
	}, nil
}

// unpackServerImagesFromRelease will extract all 'image-like' tar archives
// from the various 'server' .tar.gz files and return a map of component name
// to a slice of images.Tar for each image in the bundle.
func unpackServerImagesFromRelease(ctx context.Context, s *Staged) (map[string][]*images.Tar, error) {
	log.Printf("Unpacking 'server' type artifacts")
	serverA := s.ArtifactsOfKind("server")
	return unpackImages(ctx, serverA, "")
}

// unpackCtlFromRelease extracts all ctl archives from the various 'ctl' .tar.gz / .zip files
// a slice of binaries.Archive holding each ctl binary in the bundle.
func unpackCtlFromRelease(ctx context.Context, s *Staged) ([]binaries.Archive, error) {
	log.Printf("Unpacking 'cmctl' and 'kubectl-cert_manager' type artifacts")

	if s.Metadata().BuildSource == BuildSourceMake {
		return unpackCtlFromMakeRelease(ctx, s)
	} else {
		return unpackCtlFromBazelRelease(ctx, s)
	}
}

func unpackCtlFromMakeRelease(ctx context.Context, s *Staged) ([]binaries.Archive, error) {
	// Example layouts of make ctl archives, containing just the binary + license file
	// cert-manager-cmctl-linux-amd64.tar.gz
	//   ├── cmctl
	//   └── LICENSE
	// cert-manager-cmctl-windows-amd64.zip
	//   ├── cmctl
	//   └── LICENSE
	var binaryBundles []binaries.Archive

	for _, name := range []string{"kubectl-cert_manager", "cmctl"} {
		ctlA := s.ArtifactsOfKind(name)
		for _, a := range ctlA {
			f, err := downloadStagedArtifact(ctx, &a)
			if err != nil {
				return nil, fmt.Errorf("failed to download %q: %w", a.Metadata.Name, err)
			}

			binaryArchive := binaries.NewArchive(name, f.Name(), a.Metadata.OS, a.Metadata.Architecture, a.Metadata.Name)

			log.Printf("Found %s CLI binary archive for os=%s, arch=%s", name, binaryArchive.OS(), binaryArchive.Architecture())

			binaryBundles = append(binaryBundles, *binaryArchive)
		}
	}

	return binaryBundles, nil
}

func unpackCtlFromBazelRelease(ctx context.Context, s *Staged) ([]binaries.Archive, error) {
	// Example layout of a bazel ctl archive, containing another embedded archive
	// cert-manager-cmctl-linux-amd64.tar.gz
	//   ├── cmctl-linux-amd64.tar.gz
	//   │   ├── cmctl
	//   │   └── LICENSES
	//   └── version

	var binaryBundles []binaries.Archive

	for _, name := range []string{"kubectl-cert_manager", "cmctl"} {
		ctlA := s.ArtifactsOfKind(name)
		for _, a := range ctlA {
			dir, err := extractStagedArtifactToTempDir(ctx, &a)
			if err != nil {
				return nil, err
			}

			binaryArchives, err := recursiveFindWithExt(dir, ".gz")
			if err != nil {
				return nil, err
			}

			for _, archive := range binaryArchives {
				binaryArchive := binaries.NewArchive(name, archive, a.Metadata.OS, a.Metadata.Architecture, a.Metadata.Name)

				log.Printf("Found %s CLI binary archive for os=%s, arch=%s", name, binaryArchive.OS(), binaryArchive.Architecture())

				binaryBundles = append(binaryBundles, *binaryArchive)
			}
		}
	}

	return binaryBundles, nil
}

func unpackImages(ctx context.Context, artifacts []StagedArtifact, trimSuffix string) (map[string][]*images.Tar, error) {
	// tarBundles is a map from component name to slices of images.Tar
	tarBundles := make(map[string][]*images.Tar)

	for _, a := range artifacts {
		// each server bundle is a .tar.gz file which looks like this:
		// cert-manager-server-<os>-<arch>
		// ├── LICENSES
		// ├── server
		// │   └── images
		// │       ├── acmesolver.docker_tag
		// │       ├── acmesolver.tar
		// │       ├── cainjector.docker_tag
		// │       ├── cainjector.tar
		// │       ├── controller.docker_tag
		// │       ├── controller.tar
		// │       ├── ctl.docker_tag
		// │       ├── ctl.tar
		// │       ├── webhook.docker_tag
		// │       └── webhook.tar
		// └── version

		// Each .tar file is a separate container

		dir, err := extractStagedArtifactToTempDir(ctx, &a)
		if err != nil {
			return nil, err
		}

		// imageArchives becomes a list of each container packaged in this artifact
		imageArchives, err := recursiveFindWithExt(dir, ".tar")
		if err != nil {
			return nil, err
		}

		for _, archive := range imageArchives {
			imageTar, err := images.NewTar(archive, a.Metadata.OS, a.Metadata.Architecture)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect image tar at path %q: %w", archive, err)
			}

			baseName := filepath.Base(archive)
			componentName := strings.TrimSuffix(baseName[:len(baseName)-len(filepath.Ext(baseName))], trimSuffix)
			log.Printf("Found image for component %q with name %q", componentName, imageTar.RawImageName())
			tarBundles[componentName] = append(tarBundles[componentName], imageTar)
		}
	}

	return tarBundles, nil
}

// recursiveFindWithExt will recursively Walk a directory searching for files
// that have the given extension and return their path.
func recursiveFindWithExt(path, ext string) ([]string, error) {
	var paths []string
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ext {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, err
	}
	return paths, nil
}

func manifestArtifactForStaged(s *Staged) (*StagedArtifact, error) {
	artifacts := s.ArtifactsOfKind("manifests")
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("cannot find 'manifests' artifact in staged release %q", s.Name())
	}
	if len(artifacts) > 1 {
		return nil, fmt.Errorf("found multiple 'manifests' artifacts in staged release %q", s.Name())
	}
	return &artifacts[0], nil
}

func downloadStagedArtifact(ctx context.Context, a *StagedArtifact) (*os.File, error) {
	f, err := os.CreateTemp("", "temp-artifact-")
	if err != nil {
		f.Close()
		return nil, err
	}

	r, err := a.ObjectHandle.NewReader(ctx)
	if err != nil {
		f.Close()
		return nil, err
	}

	defer r.Close()

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return nil, err
	}

	// flush data to disk
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}

	// seek back to the start of the file so it can be read again
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}

	// validate the sha256sum
	downloadedSum, err := sha256SumFile(f.Name())
	if err != nil {
		f.Close()
		return nil, err
	}

	if downloadedSum != a.Metadata.SHA256 {
		f.Close()
		return nil, fmt.Errorf("artifact %q has a mismatching checksum - refusing to extract", a.Metadata.Name)
	}

	log.Printf("Validated sha256sum of artifact %q: %s", a.Metadata.Name, downloadedSum)

	return f, nil
}

func extractStagedArtifactToTempDir(ctx context.Context, a *StagedArtifact) (string, error) {
	dest, err := os.MkdirTemp("", "extracted-artifact-")
	if err != nil {
		return "", err
	}
	log.Printf("Extracting artifact file: %q", a.Metadata.Name)
	return dest, extractStagedArtifact(ctx, a, dest)
}

func extractStagedArtifact(ctx context.Context, a *StagedArtifact, dest string) error {
	// download the file to disk first
	f, err := downloadStagedArtifact(ctx, a)
	if err != nil {
		return err
	}

	defer f.Close()

	if filepath.Ext(a.Metadata.Name) == ".zip" {
		return zip.Unzip(dest, f)
	}

	return tar.UntarGz(dest, f)
}

func sha256SumFile(filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
