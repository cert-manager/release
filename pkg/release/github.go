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
	"encoding/json"
	"fmt"
	"net/http"
)

// LookupBranchRef will lookup the git commit ref of the HEAD of the branch
// in the given repository.
// It does this by querying the GitHub v3 API at:
// https://api.github.com/repos/{org}/{repo}/git/ref/heads/{branch}
func LookupBranchRef(org, repo, branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/%s", org, repo, branch)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	type payload struct {
		Object struct {
			SHA string
		}
	}
	p := payload{}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return "", err
	}

	return p.Object.SHA, nil
}
