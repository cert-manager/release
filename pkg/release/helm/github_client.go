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

	"github.com/google/go-github/v80/github"
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
	CreateTree(ctx context.Context, owner string, repo string, baseTree string, entries []*github.TreeEntry) (*github.Tree, *github.Response, error)
	CreateCommit(ctx context.Context, owner string, repo string, commit *github.Commit) (*github.Commit, *github.Response, error)
	UpdateRef(ctx context.Context, owner string, repo string, ref *github.Reference, force bool) (*github.Reference, *github.Response, error)
	CreateBlob(ctx context.Context, owner string, repo string, blob *github.Blob) (*github.Blob, *github.Response, error)
}

type RepositoriesClient interface {
	// GetPermissionLevel retrieves the specific permission level a collaborator has for a given repository.
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/repos/#get-repository-permissions-for-a-user
	GetPermissionLevel(ctx context.Context, owner, repo, user string) (*github.RepositoryPermissionLevel, *github.Response, error)
	GetCommit(ctx context.Context, owner, repo, sha string) (*github.RepositoryCommit, *github.Response, error)
}

type UsersClient interface {
	// Get fetches a user. Passing the empty string will fetch the authenticated
	// user.
	//
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/users/#get-the-authenticated-user
	// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/users/#get-a-user
	Get(ctx context.Context, user string) (*github.User, *github.Response, error)
}
