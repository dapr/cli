# Actions CLI

The Actions CLI allows you to setup Actions on your local dev machine or on a Kubernetes cluster, provides debugging support, launches and manages Actions instances.


## Launch Actions and your app

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


## Publish/Subscribe

To use pub-sub with your app, make sure that your app has a ```POST``` HTTP endpoint with some name, say ```myevent```.
This sample assumes your app is listening on port 3000.

Launch Actions and your app:

```
$ actions run --app-id nodeapp --app-port 3000 node app.js
```

Publish a message:

```
$ actions publish --app-id nodeapp --topic myevent
```

Publish a message with a payload:

```
$ actions publish --app-id nodeapp --topic myevent --payload '{ "name": "yoda" }'
```


## Invoking

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


## List

To list all Actions instances running on your machine:

```
$ actions list
```

To list all Actions instances running in a Kubernetes cluster:

```
$ actions list --kubernetes
```
