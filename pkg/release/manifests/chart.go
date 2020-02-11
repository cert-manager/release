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

package manifests

import (
	"compress/gzip"
	"fmt"
	"os"

	"github.com/ghodss/yaml"

	"github.com/cert-manager/release/pkg/release/tar"
)

type Chart struct {
	path string
	meta chartMeta
}

type chartMeta struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

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

	return &Chart{
		path: path,
		meta: meta,
	}, nil
}

func (c *Chart) PackageFileName() string {
	return fmt.Sprintf("%s-%s.tgz", c.meta.Name, c.Version())
}

func (c *Chart) Path() string {
	return c.path
}

func (c *Chart) Version() string {
	return c.meta.Version
}

func (c *Chart) AppVersion() string {
	return c.meta.AppVersion
}
