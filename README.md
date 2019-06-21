# Actions CLI

[![Build Status](https://dev.azure.com/azure-octo/Actions/_apis/build/status/builds/cli%20build?branchName=master)](https://dev.azure.com/azure-octo/Actions/_build/latest?definitionId=6&branchName=master)

The Actions CLI allows you to setup Actions on your local dev machine or on a Kubernetes cluster, provides debugging suppors, launches and manages Actions instances.

## Setup

* Download the [release](https://github.com/actionscore/cli/releases) for your OS
* Unpack it
* Move it to your desired location (for Mac/Linux - ```mv actions /usr/local/bin```. For Windows, add the executable to your System PATH.)

### Usage

To setup Actions on your local machine:

__*Note: For Windows users, run the cmd terminal in administrator mode*__

```
$ actions init
⌛  Making the jump to hyperspace...
✅  Success! Get ready to rumble
```

To setup Actions on Kubernetes:

```
$ actions init --kubernetes
⌛  Making the jump to hyperspace...
✅  Success! Get ready to rumble
```
