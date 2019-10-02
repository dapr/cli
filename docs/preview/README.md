## Prerequisites

* Download the [release](https://github.com/dapr/cli/releases) for your OS
* Unpack it
* Move it to your desired location (for Mac/Linux - ```mv dapr /usr/local/bin```. For Windows, add the executable to your System PATH.)

__*Note: For Windows users, run the cmd terminal in administrator mode*__

__*Note: For Linux users, if you run docker cmds with sudo, you need to use ```sudo dapr init```*__

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Dapr instances.

## Launch Dapr and your app

The Dapr CLI lets you debug easily by launching both Dapr and your app.
Logs from both the Dapr Runtime and your app will be displayed in real time!

Example of launching Dapr with a node app:

```bash
$ dapr run --app-id nodeapp node app.js
```

Example of launching Dapr with a node app listening on port 3000:

```bash
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Example of launching Dapr on port 6000:

```bash
$ dapr run --app-id nodeapp --app-port 3000 --port 6000 node app.js
```

## Publish/Subscribe

To use pub-sub with your app, make sure that your app has a ```POST``` HTTP endpoint with some name, say ```myevent```.
This sample assumes your app is listening on port 3000.

Launch Dapr and your app:

```bash
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Publish a message:

```bash
$ dapr publish --topic myevent
```

Publish a message with a payload:

* Linux/Mac
```bash
$ dapr publish --topic myevent --payload '{ "name": "yoda" }'
```
* Windows
```bash
C:> dapr publish --topic myevent --payload "{ \"name\": \"yoda\" }"
```

## Invoking

To test your endpoints with Dapr, simply expose any ```POST``` HTTP endpoint.
For this sample, we'll assume a node app listening on port 300 with a ```/mymethod``` endpoint.

Launch Dapr and your app:

```bash
$ dapr run --app-id nodeapp --app-port 3000 node app.js
```

Invoke your app:

```bash
$ dapr send --app-id nodeapp --method mymethod
```

## List

To list all Dapr instances running on your machine:

```bash
$ dapr list
```

To list all Dapr instances running in a Kubernetes cluster:

```bash
$ dapr list --kubernetes
```

## Stop

Use ```dapr list``` to get a list of all running instances.
To stop an dapr app on your machine:

```bash
$ dapr stop --app-id myAppID
```
