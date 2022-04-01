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

	// rawImageName is the name of the image stored in the tar file, extracted by
	// reading the manifest.json file in the archive. Not necessarily the name
	// which will be used to push the image; a constructed tag will be used
	// for that, and the image will be retagged before pushing.
	rawImageName string

	// PublishedTag is the tag which the image has been published under, which might
	// be different to the raw image name which was part of the release. This should
	// be set after an image has been re-tagged and pushed.
	PublishedTag string
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
	rawImageName := meta.RepoTags[0]
	return &Tar{
		path:         path,
		os:           osStr,
		arch:         arch,
		rawImageName: rawImageName,
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

func (i *Tar) RawImageName() string {
	return i.rawImageName
}

func (i *Tar) ImageTag() string {
	s := strings.Split(i.rawImageName, ":")
	if len(s) < 2 {
		return "latest"
	}
	return s[1]
}
