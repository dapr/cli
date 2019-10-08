## Prerequisites

* Download the [release](https://github.com/dapr/cli/releases) for your OS
* Unpack it
* Move it to your desired location (for Mac/Linux - ```mv dapr /usr/local/bin```. For Windows, add the executable to your System PATH.)

__*Note: For Windows users, run the cmd terminal in administrator mode*__

__*Note: For Linux users, if you run docker cmds with sudo, you need to use ```sudo dapr init```*__

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Dapr instances.

## Install Dapr

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

### Installing a specific version (Standalone)

Using `dapr init` will download and install the latest version of Dapr.
In order to specify a specific version of the Dapr runtime, use the `runtime-version` flag: 

```
$ dapr init --runtime-version v0.3.0-alpha
⌛  Making the jump to hyperspace...
↗   Downloading binaries and setting up components...
✅  Success! Dapr is up and running
```

*Note: The init command will install the latest stable version of Dapr on your cluster. For more advanced use cases, use our [Helm Chart](https://github.com/dapr/dapr/tree/master/charts/dapr-operator).*

## Launch Dapr and your app

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

## Use gRPC

If your app uses gRPC instead of HTTP to receive Dapr events, run the CLI with the following command:

```
dapr run --app-id nodeapp --protocol grpc --app-port 6000 node app.js
```

The example above assumed your app port is 6000.

## Publish/Subscribe

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

## Invoking

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

## List

To list all Dapr instances running on your machine:

```
$ dapr list
```

To list all Dapr instances running in a Kubernetes cluster:

```
$ dapr list --kubernetes
```

## Stop

Use ```dapr list``` to get a list of all running instances.
To stop an dapr app on your machine:

```
$ dapr stop --app-id myAppID
```

## Enable profiling

In order to enable profiling, use the `enable-profiling` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --enable-profiling
```

Dapr will automatically assign a profile port for you.
If you want to manually assign a profiling port, use the `profile-port` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --enable-profiling --profile-port 7777
```

## Set log level

In order to set the Dapr runtime log verbosity level, use the `log-level` flag:

```
$ dapr run --app-id nodeapp --app-port 3000 node app.js --log-level debug
```

This sets the Dapr log level to `debug`.
The default is `info`.

## Uninstall

### Standalone

To remove Dapr placement container, use the `uninstall` command

```
$ dapr uninstall
```

### Kubernetes

To remove Dapr from your Kubernetes cluster, use the `uninstall` command with `--kubernetes`

*Note this won't remove Dapr installations that were deployed using Helm.*

```
$ dapr uninstall --kubernetes
```