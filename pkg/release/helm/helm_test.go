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

	for key, _ := range config {
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
