# Dapr CLI Installer

## Windows

### Get the latest stable version

```
powershell -Command "iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1 | iex"
```

### Get the specific version

```
powershell -Command "$script=iwr -useb https://raw.githubusercontent.com/dapr/cli/youngp/add-version-param/install/install.ps1; $block=[ScriptBlock]::Create($script); invoke-command -ScriptBlock $block -ArgumentList 1.0.0-rc.1"
```

## MacOS

### Get the latest stable version

```
curl -fsSL https://raw.githubusercontent.com/dapr/cli/master/install/install.sh | /bin/bash
```

### Get the specific version

```
curl -fsSL https://raw.githubusercontent.com/dapr/cli/master/install/install.sh | /bin/bash -s <Version>
```

## Linux

### Get the latest stable version

```
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash
```

### Get the specific version

```
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s <Version>
```