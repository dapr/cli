
# Release Guide

This document describes how to release Dapr CLI along with associated artifacts.

## Prerequisites

Only the repository maintainers and release team are allowed to execute the below steps.

## Pre-release build

### Dapr CLI manual release process

Pre-release build will be built from `release-<major>.<minor>` branch and versioned by git version tag suffix e.g. `-rc.0`, `-rc.1`, etc. This build is not released to users who use the latest stable version.

**Pre-release process**
1. Create a PR to update the `Dapr runtime` pre-release version in workflow and tests files wherever applicable, and merge the PR to master branch. For example, please take a look at this PR - https://github.com/dapr/cli/pull/1720, which updates `DAPR_RUNTIME_PINNED_VERSION` in `kind_e2e.yaml` and `self_hosted_e2e.yaml` and the upgrade versions in `tests/e2e/upgrade/upgrade_test.go`. Please note that this PR is just for reference and only reflects the absolute minimum number of file changes. These versions update could lead to some other changes also. Also update the Go module dependencies to the matching pre-release versions, see [Go module dependencies](#go-module-dependencies).

2. Create branch `release-<major>.<minor>` from master and push the branch. e.g. `release-1.11`. You can use the github web UI to create the branch or use the following command.

```sh
$ git checkout master && git reset --hard upstream/master && git pull upstream master
$ git checkout -b release-1.11
$ git push upstream release-1.11
```
3. Add pre-release version tag (with suffix -rc.0 e.g. v1.11.0-rc.0) and push the tag.

```sh
$ git tag "v1.11.0-rc.0" -m "v1.11.0-rc.0"
$ git push upstream v1.11.0-rc.0

```
4. CI creates the new build artifacts.
5. Test and validate the functionalities with the specific version.
6. If there are regressions and bugs, fix them on master and backport them to the release-* branch, e.g. `release-1.11`. See [Backporting fixes](#backporting-fixes).
7. Create new pre-release version tag (with suffix -rc.1, -rc.2, etc).
8. Repeat from 5 to 7 until all bugs are fixed.

### Dapr CLI automated release process

The Dapr CLI automated release process uses the `Create a release` workflow (`.github/workflows/create-release.yaml`). Run it manually from the GitHub Actions UI with the `rel_version` input, e.g. `1.11.0-rc.0`, `1.11.0` or `1.11.1`.

The workflow creates the `release-<major>.<minor>` branch if it doesn't exist yet, tags the version on it, and triggers the Dapr CLI build. It fails if the tag already exists. `-alpha` versions are tagged on master instead. Use the same workflow for pre-release, stable and patch versions. For a stable version, ensure the new release is set to `latest` and not a `pre-release`.


## Release the stable version to users

> **Note**: Make sure a stable version of `dapr runtime` is released before releasing the CLI and update its references in workflow and tests files wherever applicable. Also update the Go module dependencies to the stable versions, see [Go module dependencies](#go-module-dependencies).

Once all bugs are fixed, stable version can be released. Create a new git version tag (without the suffix -rc.x e.g. v1.11.0) and push the tag. CI will create the new build artifacts and release them.

## Release Patch version

Patch fixes go to master first and are backported to the existing `release-<major>.<minor>` branch, see [Backporting fixes](#backporting-fixes). Once all bugs are fixed, create a new patch version tag, such as `v1.11.1-rc.0`. After verifying the fixes on this pre-release, create a new git version tag such as `v1.11.1` and push the tag. CI will create the new build artifacts and release them.

## Go module dependencies

The CLI imports packages from `github.com/dapr/dapr`, so its Go modules have to match the Dapr runtime it is released with. Update them before the first pre-release (to the matching `-rc` versions) and again before the stable release (to the stable versions), so the e2e tests run against the combination that ships.

The Dapr Go modules depend on each other, and the CLI can only be updated after all of them are tagged:

1. `github.com/dapr/kit`
2. `github.com/dapr/durabletask-go` and `github.com/dapr/components-contrib`
3. `github.com/dapr/dapr`
4. `github.com/dapr/go-sdk`
5. `github.com/dapr/cli`

Update `dapr/dapr` and `components-contrib` in a single command, then tidy:

```sh
$ go get github.com/dapr/dapr@v1.11.0 github.com/dapr/components-contrib@v1.11.0
$ go mod tidy
```

`dapr/go-sdk` has its own version numbers. Update it to the release built against the same `dapr/dapr` version, e.g. go-sdk `v1.16.0` for Dapr `v1.19.0`.

`dapr/dapr` pins `dapr/kit` and `dapr/durabletask-go`, so `go get` updates those too. Updating `components-contrib` on its own can break the build, because the `dapr/dapr` version still in `go.mod` can import contrib packages that the new contrib version removed.

The update can raise the `go` directive in `go.mod`. The CI workflows read the Go version from `go.mod` (`go-version-file`), so they don't need a separate change.

Before the stable release, check that `go.mod` on the release branch has no previous-minor or `-rc` versions of these modules:

```sh
$ grep -E 'github.com/dapr/(dapr|components-contrib|go-sdk|kit|durabletask-go) ' go.mod
```

## Backporting fixes

Changes for a release branch, including the dependency updates above, are merged to master first. To backport a PR, add the `backport release-<major>.<minor>` label to it, e.g. `backport release-1.11`. When the PR is merged (or when the label is added to an already merged PR), dapr-bot cherry-picks it and opens a `[Backport release-<major>.<minor>] ...` PR against the release branch. If the cherry-pick has conflicts, resolve them in the backport PR.

## Project Release Guidelines

See [this document](https://github.com/dapr/community/blob/master/release-process.md) for the project's release process and guidelines.