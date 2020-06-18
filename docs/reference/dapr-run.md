# dapr run

## Description

Launches Dapr and your app side-by-side

## Usage

```bash
dapr run [flags] [command]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| `--app-id` | | | An ID for your application, used for service discovery |
| `--app-port` | | `-1` | The port your application is listening on |
| `--components-path` | | `~/.dapr/components or %USERPROFILE%\.dapr\components` | Path for components directory |
| `--config` | | `~/.dapr/config.yaml or %USERPROFILE%\.dapr\config.yaml` | Dapr configuration file |
| `--enable-profiling` | | | Enable `pprof` profiling via an HTTP endpoint |
| `--grpc-port` | | `-1` | The gRPC port for Dapr to listen on |
| `--help`, `-h` | | | Help for run |
| `--image` | | | The image to build the code in. Input is: `repository/image` |
| `--log-level` | | `info` | Sets the log verbosity. Valid values are: `debug`, `info`, `warning`, `error`, `fatal`, or `panic` |
| `--max-concurrency` | | `-1` | Controls the concurrency level of the app |
| `--placement-host` | `DAPR_PLACEMENT_HOST` | `localhost` | The host on which the placement service resides |
| `--port`, `-p` | | `-1` | The HTTP port for Dapr to listen on |
| `--profile-port` | | `-1` | The port for the profile server to listen on |
| `--protocol` | | `http` | Tells Dapr to use HTTP or gRPC to talk to the app. Valid values are: `http` or `grpc` |
