package helm

import (
	"context"

	"github.com/google/go-github/v29/github"
)

// GitHubClient provides the minimum necessary GitHub API client methods
// required by the RepositoryManager.
type GitHubClient struct {
	GitClient
	PullRequestClient
	RepositoriesClient
}

type PullRequestClient interface {
	Create(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
}

type GitClient interface {
	GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
	CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
}

type RepositoriesClient interface {
	// CreateFile creates a new file in a repository at the given path and returns
	// the commit and file metadata.
	//
	// GitHub API docs: https://developer.github.com/v3/repos/contents/#create-a-file
	CreateFile(ctx context.Context, owner, repo, path string, opt *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error)
}
