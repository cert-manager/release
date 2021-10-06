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

package release

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestOSListFromString(t *testing.T) {
	tests := map[string]struct {
		input        string
		expectedOSes []string
		expectErr    bool
	}{
		"valid manually specified OSes": {
			input:        "linux, Windows",
			expectedOSes: []string{"linux", "windows"},
			expectErr:    false,
		},
		"valid asterisk": {
			input: "*",
			// this test will break if we add more OSes but that'll be rare
			expectedOSes: []string{"linux", "windows", "darwin"},
			expectErr:    false,
		},
		"valid with deduping": {
			input:        "linux, Windows, LINUX",
			expectedOSes: []string{"linux", "windows"},
			expectErr:    false,
		},
		"no OSes should error": {
			input:        "",
			expectedOSes: nil,
			expectErr:    true,
		},
		"individual invalid OS should error": {
			input:        "templeos",
			expectedOSes: nil,
			expectErr:    true,
		},
		"invalid OS among valid should error": {
			input:        "linux,templeos,windows",
			expectedOSes: nil,
			expectErr:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			outputSet, err := OSListFromString(test.input)

			if test.expectErr {
				if err == nil {
					t.Errorf("expectErr=%v, err=%v", test.expectErr, err)
				}

				return
			}

			output := outputSet.List()

			sort.Strings(test.expectedOSes)
			sort.Strings(output)

			if !reflect.DeepEqual(test.expectedOSes, output) {
				t.Errorf("expected %#v but got %#v", test.expectedOSes, output)
				return
			}
		})
	}
}

func TestArchListFromString(t *testing.T) {
	tests := map[string]struct {
		input          string
		inputOSes      []string
		expectedArches []string
		expectErr      bool
	}{
		"valid manually specified arches": {
			input:          "amd64, s390X, ARM64  ",
			inputOSes:      []string{"linux"},
			expectedArches: []string{"amd64", "arm64", "s390x"},
			expectErr:      false,
		},
		"valid asterisk": {
			input:     "*",
			inputOSes: []string{"linux"},
			// this test will break if we add more arches but that'll be rare
			expectedArches: []string{"amd64", "arm", "arm64", "ppc64le", "s390x"},
			expectErr:      false,
		},
		"valid manually specified arches with deduping": {
			input:          "amd64, s390X, ARM64  ,AMd64",
			inputOSes:      []string{"linux"},
			expectedArches: []string{"amd64", "arm64", "s390x"},
			expectErr:      false,
		},
		"no arches should error": {
			input:          "",
			inputOSes:      []string{"linux"},
			expectedArches: nil,
			expectErr:      true,
		},
		"individual invalid arch should error": {
			input:          "notanarch",
			inputOSes:      []string{"linux"},
			expectedArches: nil,
			expectErr:      true,
		},
		"invalid arch among valid should error": {
			input:          "arm64,notanarch,amd64",
			inputOSes:      []string{"linux"},
			expectedArches: nil,
			expectErr:      true,
		},
		"invalid arch for OS should error": {
			// will break if we add support for s390x on windows, but that's not likely!
			input:          "s390x",
			inputOSes:      []string{"windows"},
			expectedArches: nil,
			expectErr:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			outputSet, err := ArchListFromString(test.input, sets.NewString(test.inputOSes...))

			if test.expectErr {
				if err == nil {
					t.Errorf("expectErr=%v, err=%v", test.expectErr, err)
				}

				return
			}

			output := outputSet.List()

			sort.Strings(test.expectedArches)
			sort.Strings(output)

			if !reflect.DeepEqual(test.expectedArches, output) {
				t.Errorf("expected %#v but got %#v", test.expectedArches, output)
				return
			}
		})
	}
}
