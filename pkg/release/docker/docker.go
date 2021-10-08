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

package docker

import (
	"context"

	"github.com/cert-manager/release/pkg/release/shell"
)

// Load runs 'docker load' against the named .tar file
func Load(ctx context.Context, path string) error {
	return shell.Command(ctx, "", "docker", "load", "-i", path)
}

// Push runs 'docker push' with the given image name
func Push(ctx context.Context, image string) error {
	return shell.Command(ctx, "", "docker", "push", image)
}

// CreateManifestList creates a docker manifest list; see the `docker manifest create`
// command's `--help` for more information
func CreateManifestList(ctx context.Context, name string, imageNames []string) error {
	args := append([]string{"manifest", "create", name}, imageNames...)
	return shell.Command(ctx, "", "docker", args...)
}

// AnnotateManifestList annotates a docker manifest list; see the `docker manifest annotate`
// command's `--help` for more information
func AnnotateManifestList(ctx context.Context, manifestName, imageName, os, arch, variant string) error {
	return shell.Command(ctx, "",
		"docker", "manifest", "annotate", manifestName, imageName,
		"--os", os,
		"--arch", arch,
		"--variant", variant,
	)
}

// PushManifestList pushes a docker manifest list; see the `docker manifest push`
// command's `--help` for more information
func PushManifestList(ctx context.Context, name string) error {
	return shell.Command(ctx, "", "docker", "manifest", "push", name)
}
