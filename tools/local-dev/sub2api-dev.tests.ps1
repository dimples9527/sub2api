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
foreach ($command in @('start', 'restart', 'restart-backend', 'restart-frontend', 'stop', 'status', 'logs')) {
    if ($helpText -notmatch [regex]::Escape($command)) {
        throw "Help output does not mention command: $command"
    }
}

$rootLauncher = Join-Path (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path 'dev.bat'
if (-not (Test-Path -LiteralPath $rootLauncher)) {
    throw "Expected root development launcher at $rootLauncher"
}

$launcherSource = Get-Content -LiteralPath $rootLauncher -Raw
if ($launcherSource -notmatch [regex]::Escape('sub2api-dev.ps1" %*')) {
    throw 'Root launcher does not forward command-line arguments.'
}

if ($launcherSource -notmatch 'sub2api-dev\.ps1" menu') {
    throw 'Root launcher does not open the PowerShell menu by default.'
}

$scriptSource = Get-Content -LiteralPath $scriptPath -Raw -Encoding UTF8
foreach ($text in @('本地开发', '启动全部服务', '重新编译并重启后端', '操作执行成功', '输入无效')) {
    if ($scriptSource -notmatch [regex]::Escape($text)) {
        throw "PowerShell menu does not contain Chinese text: $text"
    }
}

foreach ($text in @('TotpEncryptionKey', 'TOTP_ENCRYPTION_KEY')) {
    if ($scriptSource -notmatch [regex]::Escape($text)) {
        throw "Local development script does not support fixed encryption key: $text"
    }
}

$exampleConfig = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'local.env.example.ps1') -Raw -Encoding UTF8
if ($exampleConfig -notmatch 'TotpEncryptionKey') {
    throw 'Example local configuration does not document TotpEncryptionKey.'
}

$previousErrorActionPreference = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
try {
    $launcherStatusOutput = & cmd.exe /d /c "`"$rootLauncher`" status" 2>&1
} finally {
    $ErrorActionPreference = $previousErrorActionPreference
}
if ($LASTEXITCODE -ne 0) {
    throw "Root launcher status command failed: $($launcherStatusOutput -join [Environment]::NewLine)"
}
if (($launcherStatusOutput -join [Environment]::NewLine) -notmatch 'Backend') {
    throw 'Root launcher status output does not contain Backend status.'
}

$invalidProcess = Start-Process -FilePath 'cmd.exe' -ArgumentList @('/d', '/c', "`"$rootLauncher`" invalid-command") -WindowStyle Hidden -Wait -PassThru
if ($invalidProcess.ExitCode -eq 0) {
    throw 'Root launcher must preserve a non-zero exit code for invalid commands.'
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
