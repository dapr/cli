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
| `--redis-host` | `DAPR_REDIS_HOST` | `localhost` | The host on which the Redis service resides |
| `--install-path` |  | `/usr/local/bin` for Linux/Mac and `C:\dapr` for Windows | The optional location to install Dapr to. |
| `--slim`, `-s` | | `false` | Initialize dapr in self-hosted mode without docker.|
