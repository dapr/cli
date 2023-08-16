
# dapr-cli (dapr-cli)

Install the Dapr CLI

## Example Usage

```json
"features": {
    "ghcr.io/dapr/cli/dapr-cli:0": {}
}
```

## Options

| Options Id | Description | Type | Default Value |
|-----|-----|-----|-----|
| version | Version of the Dapr CLI to install (or "latest") | string | latest |

## Using with `docker-in-docker` feature

Since the Dapr CLI requires Docker, an easy way to get started is to use the `docker-in-docker` feature. This will install a separate Docker daemon inside the container for `dapr` to use:

```jsonc
"features": {
    // Install the Dapr CLI
    "ghcr.io/dapr/cli/dapr-cli:0": {},
    // Enable Docker (via Docker-in-Docker)
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    // Alternatively, use Docker-outside-of-Docker (uses Docker in the host)
    //"ghcr.io/devcontainers/features/docker-outside-of-docker:1": {},
}
```

For more details on setting up a Dev Container with Dapr, see the [Developing Dapr applications with Dev Containers docs](https://docs.dapr.io/developing-applications/local-development/ides/vscode/vscode-remote-dev-containers/).

---

_Note: This file was auto-generated from the [devcontainer-feature.json](https://github.com/dapr/cli/blob/master/dev-container-feature/src/dapr-cli/devcontainer-feature.json).  Add additional notes to a `NOTES.md`._
