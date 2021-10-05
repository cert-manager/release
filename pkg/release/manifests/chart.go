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

package manifests

import (
	"compress/gzip"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"k8s.io/utils/pointer"

	"github.com/cert-manager/release/pkg/release/tar"
)

type Chart struct {
	path     string
	provPath *string

	meta chartMeta
}

type chartMeta struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

// NewChart tries to read and extract metadata from a chart at `path`. It also searches
// `path`+".prov" to check for a signature, and stores the path to a signature if found.
func NewChart(path string) (*Chart, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	defer gzr.Close()

	chartMetaBytes, err := tar.ReadSingleFile("cert-manager/Chart.yaml", gzr)
	if err != nil {
		return nil, err
	}

	meta := chartMeta{}
	if err := yaml.Unmarshal(chartMetaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to decode chart metadata: %w", err)
	}

	provPath := pointer.String(path + ".prov")

	_, err = os.Stat(*provPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check for %q: %w", *provPath, err)
		}

		provPath = nil
	}

	return &Chart{
		path: path,
		meta: meta,

		provPath: provPath,
	}, nil
}

func (c *Chart) PackageFileName() string {
	return fmt.Sprintf("%s-%s.tgz", c.meta.Name, c.Version())
}

func (c *Chart) Path() string {
	return c.path
}

func (c *Chart) ProvPath() *string {
	return c.provPath
}

func (c *Chart) Version() string {
	return c.meta.Version
}

func (c *Chart) AppVersion() string {
	return c.meta.AppVersion
}
