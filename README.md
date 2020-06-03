# Dapr CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/dapr/cli)](https://goreportcard.com/report/github.com/dapr/cli)
[![Build Status](https://github.com/dapr/cli/workflows/dapr_cli/badge.svg)](https://github.com/dapr/cli/actions?workflow=dapr_cli)

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Dapr instances.

## Getting started

### Prerequisites

* Install [Docker](https://docs.docker.com/install/)

*__Note: On Windows, Docker must be running in Linux Containers mode__*

### Installing Dapr CLI

#### Using script to install the latest release

**Windows**

Install the latest windows Dapr CLI to `c:\dapr` and add this directory to User PATH environment variable. Use `-DaprRoot [path]` to change the default installation directory

```powershell
powershell -Command "iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1 | iex"
```

**Linux**

Install the latest linux Dapr CLI to `/usr/local/bin`

```bash
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash
```

**MacOS**

Install the latest darwin Dapr CLI to `/usr/local/bin`

```bash
curl -fsSL https://raw.githubusercontent.com/dapr/cli/master/install/install.sh | /bin/bash
```

#### From the Binary Releases

Each release of Dapr CLI includes various OSes and architectures. These binary versions can be manually downloaded and installed.

1. Download the [Dapr CLI](https://github.com/dapr/cli/releases)
2. Unpack it (e.g. dapr_linux_amd64.tar.gz, dapr_windows_amd64.zip)
3. Move it to your desired location.
   * For Linux/MacOS - `/usr/local/bin`
   * For Windows, create a directory and add this to your System PATH. For example create a directory called `c:\dapr` and add this directory to your path, by editing your system environment variable.

### Install Dapr on your local machine (standalone)

```
$ dapr init
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Dapr is up and running
```

> Note: To see that Dapr has been installed successfully, from a command prompt run the `docker ps` command and check that the `daprio/dapr:latest` and `redis` container images are both running. Also, this step creates a default components folder under your home directory/.dapr/components which is later used at runtime unless the --components-path option is provided

#### Install a specific runtime version

You can install or upgrade to a specific version of the Dapr runtime using `dapr init --runtime-version`. You can find the list of versions in [Dapr Release](https://github.com/dapr/dapr/releases).

```bash
# Install v0.1.0 runtime
$ dapr init --runtime-version 0.1.0

# Check the versions of CLI and runtime
$ dapr --version
CLI version: v0.1.0
Runtime version: v0.1.0
```

#### Install to a specific Docker network

You can install the Dapr runtime to a specific Docker network in order to isolate it from the local machine (e.g. to use Dapr from *within* a Docker container).

```bash
# Create Docker network
$ docker network create dapr-network

# Install Dapr to the network
$ dapr init --network dapr-network
```

> Note: When installed to a specific Docker network, you will need to add the `--redis-host` and `--placement-host` arguments to `dapr run` commands run in any containers within that network.

### Uninstall Dapr in a standalone mode


Uninstalling will remove the placement container.  


```bash
$ dapr uninstall
```


The command above won't remove the redis container by default in case you were using it for other purposes.  To remove both the placement and redis container:

```bash
$ dapr uninstall --all
```

You should always run a `dapr uninstall` before running another `dapr init`.	

#### Uninstall Dapr from a specific Docker network

If previously installed to a specific Docker network, Dapr can be uninstalled with the `--network` argument:

```bash
$ dapr uninstall --network dapr-network
```

### Install Dapr on Kubernetes

The init command will install the latest stable version of Dapr on your cluster. For more advanced use cases, use our [Helm Chart](https://github.com/dapr/dapr/tree/master/charts/dapr).

> Please note, that using the CLI does not support non-default namespaces.  
> If you need a non-default namespace, please use Helm.

```
$ dapr init --kubernetes
⌛  Making the jump to hyperspace...
ℹ️  Note: this installation is recommended for testing purposes. For production environments, please use Helm

✅  Deploying the Dapr control plane to your cluster...
✅  Success! Dapr has been installed. To verify, run 'kubectl get pods -w' or 'dapr status -k' in your terminal. To get started, go here: https://aka.ms/dapr-getting-started
```

#### Uninstall Dapr on Kubernetes

To remove Dapr from your Kubernetes cluster, use the `uninstall` command with `--kubernetes`

> Note: this won't remove Dapr installations that were deployed using Helm.

```
$ dapr uninstall --kubernetes
```

### Launch Dapr and your app

The Dapr CLI lets you debug easily by launching both Dapr and your app.
Logs from both the Dapr Runtime and your app will be displayed in real time!

Example of launching Dapr with a node app:

```
$ dapr run --app-id nodeapp node app.js
```

Example of launching Dapr with a node app listening on port 3000:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Example of launching Dapr on HTTP port 6000:

```
$ dapr run --app-id nodeapp --app-port 3000 --port 6000 node app.js
```

Example of launching Dapr on gRPC port 50002:

```
$ dapr run --app-id nodeapp --app-port 3000 --grpc-port 50002 node app.js
```

Example of launching Dapr within a specific Docker network:

```bash
$ dapr run --app-id nodeapp --redis-host dapr_redis --placement-host dapr_placement node app.js
```

> Note: When in a specific Docker network, the Redis and placement service containers are given specific network aliases, `dapr_redis` and `dapr_placement`, respectively.

### Use gRPC

If your app uses gRPC instead of HTTP to receive Dapr events, run the CLI with the following command:

```
dapr run --app-id nodeapp --protocol grpc --app-port 6000 node app.js
```

The example above assumed your app port is 6000.

### Publish/Subscribe

To use pub-sub with your app, make sure that your app has a ```POST``` HTTP endpoint with some name, say ```myevent```.
This sample assumes your app is listening on port 3000.

Launch Dapr and your app:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Publish a message:

```
$ dapr publish --topic myevent
```

Publish a message:

* Linux/Mac
```bash
$ dapr publish --topic myevent --data '{ "name": "yoda" }'
```

* Windows
```bash
C:> dapr publish --topic myevent --data "{ \"name\": \"yoda\" }"
```

### Invoking

To test your endpoints with Dapr, simply expose any ```POST``` HTTP endpoint.
For this sample, we'll assume a node app listening on port 3000 with a ```/mymethod``` endpoint.

Launch Dapr and your app:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Note: To choose a non-default components folder, use the --components-path option.

Invoke your app:

```
$ dapr invoke --app-id nodeapp --method mymethod
```

### List

To list all Dapr instances running on your machine:

```
$ dapr list
```

To list all Dapr instances running in a Kubernetes cluster:

```
$ dapr list --kubernetes
```

### Check system services (control plane) status

Check Dapr's system services (control plane) health status in a Kubernetes cluster:

```
$ dapr status --kubernetes
```

### Check mTLS status

To check if Mutual TLS is enabled in your Kubernetes cluster:

```
$ dapr mtls --kubernetes
```

### List Components

To list all Dapr components on Kubernetes:

```
$ dapr components --kubernetes
```

### Use non-default Components Path

To use a custom path for component definitions

```
$ dapr run --components-path [custom path]
```


### List Configurations

To list all Dapr configurations on Kubernetes:

```
$ dapr configurations --kubernetes
```

### Stop

Use ```dapr list``` to get a list of all running instances.
To stop a Dapr app on your machine:

```
$ dapr stop myAppID
```
You can also stop multiple Dapr apps
```
$ dapr stop myAppID1 myAppID2
```
### Enable profiling

In order to enable profiling, use the `enable-profiling` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --enable-profiling
```

Dapr will automatically assign a profile port for you.
If you want to manually assign a profiling port, use the `profile-port` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --enable-profiling --profile-port 7777
```

### Set log level

In order to set the Dapr runtime log verbosity level, use the `log-level` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --log-level debug
```

This sets the Dapr log level to `debug`.
The default is `info`.


### Running sidecar only

You can run Dapr's sidecar only (`daprd`) by omitting the application's command in the end:

```
$ dapr run --app-id myapp --port 3005 --grpc-port 50001
```

## Reference for the Dapr CLI

See the [Reference Guide](docs/reference/reference.md) for more information about individual Dapr commands.

## Contributing to Dapr CLI

See the [Development Guide](https://github.com/dapr/cli/blob/master/docs/development/development.md) to get started with building and developing.

## Code of Conduct

 This project has adopted the [Microsoft Open Source Code of conduct](https://opensource.microsoft.com/codeofconduct/).
 For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
