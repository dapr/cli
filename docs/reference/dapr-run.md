# dapr run

## Description

Launches dapr and your app side-by-side

## Usage

```bash
dapr run [flags] [command]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| --app-id | N/A | N/A | An ID for your application, used for service discovery |
| --app-port | N/A | `-1` | The port your application is listening on |
| --config | N/A | N/A | Dapr configuration file |
| --enable-profiling | N/A | N/A | Enable `pprof` profiling via an HTTP endpoint |
| --grpc-port | N/A | `-1` | The gRPC port for Dapr to listen on |
| --help, -h | N/A | N/A | Help for run |
| --image | N/A | N/A | The image to build the code in. Input is: `repository/image` |
| --log-level | N/A | `info` | Sets the log verbosity. Valid values are: `debug`, `info`, `warning`, `error`, `fatal`, or `panic` |
| --max-concurrency | N/A | `-1` | Controls the concurrency level of the app |
| --placement-host | `DAPR_PLACEMENT_HOST` | `localhost` | The host on which the placement service resides |
| --port, -p | N/A | `-1` | The HTTP port for Dapr to listen on |
| --profile-port | N/A | `-1` | The port for the profile server to listen on |
| --protocol | N/A | `http` | Tells Dapr to use HTTP or gRPC to talk to the app. Valid values are: `http` or `grpc` |
| --redis-host | `DAPR_REDIS_HOST` | `localhost` | The host on which the Redis service resides |
