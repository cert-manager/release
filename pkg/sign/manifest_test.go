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

package sign

import (
	"os"
	"testing"
)

func TestSetOwnerWritable(t *testing.T) {
	tests := map[string]struct {
		inputMode    os.FileMode
		expectedMode os.FileMode
	}{
		"already writable left unchanged": {
			inputMode:    0o777,
			expectedMode: 0o777,
		},
		"sets owner writable and leaves others unchanged": {
			inputMode:    0o500,
			expectedMode: 0o700,
		},
		"example permissions from bazel output": {
			inputMode:    0o555,
			expectedMode: 0o755,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			outputMode := setOwnerWritable(test.inputMode)

			if outputMode != test.expectedMode {
				t.Errorf("wanted output mode to be '0o%o' but got '0o%o'", test.expectedMode, outputMode)
			}
		})
	}
}
