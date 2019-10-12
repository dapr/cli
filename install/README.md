# Dapr CLI Installer

## Install Dapr CLI

### Windows

```
powershell -Command "iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1 | iex"
```

Note: Until the repo is public, please use the below command.

```
powershell -Command "$Env:GITHUB_USER='your_github_id'; $Env:GITHUB_TOKEN='your_github_pat_token'; iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1?token=AC2QIRCKXC5TWOYUZLHFMWC5VJ6FI | iex"
```

### Linux/MacOS

WIP.