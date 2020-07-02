# dapr dashboard

## Description

Start Dapr dashboard.

## Usage

### Prerequisites

Dapr dashboard should be deployed in the Kubernetes cluster.

You can deploy the dashboard in the Kubernetes cluster as follows:

```bash
kubectl apply -f https://raw.githubusercontent.com/dapr/dashboard/master/deploy/dashboard.yaml
```

And then run:

```bash
dapr dashboard [flags]
```

## Flags

| Name | Environment Variable | Default | Description
| --- | --- | --- | --- |
| `--help`, `-h` | | | Help for dashboard |
| `--kubernetes`, `-k` | | `false` | Start Dapr dashboard in local browser |
| `--port`, `-p` | | `8080` | The local port on which to serve dashboard |