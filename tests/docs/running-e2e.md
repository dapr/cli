# Running E2E tests

## Prerequisites

Build the CLI first. The E2E tests exercise the built binary, and expect it in
`./dist/{os}_{arch}/release/`.

```bash
make build
```

Some standalone tests (for example `tests/e2e/standalone/uninstall_test.go`) pin the
Dapr runtime and dashboard versions they install, and need these set as environment
variables:

```bash
export DAPR_RUNTIME_PINNED_VERSION=<version>
export DAPR_DASHBOARD_PINNED_VERSION=<version>
```

Without them, the test run fails with an error naming the missing variable.

## Running the tests

E2E tests are behind the `e2e` build tag, so `go test` needs it explicitly:

```bash
go test --tags=e2e ./tests/e2e/standalone/... -run NameOfTest
```

There are also `make` targets that build the CLI and run a full E2E suite:

```bash
make e2e-build-run-sh     # self-hosted
make e2e-build-run-k8s    # Kubernetes
make e2e-build-run-upgrade
```

See the `test-e2e-*` and `e2e-build-run-*` targets in the [Makefile](../../Makefile) for
the exact test tags and paths each one uses.
