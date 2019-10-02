# Dapr CLI

[![Build Status](https://dev.azure.com/azure-octo/Dapr/_apis/build/status/builds/cli%20build?branchName=master)](https://dev.azure.com/azure-octo/Dapr/_build/latest?definitionId=6&branchName=master)

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Dapr instances.

## Getting started

### Prerequisites
* Download the [release](https://github.com/dapr/cli/releases) for your OS
* Unpack it
* Move it to your desired location (for Mac/Linux - ```mv dapr /usr/local/bin```. For Windows, add the executable to your System PATH.)

__*Note: For Windows users, run the cmd terminal in administrator mode*__

__*Note: For Linux users, if you run docker cmds with sudo, you need to use ```sudo dapr init```*__

### Install Dapr

To setup Dapr on your local machine:

__*Note: For Windows users, run the cmd terminal in administrator mode*__

```
$ dapr init
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Dapr is up and running
```

To setup Dapr on Kubernetes:

```
$ dapr init --kubernetes
⌛  Making the jump to hyperspace...
↗   Deploying the Dapr Operator to your cluster...
✅  Success! Dapr is up and running. To verify, run 'kubectl get pods' in your terminal
```

#### Installing a specific version (Standalone)

Using `dapr init` will download and install the latest version of Dapr.
In order to specify a specific version of the Dapr runtime, use the `runtime-version` flag: 

```
$ dapr init --runtime-version v0.3.0-alpha
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Dapr is up and running
```

*Note: The init command will install the latest stable version of Dapr on your cluster. For more advanced use cases, use our [Helm Chart](https://github.com/dapr/dapr/tree/master/charts/dapr-operator).*

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

Example of launching Dapr on port 6000:

```
$ dapr run --app-id nodeapp --app-port 3000 --port 6000 node app.js
```

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

Publish a message with a payload:

```
$ dapr publish --topic myevent --payload '{ "name": "yoda" }'
```

### Invoking

To test your endpoints with Dapr, simply expose any ```POST``` HTTP endpoint.
For this sample, we'll assume a node app listening on port 300 with a ```/mymethod``` endpoint.

Launch Dapr and your app:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Invoke your app:

```
$ dapr send --app-id nodeapp --method mymethod
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

### Stop

Use ```dapr list``` to get a list of all running instances.
To stop an dapr app on your machine:

```
$ dapr stop --app-id myAppID
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

### Uninstall (Kubernetes)

To remove Dapr from your Kubernetes cluster, use the `uninstall` command.

*Note this won't remove Dapr installations that were deployed using Helm.*

```
$ dapr uninstall --kubernetes
```


## Developing Dapr CLI

### Prerequisites

1. The Go language environment [(instructions)](https://golang.org/doc/install).
   * Make sure that your GOPATH and PATH are configured correctly
   ```bash
   export GOPATH=~/go
   export PATH=$PATH:$GOPATH/bin
   ```
1. [Delve](https://github.com/go-delve/delve/tree/master/Documentation/installation) for Debugging
1. *(for windows)* [MinGW](http://www.mingw.org/) to install gcc and make
   * Recommend to use [chocolatey mingw package](https://chocolatey.org/packages/mingw) and ensure that MinGW bin directory is in PATH environment variable

### Clone the repo

```bash
cd $GOPATH/src
mkdir -p github.com/dapr/cli
git clone https://github.com/dapr/cli.git github.com/dapr/cli
```

### Build

You can build dapr binaries via `make` tool and find the binaries in `./dist/{os}_{arch}/release/`.

> Note : for windows environment with MinGW, use `mingw32-make.exe` instead of `make`.

* Build for your current local environment

```bash
cd $GOPATH/src/github.com/dapr/cli/
make build
```

* Cross compile for multi platforms

```bash
make build GOOS=linux GOARCH=amd64
```

### Run unit-test

```bash
make test
```

### Debug Dapr CLI

We highly recommend to use [VSCode with Go plugin](https://marketplace.visualstudio.com/items?itemName=ms-vscode.Go) for your productivity. If you want to use the different editors, you can find the [list of editor plugins](https://github.com/go-delve/delve/blob/master/Documentation/EditorIntegration.md) for Delve.

This section introduces how to start debugging with Delve CLI. Please see [Delve documentation](https://github.com/go-delve/delve/tree/master/Documentation) for the detail usage.

#### Start with debugger

```bash
$ cd $GOPATH/src/github.com/dapr/cli
$ dlv debug .
Type 'help' for list of commands.
(dlv) break main.main
(dlv) continue
```

#### Debug unit-tests

```bash
# Specify the package that you want to test
# e.g. debuggin ./pkg/actors
$ dlv test ./pkg/actors
```
