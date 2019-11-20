# Dapr (`dapr`) Command Line Interface (CLI)

The Dapr CLI allows you to setup Dapr on your local dev machine or on a Kubernetes cluster, provides debugging support, and launches and manages Dapr instances.

```bash
         __                
    ____/ /___ _____  _____
   / __  / __ '/ __ \/ ___/
  / /_/ / /_/ / /_/ / /    
  \__,_/\__,_/ .___/_/     
              /_/            
                                                                           
======================================================
A serverless runtime for hyperscale, distributed systems

Usage:
  dapr [command]

Available Commands:
  help        Help about any command
  init        Setup dapr in Kubernetes or Standalone modes
  list        List all dapr instances
  publish     publish an event to multiple consumers
  run         Launches dapr and your app side by side
  send        invoke a dapr app with an optional payload
  stop        Stops a running dapr instance and its associated app
  uninstall   removes a dapr installation

Flags:
  -h, --help      help for dapr
      --version   version for dapr

Use "dapr [command] --help" for more information about a command.
```

## Command Reference

You can learn more about each Dapr command from the links below.

 - [`dapr help`](dapr-help.md)
 - [`dapr init`](dapr-init.md)
 - [`dapr list`](dapr-list.md)
 - [`dapr publish`](dapr-publish.md)
 - [`dapr run`](dapr-run.md)
 - [`dapr send`](dapr-send.md)
 - [`dapr stop`](dapr-stop.md)
 - [`dapr uninstall`](dapr-uninstall.md)

## Environment Variables

Some Dapr flags can be set via environment variables (e.g. `DAPR_NETWORK` for the `--network` flag of the `dapr init` command). Note that specifying the flag on the command line overrides any set environment variable.