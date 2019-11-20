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
| `--all` | | `false` | Remove Redis container in addition to actor placement container |
| `--help`, `-h` | | | Help for uninstall |
| `--kubernetes` | | `false` | Uninstall Dapr from a Kubernetes cluster |
| `--network` | `DAPR_NETWORK` | | The Docker network from which to remove the Dapr runtime |
