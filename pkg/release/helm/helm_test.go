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

package helm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/cert-manager/release/pkg/release/manifests"
)

// TestIntegration is designed to be run manually to test the
// GitHubRepositoryManager.Publish function with a real GitHub repository.
// You will need a GitHub personal access token with at least `repo` scope,
// and the associated user must have permission to create branches and PRs in
// the target repository.
// You MUST configure the GITHUB_TOKEN and GITHUB repository settings as
// environment variables. E.g.
//
// export GITHUB_TOKEN=ghp_<redacted>
// export HELM_GITHUB_OWNER=wallrj
// export HELM_GITHUB_REPO=my-charts
// export HELM_GITHUB_SOURCE_BRANCH=main
func TestIntegration(t *testing.T) {

	ctx := context.TODO()

	config := map[string]string{
		"GITHUB_TOKEN":              "",
		"HELM_GITHUB_OWNER":         "",
		"HELM_GITHUB_REPO":          "",
		"HELM_GITHUB_SOURCE_BRANCH": "",
	}

	for key := range config {
		val := os.Getenv(key)
		if val == "" {
			t.Skipf("%q environment variable not set", key)
		}
		config[key] = val
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config["GITHUB_TOKEN"]},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	r := NewGitHubRepositoryManager(
		&GitHubClient{
			GitClient:          githubClient.Git,
			PullRequestClient:  githubClient.PullRequests,
			RepositoriesClient: githubClient.Repositories,
			UsersClient:        githubClient.Users,
		},
		config["HELM_GITHUB_OWNER"],
		config["HELM_GITHUB_REPO"],
		config["HELM_GITHUB_SOURCE_BRANCH"],
	)

	t.Run("Check", func(t *testing.T) {
		err := r.Check(ctx)
		require.NoError(t, err)
	})

	t.Run("Publish", func(t *testing.T) {
		chart, err := manifests.NewChart("testdata/cert-manager-v0.1.0-test.1.tgz")
		require.NoError(t, err)
		fakeReleaseName := fmt.Sprintf("cert-manager-%s-%d", chart.Version(), time.Now().Unix())

		prURL, err := r.Publish(ctx, fakeReleaseName, *chart)
		require.NoError(t, err)

		expectedURLPattern := fmt.Sprintf(
			`https://github.com/%s/%s/pull/\d+`,
			config["HELM_GITHUB_OWNER"],
			config["HELM_GITHUB_REPO"],
		)
		require.Regexp(t, expectedURLPattern, prURL)
	})
}
