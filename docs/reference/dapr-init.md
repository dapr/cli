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
| `--help`, `-h` | | | Help for init |
| `--install-path` | `DAPR_INSTALL_PATH` | `Linux & Mac: /usr/local/bin` `Windows: C:\dapr` | The optional location to install Dapr to.  The default is /usr/local/bin for Linux/Mac and C:\dapr for Windows |
| `--kubernetes`, `-k` | | `false` | Deploy Dapr to a Kubernetes cluster |
| `--network` | `DAPR_NETWORK` | | The Docker network on which to deploy the Dapr runtime |
| `--runtime-version` | | `latest` | The version of the Dapr runtime to install. for example: v0.1.0 (default "latest") |
| `--runtime-version` | | `latest` | The version of the Dapr runtime to install, for example: `v0.1.0-alpha` |
| `--redis-host` | `DAPR_REDIS_HOST` | `localhost` | The host on which the Redis service resides |
| `--install-path` |  | `/usr/local/bin` for Linux/Mac and `C:\dapr` for Windows | The optional location to install Dapr to. |
| `--slim`, `-s` | | `false` | Initialize dapr in self-hosted mode without placement, redis and zipkin containers.|
