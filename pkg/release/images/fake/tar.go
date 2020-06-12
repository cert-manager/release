package fake

import (
	"github.com/cert-manager/release/pkg/release/images"
)

// FakeImageTar is a fake of the image.Tar struct
type FakeImageTar struct {
	architecture      string
	imageArchitecture string
	imageName         string
	imageTag          string
	filePath          string
	os                string
}

// New gives a new FakeImageTar
func New(imageName, imageTag, filePath, os, architecture, imageArchitecture string) images.TarInterface {
	return &FakeImageTar{
		architecture:      architecture,
		imageArchitecture: imageArchitecture,
		imageName:         imageName,
		imageTag:          imageTag,
		filePath:          filePath,
		os:                os,
	}
}

func (f *FakeImageTar) Architecture() string {
	return f.architecture
}

func (f *FakeImageTar) ImageArchitecture() string {
	return f.imageArchitecture
}

func (f *FakeImageTar) ImageName() string {
	return f.imageName
}

func (f *FakeImageTar) ImageTag() string {
	return f.imageTag
}

func (f *FakeImageTar) OS() string {
	return f.os
}

func (f *FakeImageTar) Filepath() string {
	return f.filePath
}
