# dapr init

## Description

Setup Dapr in Kubernetes or Standalone modes

## Usage

```bash
dapr init [flags]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| `--kubernetes` | | `false` | Deploy Dapr to a Kubernetes cluster |
| `--help`, `-h` | | | Help for init |
| `--network` | `DAPR_NETWORK` | | The Docker network on which to deploy the Dapr runtime |
| `--runtime-version` | | `latest` | The version of the Dapr runtime to install, for example: `v0.1.0-alpha` |