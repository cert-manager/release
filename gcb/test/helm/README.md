# README

This cloudbuild config allows you to run the integration tests for the GitHub implementation of the Helm RepositoryManager
in cloudbuild using the [jetstack-release-bot][] credentials.

Use this to verify that the [jetstack-release-bot][] user has permission to create branches and PRs in the [jetstack-charts repo][].

Run the tests using [gcloud builds submit][] :

```sh
$ gcloud builds submit gcb/test/helm/ --config=gcb/test/helm/cloudbuild.yaml
```

[jetstack-release-bot]: https://github.com/jetstack-release-bot
[jetstack-charts repo]: https://github.com/jetstack/jetstack-charts
[gcloud builds submit]: https://cloud.google.com/sdk/gcloud/reference/builds/submit
