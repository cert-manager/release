package helm

import (
	"context"

	"github.com/google/go-github/v35/github"
)

// GitHubClient provides the minimum necessary GitHub API client methods
// required by the RepositoryManager.
type GitHubClient struct {
	GitClient
	PullRequestClient
	RepositoriesClient
	UsersClient
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
	// GetPermissionLevel retrieves the specific permission level a collaborator has for a given repository.
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/repos/#get-repository-permissions-for-a-user
	GetPermissionLevel(ctx context.Context, owner, repo, user string) (*github.RepositoryPermissionLevel, *github.Response, error)
}

type UsersClient interface {
	// Get fetches a user. Passing the empty string will fetch the authenticated
	// user.
	//
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/users/#get-the-authenticated-user
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/users/#get-a-user
	Get(ctx context.Context, user string) (*github.User, *github.Response, error)
}
