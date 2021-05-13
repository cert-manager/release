# README

This directory contains an empty Helm chart and a Helm package for use in Go tests.
The Helm chart and the Helm package were generated as follows:

```sh
$ helm version
version.BuildInfo{Version:"v3.5.2", GitCommit:"167aac70832d3a384f65f9745335e9fb40169dc2", GitTreeState:"dirty", GoVersion:"go1.15.7"}

$ helm create ./pkg/release/helm/testdata/cert-manager

$ helm package ./pkg/release/helm/testdata/cert-manager --destination pkg/release/helm/testdata/ --app-version v0.1.0-test.1 --version v0.1.0-test.1
```
