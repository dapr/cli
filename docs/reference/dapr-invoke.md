# dapr invoke

## Description

Invokes a Dapr app with an optional payload

## Usage

```bash
dapr invoke [flags]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| `--app-id`, `-a` | | | The app ID to invoke |
| `--help`, `-h` | | | Help for invoke |
| `--method`, `-m` | | | The method to invoke |
| `--payload`, `-p` | | | (optional) a json payload |
| `--verb`, `-v` | | | (optional) The HTTP verb to use. default is POST |
