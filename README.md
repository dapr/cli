# Dapr CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/dapr/cli)](https://goreportcard.com/report/github.com/dapr/cli)
[![Build Status](https://github.com/dapr/cli/workflows/dapr_cli/badge.svg)](https://github.com/dapr/cli/actions?workflow=dapr_cli)
[![codecov](https://codecov.io/gh/dapr/cli/branch/master/graph/badge.svg)](https://codecov.io/gh/dapr/cli)

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Dapr instances.

## Getting started

### Prerequisites

On default, during initialization the Dapr CLI will install the Dapr binaries as well as setup a developer environment to help you get started easily with Dapr. This environment uses Docker containers, therefore Docker needs to be installed. If you prefer to run Dapr without this environment and no dependency on Docker, after installation of the CLI make sure follow the instructions to initialize Dapr using [slim init](#slim-init).

Note, if you are a new user, it is strongly recommended to intall Docker and use the regular init command.

* Install [Docker](https://docs.docker.com/install/)

>__Note: On Windows, Docker must be running in Linux Containers mode__

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

### Install Dapr on your local machine (self-hosted)

In self-hosted mode, dapr can be initialized using the CLI  with the placement, redis and zipkin containers enabled by default(recommended) or without them which also does not require docker to be available in the environment.

#### Initialize Dapr
([Prerequisite](#Prerequisites): Docker is available in the environment - recommended)

Use the init command to initialize Dapr. On init, multiple default configuration files and containers are installed along with the dapr runtime binary. Dapr runtime binary is installed under $HOME/.dapr/bin for Mac, Linux and %USERPROFILE%\.dapr\bin for Windows.

```bash
dapr init
```

> For Linux users, if you run your docker cmds with sudo, you need to use "**sudo dapr init**"

Output should look like so:

```
⌛  Making the jump to hyperspace...
✅  Downloaded binaries and completed components set up.
ℹ️  daprd binary has been installed to  $HOME/.dapr/bin.
ℹ️  dapr_placement container is running.
ℹ️  dapr_redis container is running.
ℹ️  dapr_zipkin container is running.
ℹ️  Use `docker ps` to check running containers.
✅  Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started
```

> Note: To see that Dapr has been installed successfully, from a command prompt run the `docker ps` command and check that the `daprio/dapr:latest`,  `dapr_redis` and `dapr_zipkin` container images are all running.

This step creates the following defaults:
1. components folder which is later used during `dapr run` unless the `--components-path` option is provided. For Linux/MacOS, the default components folder path is `$HOME/.dapr/components` and for Windows it is `%USERPROFILE%\.dapr\components`.
2. component files in the components folder called `pubsub.yaml`, `statestore.yaml` and `zipkin.yaml`.
3. default config file `$HOME/.dapr/config.yaml` for Linux/MacOS or for Windows at `%USERPROFILE%\.dapr\config.yaml` to enable tracing on `dapr init` call. Can be overridden with the `--config` flag on `dapr run`.

#### Slim Init

Alternatively to the above, to have the CLI not install any default configuration files or run Docker containers, use the `--slim` flag with the init command. Only Dapr binaries will be installed.

```bash
dapr init --slim
```

Output should look like so:

```bash
⌛  Making the jump to hyperspace...
✅  Downloaded binaries and completed components set up.
ℹ️  daprd binary has been installed to $HOME/.dapr/bin.
ℹ️  placement binary has been installed.
✅  Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started
```

>Note: When initializing Dapr with the `--slim` flag only the Dapr runtime binary and the placement service binary are installed. An empty default components folder is created with no default configuration files. During `dapr run` user should use `--components-path` to point to a components directory with custom configurations files or alternatively place these files in the default directory. For Linux/MacOS, the default components directory path is `$HOME/.dapr/components` and for Windows it is `%USERPROFILE%\.dapr\components`.

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

> Note: When installed to a specific Docker network, you will need to add the `--placement-host-address` arguments to `dapr run` commands run in any containers within that network.

#### Install with a specific host on which the Redis service resides
```bash
# Specify a particular redis host
$ dapr init --redis-host 10.0.0.1
```

### Uninstall Dapr in a standalone mode

Uninstalling will remove daprd binary and the placement container (if installed with Docker or the placement binary if not).


```bash
$ dapr uninstall
```
> For Linux users, if you run your docker cmds with sudo, you need to use "**sudo dapr uninstall**" to remove the containers.

The command above won't remove the redis or zipkin containers by default in case you were using it for other purposes.  It will also not remove the default dapr folder that was created on `dapr init`. To remove all the containers (placement, redis, zipkin) and also the default dapr folder created on init run:

```bash
$ dapr uninstall --all
```

The above command can also be run when Dapr has been installed in a non-docker environment, it will only remove the installed binaries and the default dapr folder in that case.

> NB: The `dapr uninstall` command will always try to remove the placement binary/service and will throw an error is not able to.

**You should always run a `dapr uninstall` before running another `dapr init`.**

#### Uninstall Dapr from a specific Docker network

If previously installed to a specific Docker network, Dapr can be uninstalled with the `--network` argument:

```bash
$ dapr uninstall --network dapr-network
```

### Install Dapr on Kubernetes

The init command will install Dapr to a Kubernetes cluster. For more advanced use cases, use our [Helm Chart](https://github.com/dapr/dapr/tree/master/charts/dapr).

*Note: The default namespace is dapr-system*

```
$ dapr init -k

⌛  Making the jump to hyperspace...
ℹ️  Note: To install Dapr using Helm, see here:  https://github.com/dapr/docs/blob/master/getting-started/environment-setup.md#using-helm-advanced

✅  Deploying the Dapr control plane to your cluster...
✅  Success! Dapr has been installed to namespace dapr-system. To verify, run "dapr status -k" in your terminal. To get started, go here: https://aka.ms/dapr-getting-started
```

#### Installing to a custom namespace

```
$ dapr init -k -n my-namespace
```

#### Installing with a highly avaialable control plane config

```
$ dapr init -k --enable-ha=true
```

#### Installing with mTLS disabled

```
$ dapr init -k --enable-mtls=false
```

#### Uninstall Dapr on Kubernetes

To remove Dapr from your Kubernetes cluster, use the `uninstall` command with `--kubernetes` flag or the `-k` shorthand.

```
$ dapr uninstall -k
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
$ dapr run --app-id nodeapp --app-port 3000 --dapr-http-port 6000 node app.js
```

Example of launching Dapr on gRPC port 50002:

```
$ dapr run --app-id nodeapp --app-port 3000 --dapr-grpc-port 50002 node app.js
```

Example of launching Dapr within a specific Docker network:

```bash
$ dapr init --redis-host dapr_redis --network dapr-network
$ dapr run --app-id nodeapp --placement-host-address dapr_placement node app.js
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

The `--pubsub` parameter takes in the name of the pub/sub.  The default name of the pub/sub configed by the CLI is "pubsub".


Publish a message:

* Linux/Mac
```bash
$ dapr publish --pubsub pubsub --topic myevent --data '{ "name": "yoda" }'
```

* Windows
```bash
C:> dapr publish --pubsub pubsub --topic myevent --data "{ \"name\": \"yoda\" }"
```

### Invoking

To test your endpoints with Dapr, simply expose any HTTP endpoint.
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

Specify a verb:

By default, Dapr will use the `POST` verb. If your app uses Dapr for gRPC, you should use `POST`.

```
$ dapr invoke --app-id nodeapp --method mymethod --verb GET
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

### Export TLS certificates

To export the root cert, issuer cert and issuer key created by Dapr from a Kubernetes cluster to a local path:

```
$ dapr mtls export
```

This will save the certs to the working directory.

To specify a custom directory:

```
$ dapr mtls export -o certs
```

### Check root certificate expiry

```
$ dapr mtls expiry
```

This can be used when upgrading to a newer version of Dapr, as it's recommended to carry over the existing certs for a zero downtime upgrade.

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

### Enable SSL when invoking an app

If your app is listening on `https` or has a gRPC TLS configuration enabled, use the following `app-ssl` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --app-ssl
```

This will have Dapr invoke the app over an insecure SSL channel.

The default is false.

### Running sidecar only

You can run Dapr's sidecar only (`daprd`) by omitting the application's command in the end:

```
$ dapr run --app-id myapp --dapr-http-port 3005 --dapr-grpc-port 50001
```

### Generate shell completion scripts

To generate shell completion scripts:

```
$ dapr completion
```

For more details, please run the command and check the examples to apply to your shell.

## Reference for the Dapr CLI

See the [Reference Guide](docs/reference/reference.md) for more information about individual Dapr commands.

## Contributing to Dapr CLI

See the [Development Guide](https://github.com/dapr/cli/blob/master/docs/development/development.md) to get started with building and developing.

## Code of Conduct

Please refer to our [Dapr Community Code of Conduct](https://github.com/dapr/community/blob/master/CODE-OF-CONDUCT.md)
