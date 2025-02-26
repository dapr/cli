function Uninstall-Dapr {
    $command = "dapr uninstall --all"

    $output = Invoke-Expression $command

    if ($output -match "Dapr has been removed successfully") {
        Write-Output "Uninstalled successfully"
    }
    else {
        Write-Output "Failed to uninstall dapr"
        $output
        exit 1
    }
}

function Initialize-Dapr($version, [string[]]$arguments) {
    $commandArguments = "init"
    if ($null -ne $version) {
        $commandArguments += " --runtime-version $version"
    }
    if ($null -ne $arguments) {
        foreach ($arg in $arguments) {
            $commandArguments += " $arg"
        }
    }
    Invoke-CommandCustom "dapr" $commandArguments
}

function Invoke-CommandCustom($command, [string[]]$arguments) {
    if ($null -ne $arguments) {
        foreach ($arg in $arguments) {
            $commandArguments += " $arg"
        }
    }

    $pinfo = New-Object System.Diagnostics.ProcessStartInfo
    $pinfo.FileName = $command
    $pinfo.RedirectStandardError = $true
    $pinfo.RedirectStandardOutput = $true
    $pinfo.UseShellExecute = $false
    $pinfo.Arguments = $commandArguments
    $p = New-Object System.Diagnostics.Process
    $p.StartInfo = $pinfo
    $p.Start() | Out-Null
    $p.WaitForExit()
    $stdout = $p.StandardOutput.ReadToEnd()
    $stderr = $p.StandardError.ReadToEnd()
    return $stdout + " " + $stderr
}

function VerifyContainers() {
    $output = Invoke-CommandCustom "docker" @("ps", "--format", "{{.Names}}")
    $expectedContainers = @("dapr_scheduler", "dapr_placement", "dapr_redis", "dapr_zipkin")
    
    $containers = $output -split "`n"
   
    $missingContainers = @()

    foreach ($container in $expectedContainers) {
        if ($containers -notcontains $container) {
            $missingContainers += $container
        }
    }
    return $missingContainers
}

function VerifyBinaries() {
    $expectedBinaries = @("daprd.exe", "dashboard.exe")
    $missingBinaries = @()

    $userHomeDirectory = [System.Environment]::GetFolderPath("UserProfile")
    foreach ($binary in $expectedBinaries) {
        if (-not (Test-Path "$userHomeDirectory\.dapr\bin\$binary")) {
            $missingBinaries += $binary
        }
       
    }
    return $missingBinaries
}

##### Test cases

function TestDaprInitSuccess() {
    Write-Output "TestDaprInitSuccess"

    $version = "1.15.0-rc.10"
    $arguments = @("--dashboard-version 0.15.0", "--enable-ha")
    Uninstall-Dapr
    $output = Initialize-Dapr $version $arguments
    if ($output.Contains("Success! Dapr is up and running")) {
        return $true
    }
    else {
        Write-Output "TestDaprInitSuccess failed"
        $output
        exit 1
    }
}

function TestDaprInitWithInvalidRegistry() {
    Write-Output "TestDaprInitWithInvalidRegistry"

    $version = "1.15.0-rc.10"
    $arguments = @("--image-registry someunknownnonexistinghost.io/owner")

    Uninstall-Dapr
    $output = Initialize-Dapr $version $arguments
    $result1 = $output.Contains("No connection could be made because the target machine actively refused it")
    $result2 = $output.Contains("A connection attempt failed because the connected party did not properly respond after a period of time")
    $result3 = $output.Contains("init failed")
    if ($result1 -or $result2 -or $result3) {
        return $true
    }
    else {
        Write-Output "TestDaprInitWithInvalidRegistry failed"
        $output
        exit 1
    }
}

function TestDaprInitWithInvalidArguments() {
    Write-Output "TestDaprInitWithInvalidArguments"

    $version = "1.15.0-rc.10"
    $arguments = @("--from-dir invalid", "--image-registry localhost:5000")

    Uninstall-Dapr
    $output = Initialize-Dapr $version $arguments
    if ($output.Contains("both --image-registry and --from-dir flags cannot be given at the same time")) {
        return $true
    }
    else {
        Write-Output "TestDaprInitWithInvalidArguments failed"
        $output
        exit 1
    }
}

function TestDaprInitAllInPlace() {
    Write-Output "TestDaprInitAllInPlace"
    $version = "1.15.0-rc.10"

    Uninstall-Dapr
    $output = Initialize-Dapr $version
    
    if ($output.Contains("Success! Dapr is up and running")) {
        Write-Output "Dapr initialization succeeded"
    }
    else {
        Write-Output "TestDaprInitAllInPlace failed"
        $output
        exit 1
    }

    $missingContainers = VerifyContainers
    if ($missingContainers.Count -eq 0) {
        Write-Output "All expected containers are running"
    }
    else {
        Write-Output "TestDaprInitAllInPlace"
        Write-Output "The following containers are missing: $missingContainers"
        exit 1
    }

    $missingBinaries = VerifyBinaries
    if ($missingBinaries.Count -eq 0) {
        Write-Output "All expected binaries are installed"
    }
    else {
        Write-Output "TestDaprInitAllInPlace"
        Write-Output "The following binaries are missing: $missingBinaries"
        exit 1
    }
}

TestDaprInitSuccess
TestDaprInitWithInvalidRegistry
TestDaprInitWithInvalidArguments
TestDaprInitAllInPlace

