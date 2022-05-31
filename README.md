<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# cert-manager Release Tooling

This repository contains release tooling for the cert-manager project.

NB: The most up-to-date release process is documented on the [cert-manager website](https://cert-manager.io/docs/contributing/release-process/).
If you're trying to do a cert-manager release, you should start on the website. The docs
here are mostly intended for people developing cert-manager tooling.

## cmrel

`cmrel` is a small tool to help with building and releasing cert-manager.

The key commands are:

- `cmrel makestage` - Build and stage a cert-manager release from a given git ref
- `cmrel publish` - Publish a previously staged release

## makestage

`cmrel makestage` is a totally minimal wrapper for building a full cert-manager release.

The actual commands which are run are defined entirely in a Makefile in the cert-manager repo. This command is essentially
just glue to call that Makefile in Google Cloud Build, and then to copy the resulting files to GCS.

The only argument which would normally be required is `--git-ref` which specifies the git ref to check out for the cert-manager
repo. This might be a commit, a tag, or a branch. Usually for a release, a tag would be specified.

An example invocation might be:

```console
$ cmrel makestage --ref master
... lots of output ...
```

## publish

`cmrel publish` takes a staged release from Google Cloud Storage, validates it, and then
pushes it out to public facing locations including Helm repos, container registries, and GitHub
releases.

```console
$ cmrel publish \
    --release-name v0.14.0-f6da9c76877551ef32503b17189bb178501f59a7 \
    --nomock
```

# Legacy Docs

All below docs are legacy and are preserved only for the transition from bazel to make.

## Control Flow During a Release

`cmrel` is used in various places - including by itself - to carry out a release.

The process can be summarised roughly as follows:

- A developer calls `cmrel stage` on their machine which triggers the "stage" [GCB job](./gcb/stage/cloudbuild.yaml)
- The "stage" GCB job calls `cmrel gcb stage` which creates cert-manager artifacts
- A developer calls `cmrel publish` on their machine, which triggers the "publish" [GCB job](./gcb/publish/cloudbuild.yaml)
- The "publish" GCB job calls `cmrel gcb publish` which uploads the artifacts wherever they need to be published

## cmrel

cmrel is the central hub for release managers interacting with the release
process.

It has 3 primary functions:

* Staging new releases
* Listing staged releases
* Publishing a staged release

### Creating an Official Release

> *WARNING*: following these steps exactly will push out a *public facing release*!
> Please use this as an example *only*.

In this example, we're going to build, stage and publish a full official
release from source.

For example purposes, we'll:

1) Use the `release-0.14` branch in cert-manager as the 'source'
2) Create the release with version `v0.14.0`

#### Step 1 - stage the release

'Staging' a release is the process of:

* Cloning the cert-manager repository
* Running a 'release build'
* Storing build release artifacts & associated metadata in Google Cloud Storage

The `cmrel` tool provides a subcommand to start this process, `cmrel stage`.
Full usage information for `cmrel stage`:

```text
Flags:
      --branch string                 The git branch to build the release from. If --git-ref is not specified, the HEAD of this branch will be looked up on GitHub. (default "master")
      --bucket string                 The name of the GCS bucket to stage the release to. (default "cert-manager-release")
      --cloudbuild string             The path to the cloudbuild.yaml file used to perform the cert-manager crossbuild. The default value assumes that this tool is run from the root of the release repository. (default "./gcb/stage/cloudbuild.yaml")
      --git-ref string                The git commit ref of cert-manager that should be staged.
  -h, --help                          help for stage
      --org string                    Name of the GitHub org to fetch cert-manager sources from. (default "cert-manager")
      --project string                The GCP project to run the GCB build jobs in. (default "cert-manager-release")
      --published-image-repo string   The docker image repository set when building the release. (default "quay.io/jetstack")
      --release-version string        Optional release version override used to force the version strings used during the release to a specific value.
      --repo string                   Name of the GitHub repo to fetch cert-manager sources from. (default "cert-manager")
```

