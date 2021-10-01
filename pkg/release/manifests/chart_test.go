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

import "testing"

func TestNewChart(t *testing.T) {
	tests := map[string]struct {
		path    string
		hasProv bool
	}{
		"can load a chart with prov": {
			path:    "testdata/withprov/cert-manager.tgz",
			hasProv: true,
		},
		"can load a chart without prov": {
			path:    "testdata/withoutprov/cert-manager.tgz",
			hasProv: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			chart, err := NewChart(test.path)
			if err != nil {
				t.Errorf("got an error but didn't expect one: %v", err)
			}

			if (chart.ProvPath() == nil) == test.hasProv {
				t.Errorf("wanted hasProv=%v but got %v", test.hasProv, (chart.ProvPath() != nil))
			}
		})
	}
}
