
# Release Guide

This document describes how to release Dapr CLI along with associated artifacts.

## Prerequisites

Only the repository maintainers and release team are allowed to execute the below steps.

## Pre-release build

Pre-release build will be built from `release-<major>.<minor>` branch and versioned by git version tag suffix e.g. `-rc.0`, `-rc.1`, etc. This build is not released to users who use the latest stable version.

**Pre-release process**
1. Create a PR to update the `Dapr runtime` and `Dapr dashboard` pre-release versions in workflow and tests files wherever applicable, and merge the PR to master branch.

2. Create branch `release-<major>.<minor>` from master and push the branch. e.g. `release-1.11`. You can use the github web UI to create the branch or use the following command.

```sh
$ git checkout master && git reset --hard upstream/master && git pull upstream master
$ git checkout -b release-1.11
$ git push upstream release-1.11
```
3. Add pre-release version tag(with suffix -rc.0 e.g. v1.11.0-rc.0) and push the tag.

```sh
$ git tag "v1.11.0-rc.0" -m "v1.11.0-rc.0"
$ git push upstream v1.11.0-rc.0

```
4. CI creates the new build artifacts.
5. Test and validate the functionalities with the specific version.
6. If there are regressions and bugs, fix them in release-* branch. e.g `release-1.11` branch.
7. Create new pre-release version tag(with suffix -rc.1, -rc.2, etc).
8. Repeat from 5 to 7 until all bugs are fixed.


## Release the stable version to users

> Note: Make sure stable version of `dapr runtime` and `dapr dashboard` is released before releasing the CLI and update their references in workflow and tests files wherever applicable.

Once all bugs are fixed we are ready to release the stable version to users. We will create a new git version tag (without the suffix -rc.x e.g. v1.11.0) and push the tag. CI will create the new build artifacts and release them to users.

## Release Patch version

We will work on the existing `release-<major>.<minor>` branch to release patch version. Once all bugs are fixed, we will add new patch version tag, such as `v1.11.1-rc.0`. After verifying the fixes on this pre-release, we create a new git version tag such as `v1.11.1` and push the tag. CI will create the new build artifacts and release them to users.