The default values here are optimised for pushing an official release. If you
are intending to publish this release to your own project/namespace, you should
be sure to change the `--published-image-repo` flag accordingly.

If you are not a 'cert-manager release manager', you will also need to use an
alternative `--project` and `--bucket` flag that you have sufficient permission
to publish to.

We'll run `cmrel stage` below to start a GCB job to stage the release:

```console
$ cmrel stage \
    --branch release-0.14 \
    --release-version v0.14.0
```

This will trigger a job to run on GCB which will build and push the release
artifacts to the staging bucket.

After executing, you should see a message indicating where you can visit to
follow along and view logs to track the build progress.

Once complete, `cmrel stage` should exit successfully.

#### Step 2 - Listing Staged Builds

After a build has been staged, it's important that you verify the release has
been published to the bucket as expected.

The `cmrel staged` command will print a simple list of releases that have been
staged to the release bucket.

Full usage information for `cmrel staged`:

```text
Flags:
      --bucket string            The name of the GCS bucket containing the staged releases. (default "cert-manager-release")
      --git-ref string           Optional specific git reference to list staged releases for - if specified, --release-version must also be specified.
  -h, --help                     help for staged
      --release-type string      The type of release to list - usually one of 'release' or 'devel' (default "release")
      --release-version string   Optional release version override used to force the version strings used during the release to a specific value.
```

Running it will print a list of staged releases:

```console
$ cmrel staged
...
NAME                                             VERSION DATE
v0.14.0-f6da9c76877551ef32503b17189bb178501f59a7 v0.14.0 UNKNOWN
```

Here we can see a single release in the bucket, version `v0.14.0`.
The git commit ref is also included as part of the name, so you can be sure
that the correct revision of cert-manager has in fact been built.

Once you have found the release you wish to stage in this list, make a note of
the release's `name` and proceed to step 3!

#### Step 3 - Publishing a Staged Release

Once a release has been staged into the release bucket and we've verified it
has been built from the correct revision of cert-manager, we are now ready to
trigger the publishing stage of the release.

In this step, the staged release is fetched from Google Cloud Storage,
validated, and then pushed out to public facing locations.

```console
$ cmrel publish \
    --release-name v0.14.0-f6da9c76877551ef32503b17189bb178501f59a7 \
    --nomock
```

If you do not specify the `--nomock` flag, `cmrel` will *not* push any
artifacts and will only fetch and validate the release before exiting.

The final stage of this step is to create a GitHub release in the cert-manager
repository, as well as uploading 'static manifests' to the release.
To allow you to update the release with proper release notes
*before publishing*, `cmrel` will mark the created release as a **DRAFT**.
You must then edit the release to include the appropriate release notes, and
then hit 'Publish'!

If you are intending to publish to your own, private release buckets (i.e. to
test this whole workflow, or for creating internal releases) you should be sure
to set the following flags when calling `cmrel publish`:

```text
    --published-image-repo='quay.io/mycompany' # prefix for images, e.g. 'quay.io/mycompany'
    --published-helm-chart-bucket='mycompany-helm-charts' # name of the GCS bucket where the built Helm chart should be stored
    --published-github-org='mycompany' # name of the GitHub org containing the repo that will be tagged at the end
```

### Development

#### Creating Development Builds

By default the artifacts created during a release process are pushed to `cert-manager-release` bucket at `/stage/gcb/release` path.
It is also possible to create a 'development' build by skipping the `--release-version` flag on `cmrel stage` command. This will result in the build artifacts being pushed to `cert-manager-release` bucket at `/stage/gcb/devel` path.

If you have made some local changes to this tool and want to create a 'devel' build to test them, be mindful that the Google Cloud Build triggered by running `cmrel stage` clones this repository from GitHub and runs its own `cmrel` commands. You can modify the [Cloud Build config](https://github.com/cert-manager/release/blob/master/gcb/stage/cloudbuild.yaml) to configure a different GitHub repository/branch.
