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

// Tar is used for accessing and interacting with a tar wth a binary file stored on disk.
type Tar struct {
	// path is the path to the tar file containing the binary on disk.
	path string

	// os and arch of the binary. This must be provided at instantiation time
	// and cannot be determined by inspecting the tar file.
	os, arch string
}

func NewFile(path, osStr, arch string) (*Tar, error) {
	return &Tar{
		path: path,
		os:   osStr,
		arch: arch,
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
