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

package images

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cert-manager/release/pkg/release/tar"
)

// Tar is used for accessing and interacting with a .tar file containing
// a docker image stored on disk.
type Tar struct {
	// path is the path to the tar file containing the image on disk.
	path string

	// os and arch of the image. This must be provided at instantiation time
	// and cannot be determined by inspecting the tar file.
	os, arch string

	// imageName is the name of the image stored in the tar file, extracted by
	// reading the manifest.json file in the archive.
	imageName string

	// imageArchitecture is the name of the architecture stored in the tar file, extracted by
	// reading the manifest and config file in the archive.
	imageArchitecture string
}

type TarInterface interface {
	Architecture() string
	ImageArchitecture() string
	ImageName() string
	ImageTag() string
	Filepath() string
	OS() string
}

func NewTar(path, osStr, arch string) (*Tar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metaBytes, err := tar.ReadSingleFile("manifest.json", f)
	if err != nil {
		return nil, err
	}

	// create a var to decode the docker image manifest.json into
	// this is an extremely stripped back version of the full docker manifest
	// metadata specification.
	var metas = []struct {
		RepoTags []string `json:"RepoTags"`
		Config   string   `json:"Config"`
	}{}
	if err := json.Unmarshal(metaBytes, &metas); err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, fmt.Errorf("could not find any image entries in image tar metadata.json file")
	}
	if len(metas) > 1 {
		return nil, fmt.Errorf("found multiple image entries in image tar metadata.json file")
	}
	meta := metas[0]
	if len(meta.RepoTags) == 0 {
		return nil, fmt.Errorf("could not find any image tag entries in image tar metadata.json file")
	}
	if len(meta.RepoTags) > 1 {
		return nil, fmt.Errorf("found multiple image tag entries in image tar metadata.json file")
	}
	imageName := meta.RepoTags[0]

	var config = struct {
		Architecture string `json:"architecture"`
	}{}
	if meta.Config == "" {
		return nil, fmt.Errorf("could not find any config entries in image tar metadata.json file")
	}

	configBytes, err := tar.ReadSingleFile(meta.Config, f)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}

	return &Tar{
		path:              path,
		os:                osStr,
		arch:              arch,
		imageName:         imageName,
		imageArchitecture: config.Architecture,
	}, nil
}

func (i *Tar) Filepath() string {
	return i.path
}

func (i *Tar) OS() string {
	return i.os
}

func (i *Tar) Architecture() string {
	return i.arch
}

func (i *Tar) ImageName() string {
	return i.imageName
}

func (i *Tar) ImageArchitecture() string {
	return i.imageArchitecture
}

func (i *Tar) ImageTag() string {
	s := strings.Split(i.imageName, ":")
	if len(s) < 2 {
		return "latest"
	}
	return s[1]
}
