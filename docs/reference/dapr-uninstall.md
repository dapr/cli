# dapr uninstall

## Description

Removes a Dapr installation

## Usage

```bash
dapr uninstall [flags]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| `--all` | | `false` | Remove Redis, Zipkin containers in addition to actor placement container. Remove default dapr dir located at `$HOME/.dapr or %USERPROFILE%\.dapr\`. |
| `--help`, `-h` | | | Help for uninstall |
| `--kubernetes`, `-k` | | `false` | Uninstall Dapr from a Kubernetes cluster |
| `--network` | `DAPR_NETWORK` | | The Docker network from which to remove the Dapr runtime |
| `--runtime-version` | | `latest` | The version of the Dapr runtime to uninstall, for example: `v0.1.0-alpha` (Kubernetes mode only) |
| `--install-path` |  | `/usr/local/bin` for Linux/Mac and `C:\dapr` for Windows | The optional location to uninstall Dapr from. Use if provided during `dapr init`|
