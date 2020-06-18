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
| `--kubernetes` | | `false` | Uninstall Dapr from a Kubernetes cluster |
| `--network` | `DAPR_NETWORK` | | The Docker network from which to remove the Dapr runtime |
