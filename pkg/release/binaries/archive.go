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

package binaries

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Archive is used for accessing and interacting with a tar wth a binary file stored on disk.
type Archive struct {
	// path points to the archive file containing the binary on disk.
	path string

	// os and arch of the binary. This must be provided at instantiation time
	// and cannot be determined by inspecting the archive file.
	os, arch string

	// name is the base name of the binary, (e.g. `cmctl`).
	name string

	// ext is the type of the archive, either ".zip" or ".tar.gz"
	ext string
}

func NewArchive(name, path, osStr, arch string, rawFilename string) *Archive {
	// default to .tar.gz, but change to zip if needed
	ext := ".tar.gz"

	rawExt := strings.ToLower(filepath.Ext(rawFilename))

	if rawExt == ".zip" {
		ext = rawExt
	}

	return &Archive{
		name: name,
		path: path,
		os:   osStr,
		arch: arch,
		ext:  ext,
	}
}

func (i *Archive) Filepath() string {
	return i.path
}

func (i *Archive) OS() string {
	return i.os
}

func (i *Archive) Architecture() string {
	return i.arch
}

func (i *Archive) Name() string {
	return i.name
}

func (i *Archive) Extension() string {
	return i.ext
}

func (i *Archive) ArtifactFilename() string {
	return fmt.Sprintf("%s-%s-%s%s", i.Name(), i.OS(), i.Architecture(), i.Extension())
}
