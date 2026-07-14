$ErrorActionPreference = 'Stop'

$scriptPath = Join-Path $PSScriptRoot 'sub2api-dev.ps1'
if (-not (Test-Path -LiteralPath $scriptPath)) {
    throw "Expected local development script at $scriptPath"
}

$helpOutput = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $scriptPath help 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Help command failed: $($helpOutput -join [Environment]::NewLine)"
}

$helpText = $helpOutput -join [Environment]::NewLine
foreach ($command in @('start', 'stop', 'status', 'logs')) {
    if ($helpText -notmatch [regex]::Escape($command)) {
        throw "Help output does not mention command: $command"
    }
}

$statusOutput = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $scriptPath status -Json 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Status command failed: $($statusOutput -join [Environment]::NewLine)"
}

$status = $statusOutput -join [Environment]::NewLine | ConvertFrom-Json
foreach ($property in @('backend', 'frontend', 'postgres', 'redis')) {
    if ($null -eq $status.$property) {
        throw "Status JSON does not contain property: $property"
    }
}

Write-Output 'LOCAL_DEV_SCRIPT_TESTS_PASSED'
