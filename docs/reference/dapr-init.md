# dapr init

## Description

Setup dapr in Kubernetes or Standalone modes

## Usage

```bash
dapr init [flags]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| --kubernetes | N/A | `false` | Deploy Dapr to a Kubernetes cluster |
| --network | `DAPR_NETWORK` | None | The Docker network on which to deploy the Dapr runtime |
| --runtime-version | N/A | `latest` | The version of the Dapr runtime to install, for example: `v0.1.0-alpha` |