# README

This cloudbuild config allows you to run the integration tests for the GitHub implementation of the Helm RepositoryManager
in cloudbuild using the [jetstack-release-bot][] credentials.

Use this to verify that the [jetstack-release-bot][] user has permission to create branches and PRs in the [jetstack-charts repo][].

Run the tests using [gcloud builds submit][] :

```sh
$ gcloud builds submit gcb/test/helm/ --config=gcb/test/helm/cloudbuild.yaml \
    --substitutions=_RELEASE_REPO_REF=<commit-sha>
```

`_RELEASE_REPO_REF` must be an immutable commit SHA of this repository. The build
checks out that ref and runs the tests with credentials in scope, so it must not be
left as a mutable ref such as `master`; the default value is a placeholder that fails
the build closed if no SHA is supplied.

[jetstack-release-bot]: https://github.com/jetstack-release-bot
[jetstack-charts repo]: https://github.com/jetstack/jetstack-charts
[gcloud builds submit]: https://cloud.google.com/sdk/gcloud/reference/builds/submit
