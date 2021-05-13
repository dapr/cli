# Dapr CLI Installer

## Windows

### Get the latest stable version

```
powershell -Command "iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1 | iex"
```

### Get the specific version

```
powershell -Command "$script=iwr -useb https://raw.githubusercontent.com/dapr/cli/master/install/install.ps1; $block=[ScriptBlock]::Create($script); invoke-command -ScriptBlock $block -ArgumentList <Version>"
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

## For Users with Poor Network Conditions

You can download resources from a mirror instead of from Github.

### Windows

- Create a `CustomAssetFactory` function to define what the release asset url you want to use
- Set `DaprReleaseJsonUrl` to the equivalent of the json representation of all releases at <https://api.github.com/repos/dapr/cli/releases>
- You could use cdn.jsdelivr.net global CDN for your location to download install.ps1

### Get the latest stable version

For example, if you are in Chinese mainland, you could use:

- Gitee.com to get latest release.json
- cnpmjs.org hosted by Alibaba for assets
- cdn.jsdelivr.net global CDN for install.ps1

```powershell
function CustomAssetFactory {
    param (
        $release
    )
    [hashtable]$return = @{}
    $return.url = "https://github.com.cnpmjs.org/dapr/cli/releases/download/$($release.tag_name)/dapr_windows_amd64.zip"
    $return.name = "dapr_windows_amd64.zip"
    $return
}
$params = @{
    CustomAssetFactory = ${function:CustomAssetFactory};
    DaprReleaseJsonUrl    = "https://gitee.com/dapr-cn/dapr-bin-mirror/raw/main/cli/releases.json";
}
$script=iwr -useb https://cdn.jsdelivr.net/gh/dapr/cli/install/install.ps1;
$block=[ScriptBlock]::Create(".{$script} $(&{$args} @params)");
Invoke-Command -ScriptBlock $block
```

### Get the specific version

```powershell
function CustomAssetFactory {
    param (
        $release
    )
    [hashtable]$return = @{}
    $return.url = "https://github.com.cnpmjs.org/dapr/cli/releases/download/$($release.tag_name)/dapr_windows_amd64.zip"
    $return.name = "dapr_windows_amd64.zip"
    $return
}
$params = @{
    CustomAssetFactory = ${function:CustomAssetFactory};
    DaprReleaseJsonUrl    = "https://gitee.com/dapr-cn/dapr-bin-mirror/raw/main/cli/releases.json";
    Version            = <Version>
}
$script=iwr -useb https://cdn.jsdelivr.net/gh/dapr/cli/install/install.ps1;
$block=[ScriptBlock]::Create(".{$script} $(&{$args} @params)");
Invoke-Command -ScriptBlock $block
```
