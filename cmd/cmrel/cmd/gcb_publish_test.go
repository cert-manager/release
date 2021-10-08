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

package cmd

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func sortedSlice(s []string) []string {
	sort.Strings(s)
	return s
}

func TestCanonicalizeAndVerifyPublishActions(t *testing.T) {
	oneAction := allPublishActionNames()[0]
	twoAction := allPublishActionNames()[1]

	tests := map[string]struct {
		inputActions   []string
		expectedOutput []string
		expectErr      bool
	}{
		"basic case with '*'": {
			inputActions:   []string{"*"},
			expectedOutput: sortedSlice(allPublishActionNames()),
			expectErr:      false,
		},
		"basic case with all action names": {
			inputActions:   allPublishActionNames(),
			expectedOutput: sortedSlice(allPublishActionNames()),
			expectErr:      false,
		},
		"actions form a set": {
			inputActions:   []string{oneAction, oneAction},
			expectedOutput: []string{oneAction},
			expectErr:      false,
		},
		"explicitly listed actions": {
			inputActions:   []string{oneAction, twoAction},
			expectedOutput: sortedSlice([]string{oneAction, twoAction}),
			expectErr:      false,
		},
		"explicitly listed actions, different order, same output": {
			inputActions:   []string{twoAction, oneAction},
			expectedOutput: sortedSlice([]string{oneAction, twoAction}),
			expectErr:      false,
		},
		"action removal": {
			inputActions:   []string{oneAction, twoAction, "-" + oneAction},
			expectedOutput: []string{twoAction},
			expectErr:      false,
		},
		"invalid action should error": {
			inputActions:   []string{oneAction, "notanaction"},
			expectedOutput: nil,
			expectErr:      true,
		},
		"invalid action with removal should error": {
			inputActions:   []string{oneAction, twoAction, "-notanaction"},
			expectedOutput: nil,
			expectErr:      true,
		},
		"action cleanup": {
			inputActions:   []string{"   " + strings.ToUpper(oneAction) + "   "},
			expectedOutput: []string{oneAction},
			expectErr:      false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := canonicalizeAndVerifyPublishActions(test.inputActions)

			if (err != nil) != test.expectErr {
				t.Errorf("expectedErr=%v, err=%v", test.expectErr, err)
			}

			if err != nil {
				return
			}

			if !reflect.DeepEqual(output, test.expectedOutput) {
				t.Errorf("wanted %#v but got %#v", test.expectedOutput, output)
			}
		})
	}
}
