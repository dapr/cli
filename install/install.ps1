$ErrorActionPreference = 'stop'

$DaprRoot="c:\dapr"

$basic = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes("youngubpark:2b9381bbaa27bf76a285570e71fbc65d35a47bc0"));
$authHeader = @{"Authorization"="Basic $basic"}

if((Get-ExecutionPolicy) -gt 'RemoteSigned' -or (Get-ExecutionPolicy) -eq 'ByPass') {
    Write-Output "PowerShell requires an execution policy of 'RemoteSigned' to run Scoop."
    Write-Output "To make this change please run:"
    Write-Output "'Set-ExecutionPolicy RemoteSigned -scope CurrentUser'"
    break
}

# Create Dapr Directory
Write-Output "Creating $DaprRoot directory"
New-Item -ErrorAction Ignore -Name $DaprRoot -ItemType directory
if (!(Test-Path $DaprRoot)) {
    throw "Cannot create $DaprRoot"
}

# Get the list of release from GitHub
$releases = Invoke-RestMethod -Headers $authHeader -Uri "https://api.github.com/repos/dapr/cli/releases" -Method Get
if ($releases.Count -eq 0) {
    throw "No releases from github.com/dapr/cli repo"
}

# Filter windows binary and download archive
$windowsAsset = $releases[0].assets | where-object { $_.name -Like "*windows_amd64.zip" }
if (!$windowsAsset) {
    throw "Cannot find the windows dapr cli binary"
}
$zipFilePath = $DaprRoot$windowsAsset.name

Write-Output "Downloading $windowsAsset.name to $zipFilePath..."
Invoke-WebRequest -Headers $authHeader -Uri $windowsAsset.url -OutFile $zipFilePath
if (!(Test-Path $zipFilePath)) {
    throw "Failed to download Dapr Cli archieve - $zipFilePath"
}

# Extract Dapr Runtime to c:\dapr
$DaprRuntimeFileName = "dapr.exe"
$DaprRuntimePath = $DaprRoot\$DaprRuntimeFileName
Write-Output "Extracting $zipFilePath..."
Expand-Archive -Path $zipFilePath -DestinationPath $DaprRoot
if (!(Test-Path $DaprRuntimePath)) {
    throw "Failed to download Dapr Cli archieve - $zipFilePath"
}

Write-Output "Clean up $zipFilePath..."
Remove-Item $zipFilePath -Force

Write-Output "Try to add $DaprRoot to User Path Environment variable..."
$UserPathEnvionmentVar = [Environment]::GetEnvironmentVariable("PATH", "User")
if($UserPathEnvionmentVar -notlike '*dapr*') {
    Write-Output "Skipping to add $DaprRoot to User Path - $UserPathEnvionmentVar"
} else {
    [System.Environment]::SetEnvironmentVariable("PATH", $UserPathEnvionmentVar + ";$DaprRoot", "User")
    $UserPathEnvionmentVar = [Environment]::GetEnvironmentVariable("PATH", "User")
    Write-Output "Added $DaprRoot to User Path - $UserPathEnvionmentVar"
}

Write-Output "Dapr CLI is installed successfully."
Write-Output "Visit https://github.com/dapr/docs/blob/master/getting-started/environment-setup.md to start Dapr."
