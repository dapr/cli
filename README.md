# Actions CLI

[![Build Status](https://dev.azure.com/azure-octo/Actions/_apis/build/status/builds/cli%20build?branchName=master)](https://dev.azure.com/azure-octo/Actions/_build/latest?definitionId=6&branchName=master)

The Actions CLI allows you to setup Actions on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Actions instances.

## Getting started

### Prerequisites
* Download the [release](https://github.com/actionscore/cli/releases) for your OS
* Unpack it
* Move it to your desired location (for Mac/Linux - ```mv actions /usr/local/bin```. For Windows, add the executable to your System PATH.)

__*Note: For Windows users, run the cmd terminal in administrator mode*__

__*Note: For Linux users, if you run docker cmds with sudo, you need to use ```sudo actions init```*__

### Install Actions

To setup Actions on your local machine:

__*Note: For Windows users, run the cmd terminal in administrator mode*__

```
$ actions init
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Actions is up and running
```

To setup Actions on Kubernetes:

```
$ actions init --kubernetes
⌛  Making the jump to hyperspace...
▇   Deploying the Actions Operator to your cluster...
✅  Success! Actions is up and running. To verify, run 'kubectl get pods' in your terminal
```

#### Installing a specific version (Standalone)

Using `actions init` will download and install the latest version of Actions.
In order to specify a specific version of the Actions runtime, use the `runtime-version` flag: 

```
$ actions init --runtime-version v0.3.0-alpha
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Actions is up and running
```

*Note: The init command will install the latest stable version of Actions on your cluster. For more advanced use cases, use our [Helm Chart](https://github.com/actionscore/actions/tree/master/charts/actions-operator).*

### Launch Actions and your app

The Actions CLI lets you debug easily by launching both Actions and your app.
Logs from both the Actions Runtime and your app will be displayed in real time!

Example of launching Actions with a node app:

```
$ actions run --app-id nodeapp node app.js
```

Example of launching Actions with a node app listening on port 3000:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js
```

Example of launching Actions on port 6000:

```
$ actions run --app-id nodeapp --app-port 3000 --port 6000 node app.js
```

### Publish/Subscribe

To use pub-sub with your app, make sure that your app has a ```POST``` HTTP endpoint with some name, say ```myevent```.
This sample assumes your app is listening on port 3000.

Launch Actions and your app:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js
```

Publish a message:

```
$ actions publish --topic myevent
```

Publish a message with a payload:

```
$ actions publish --topic myevent --payload '{ "name": "yoda" }'
```

### Invoking

To test your endpoints with Actions, simply expose any ```POST``` HTTP endpoint.
For this sample, we'll assume a node app listening on port 300 with a ```/mymethod``` endpoint.

Launch Actions and your app:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js
```

Invoke your app:

```
$ actions send --app-id nodeapp --method mymethod
```

### List

To list all Actions instances running on your machine:

```
$ actions list
```

To list all Actions instances running in a Kubernetes cluster:

```
$ actions list --kubernetes
```

### Stop

Use ```actions list``` to get a list of all running instances.
To stop an actions app on your machine:

```
$ actions stop --app-id myAppID
```

### Enable profiling

In order to enable profiling, use the `enable-profiling` flag:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js --enable-profiling
```

Actions will automatically assign a profile port for you.
If you want to manually assign a profiling port, use the `profile-port` flag:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js --enable-profiling --profile-port 7777
```

### Set log level

In order to set the Actions runtime log verbosity level, use the `log-level` flag:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js --log-level debug
```

This sets the Actions log level to `debug`.
The default is `info`.

### Uninstall (Kubernetes)

To remove Actions from your Kubernetes cluster, use the `uninstall` command.

*Note this won't remove Actions installations that were deployed using Helm.*

```
$ actions uninstall --kubernetes
```


## Developing Actions CLI

### Prerequisites

1. The Go language environment [(instructions)]( https://golang.org/doc/install).

    Make sure you've already configured your GOPATH and GOROOT environment variables.

### Clone the repo

```
cd $GOPATH/src
mkdir -p github.com/actionscore/cli
git clone https://github.com/actionscore/cli.git github.com/actionscore/cli
```

### Build

```
cd $GOPATH/src/actions/cli
make build
```

**Windows Users**: Actions currently takes a dependency on gcc. If building throws an error containing: `"gcc": executable file not found in %PATH%`, you'll need to install gcc through the [Cygwin Project](https://sourceware.org/cygwin/) or the [MinGW Project](http://mingw-w64.org/doku.php).