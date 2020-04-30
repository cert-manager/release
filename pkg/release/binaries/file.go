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

package binaries

// Tar is used for accessing and interacting with a binary file stored on disk.
type File struct {
	// path is the path to the tar file containing the image on disk.
	path string

	// os and arch of the image. This must be provided at instantiation time
	// and cannot be determined by inspecting the tar file.
	os, arch string
}

func NewFile(path, osStr, arch string) (*File, error) {
	return &File{
		path: path,
		os:   osStr,
		arch: arch,
	}, nil
}

func (i *File) Filepath() string {
	return i.path
}

func (i *File) OS() string {
	return i.os
}

func (i *File) Architecture() string {
	return i.arch
}
