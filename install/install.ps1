$ErrorActionPreference = 'stop'

# Constants
$DaprRoot="c:\dapr"
$DaprRuntimeFileName = "dapr.exe"
$DaprRuntimePath = "$DaprRoot\$DaprRuntimeFileName"

# Set Github request authentication for basic authentication.
if ($Env:GITHUB_USER) {
    $basicAuth = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($Env:GITHUB_USER + ":" + $Env:GITHUB_TOKEN));
    $githubHeader = @{"Authorization"="Basic $basicAuth"}
} else {
    $githubHeader = @{}
}

if((Get-ExecutionPolicy) -gt 'RemoteSigned' -or (Get-ExecutionPolicy) -eq 'ByPass') {
    Write-Output "PowerShell requires an execution policy of 'RemoteSigned'."
    Write-Output "To make this change please run:"
    Write-Output "'Set-ExecutionPolicy RemoteSigned -scope CurrentUser'"
    break
}

# Create Dapr Directory
Write-Output "Creating $DaprRoot directory"
New-Item -ErrorAction Ignore -Path $DaprRoot -ItemType "directory"
if (!(Test-Path $DaprRoot -PathType Container)) {
    throw "Cannot create $DaprRoot"
}

# Get the list of release from GitHub
$releases = Invoke-RestMethod -Headers $githubHeader -Uri "https://api.github.com/repos/dapr/cli/releases" -Method Get
if ($releases.Count -eq 0) {
    throw "No releases from github.com/dapr/cli repo"
}

# Filter windows binary and download archive
$windowsAsset = $releases[0].assets | where-object { $_.name -Like "*windows_amd64.zip" }
if (!$windowsAsset) {
    throw "Cannot find the windows dapr cli binary"
}

$zipFilePath = $DaprRoot + "\" + $windowsAsset.name
Write-Output "Downloading $zipFilePath ..."

$githubHeader.Accept = "application/octet-stream"
Invoke-WebRequest -Headers $githubHeader -Uri $windowsAsset.url -OutFile $zipFilePath
if (!(Test-Path $zipFilePath -PathType Leaf)) {
    throw "Failed to download Dapr Cli binary - $zipFilePath"
}

# Extract Dapr CLI to c:\dapr
Write-Output "Extracting $zipFilePath..."
Expand-Archive -Path $zipFilePath -DestinationPath $DaprRoot
if (!(Test-Path $DaprRuntimePath -PathType Leaf)) {
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

Write-Output "-----------------------------------"
Write-Output "Dapr CLI is installed successfully."
Write-Output "Visit https://github.com/dapr/docs/blob/master/getting-started/environment-setup.md to start Dapr."
