[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('menu', 'start', 'restart', 'restart-backend', 'restart-frontend', 'stop', 'status', 'logs', 'help')]
    [string]$Action = 'status',
    [switch]$Json,
    [switch]$SkipBuild,
    [string]$ConfigPath = ''
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($ConfigPath)) {
    $ConfigPath = Join-Path $PSScriptRoot 'local.env.ps1'
}
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$StateDir = Join-Path $RepoRoot '.dev\sub2api-local'
$BackendLog = Join-Path $StateDir 'backend.log'
$BackendErrorLog = Join-Path $StateDir 'backend.error.log'
$FrontendLog = Join-Path $StateDir 'frontend.log'
$FrontendErrorLog = Join-Path $StateDir 'frontend.error.log'
$BackendPidFile = Join-Path $StateDir 'backend.pid'
$FrontendPidFile = Join-Path $StateDir 'frontend.pid'

$Settings = [ordered]@{
    PostgresContainer = 'pgsql-local-5433'
    RedisContainer = 'redis-local'
    DatabaseAdminUser = 'postgres'
    DatabaseHost = '127.0.0.1'
    DatabasePort = 5433
    DatabaseName = 'sub2api_dev'
    DatabaseUser = 'sub2api_dev'
    DatabasePassword = ''
    RedisHost = '127.0.0.1'
    RedisPort = 6379
    RedisPassword = ''
    BackendHost = '127.0.0.1'
    BackendPort = 4000
    FrontendHost = '127.0.0.1'
    FrontendPort = 5173
    AdminEmail = 'admin@sub2api.local'
    AdminPassword = ''
    TotpEncryptionKey = ''
    GoExe = ''
}

if (Test-Path -LiteralPath $ConfigPath) {
    . $ConfigPath
    if ($Sub2ApiDev -isnot [System.Collections.IDictionary]) {
        throw "Configuration file must define a hashtable named `$Sub2ApiDev: $ConfigPath"
    }
    foreach ($key in $Sub2ApiDev.Keys) {
        if (-not $Settings.Contains($key)) { throw "Unknown local development setting: $key" }
        $Settings[$key] = $Sub2ApiDev[$key]
    }
}

function Get-CommandPath {
    param([string]$Name)
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $command) { return $null }
    return $command.Source
}

function Get-ListenerProcessId {
    param([int]$Port)
    $listener = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $listener) { return $null }
    return [int]$listener.OwningProcess
}

function Test-HttpEndpoint {
    param([string]$Url)
    try {
        $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 500
    } catch { return $false }
}

function Get-ContainerState {
    param([string]$Name)
    if (-not (Get-CommandPath 'docker')) { return 'docker-unavailable' }
    $state = & docker inspect -f '{{.State.Status}}' $Name 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace(($state -join ''))) { return 'missing' }
    return ($state -join '').Trim()
}

function Get-StatusObject {
    $backendUrl = "http://$($Settings.BackendHost):$($Settings.BackendPort)"
    $frontendUrl = "http://$($Settings.FrontendHost):$($Settings.FrontendPort)"
    return [ordered]@{
        backend = [ordered]@{
            url = $backendUrl
            port = [int]$Settings.BackendPort
            process_id = Get-ListenerProcessId ([int]$Settings.BackendPort)
            healthy = Test-HttpEndpoint "$backendUrl/health"
        }
        frontend = [ordered]@{
            url = $frontendUrl
            port = [int]$Settings.FrontendPort
            process_id = Get-ListenerProcessId ([int]$Settings.FrontendPort)
            healthy = Test-HttpEndpoint $frontendUrl
        }
        postgres = [ordered]@{
            container = $Settings.PostgresContainer
            state = Get-ContainerState $Settings.PostgresContainer
            host = $Settings.DatabaseHost
            port = [int]$Settings.DatabasePort
            database = $Settings.DatabaseName
            user = $Settings.DatabaseUser
        }
        redis = [ordered]@{
            container = $Settings.RedisContainer
            state = Get-ContainerState $Settings.RedisContainer
            host = $Settings.RedisHost
            port = [int]$Settings.RedisPort
        }
        logs = [ordered]@{
            backend = $BackendLog
            backend_error = $BackendErrorLog
            frontend = $FrontendLog
            frontend_error = $FrontendErrorLog
        }
    }
}

function Show-Status {
    $status = Get-StatusObject
    if ($Json) { $status | ConvertTo-Json -Depth 5; return }
    Write-Host 'Sub2API local development status' -ForegroundColor Cyan
    Write-Host "  Backend : $($status.backend.url)  healthy=$($status.backend.healthy)  pid=$($status.backend.process_id)"
    Write-Host "  Frontend: $($status.frontend.url)  healthy=$($status.frontend.healthy)  pid=$($status.frontend.process_id)"
    Write-Host "  Postgres: $($status.postgres.container)  state=$($status.postgres.state)  db=$($status.postgres.database)  port=$($status.postgres.port)"
    Write-Host "  Redis   : $($status.redis.container)  state=$($status.redis.state)  port=$($status.redis.port)"
}

function Assert-StartConfiguration {
    foreach ($key in @('DatabasePassword', 'AdminPassword')) {
        if ([string]::IsNullOrWhiteSpace([string]$Settings[$key])) {
            throw "Missing $key. Copy local.env.example.ps1 to local.env.ps1 and set local-only credentials."
        }
    }
    foreach ($command in @('docker', 'pnpm.cmd')) {
        if (-not (Get-CommandPath $command)) { throw "Required command not found: $command" }
    }
    if ([string]::IsNullOrWhiteSpace([string]$Settings.GoExe)) { $Settings.GoExe = Get-CommandPath 'go' }
    if (-not $Settings.GoExe -or -not (Test-Path -LiteralPath $Settings.GoExe)) {
        throw 'Go executable was not found. Set GoExe in local.env.ps1.'
    }
}

function Ensure-ContainerRunning {
    param([string]$Name)
    $state = Get-ContainerState $Name
    if ($state -eq 'missing') { throw "Docker container does not exist: $Name" }
    if ($state -eq 'docker-unavailable') { throw 'Docker is unavailable.' }
    if ($state -ne 'running') {
        & docker start $Name | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Failed to start Docker container: $Name" }
    }
}

function Ensure-Database {
    $container = [string]$Settings.PostgresContainer
    $admin = [string]$Settings.DatabaseAdminUser
    $database = [string]$Settings.DatabaseName
    $user = [string]$Settings.DatabaseUser
    $password = ([string]$Settings.DatabasePassword).Replace("'", "''")
    $roleExists = & docker exec $container psql -U $admin -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='$user';"
    if (($roleExists -join '').Trim() -ne '1') {
        & docker exec $container psql -U $admin -d postgres -v ON_ERROR_STOP=1 -c "CREATE ROLE $user LOGIN PASSWORD '$password';" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Failed to create PostgreSQL role: $user" }
    }
    $databaseExists = & docker exec $container psql -U $admin -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$database';"
    if (($databaseExists -join '').Trim() -ne '1') {
        & docker exec $container psql -U $admin -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $database OWNER $user;" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Failed to create PostgreSQL database: $database" }
    }
}

function Wait-Endpoint {
    param([string]$Url, [int]$TimeoutSeconds = 90)
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        if (Test-HttpEndpoint $Url) { return }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for endpoint: $Url"
}

function Start-Backend {
    param([switch]$ForceBuild)
    $healthUrl = "http://$($Settings.BackendHost):$($Settings.BackendPort)/health"
    if (Test-HttpEndpoint $healthUrl) { Write-Host "Backend already available: $healthUrl" -ForegroundColor DarkGray; return }
    if (Get-ListenerProcessId ([int]$Settings.BackendPort)) { throw "Backend port is already occupied: $($Settings.BackendPort)" }
    New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
    $backendExe = Join-Path $StateDir 'sub2api.exe'
    if ($ForceBuild -or -not $SkipBuild -or -not (Test-Path -LiteralPath $backendExe)) {
        Write-Host 'Building backend...' -ForegroundColor Cyan
        Push-Location (Join-Path $RepoRoot 'backend')
        try {
            & $Settings.GoExe build -o $backendExe .\cmd\server
            if ($LASTEXITCODE -ne 0) { throw 'Backend build failed.' }
        } finally { Pop-Location }
    }
    $env:DATA_DIR = $StateDir
    $env:AUTO_SETUP = 'true'
    $env:SERVER_HOST = [string]$Settings.BackendHost
    $env:SERVER_PORT = [string]$Settings.BackendPort
    $env:SERVER_MODE = 'debug'
    $env:DATABASE_HOST = [string]$Settings.DatabaseHost
    $env:DATABASE_PORT = [string]$Settings.DatabasePort
    $env:DATABASE_USER = [string]$Settings.DatabaseUser
    $env:DATABASE_PASSWORD = [string]$Settings.DatabasePassword
    $env:DATABASE_DBNAME = [string]$Settings.DatabaseName
    $env:DATABASE_SSLMODE = 'disable'
    $env:REDIS_HOST = [string]$Settings.RedisHost
    $env:REDIS_PORT = [string]$Settings.RedisPort
    $env:REDIS_PASSWORD = [string]$Settings.RedisPassword
    $env:REDIS_DB = '0'
    $env:ADMIN_EMAIL = [string]$Settings.AdminEmail
    $env:ADMIN_PASSWORD = [string]$Settings.AdminPassword
    if (-not [string]::IsNullOrWhiteSpace([string]$Settings.TotpEncryptionKey)) {
        $env:TOTP_ENCRYPTION_KEY = [string]$Settings.TotpEncryptionKey
    } else {
        Remove-Item Env:TOTP_ENCRYPTION_KEY -ErrorAction SilentlyContinue
    }
    $process = Start-Process -FilePath $backendExe -WorkingDirectory (Join-Path $RepoRoot 'backend') -WindowStyle Hidden -RedirectStandardOutput $BackendLog -RedirectStandardError $BackendErrorLog -PassThru
    Set-Content -LiteralPath $BackendPidFile -Value $process.Id -Encoding ascii
    Wait-Endpoint $healthUrl 120
    Write-Host "Backend started: $healthUrl" -ForegroundColor Green
}

function Start-Frontend {
    $frontendUrl = "http://$($Settings.FrontendHost):$($Settings.FrontendPort)"
    if (Test-HttpEndpoint $frontendUrl) { Write-Host "Frontend already available: $frontendUrl" -ForegroundColor DarkGray; return }
    if (Get-ListenerProcessId ([int]$Settings.FrontendPort)) { throw "Frontend port is already occupied: $($Settings.FrontendPort)" }
    New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
    $arguments = "/c pnpm.cmd dev --host $($Settings.FrontendHost)"
    $launcher = Start-Process -FilePath 'cmd.exe' -ArgumentList $arguments -WorkingDirectory (Join-Path $RepoRoot 'frontend') -WindowStyle Hidden -RedirectStandardOutput $FrontendLog -RedirectStandardError $FrontendErrorLog -PassThru
    Set-Content -LiteralPath $FrontendPidFile -Value $launcher.Id -Encoding ascii
    Wait-Endpoint $frontendUrl 90
    $listenerPid = Get-ListenerProcessId ([int]$Settings.FrontendPort)
    if ($listenerPid) { Set-Content -LiteralPath $FrontendPidFile -Value $listenerPid -Encoding ascii }
    Write-Host "Frontend started: $frontendUrl" -ForegroundColor Green
}

function Stop-PidFileProcess {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return }
    $processId = [int](Get-Content -LiteralPath $Path -Raw)
    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
}

function Stop-Backend {
    Stop-PidFileProcess $BackendPidFile
    $processId = Get-ListenerProcessId ([int]$Settings.BackendPort)
    if ($processId) { Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue }
    Write-Host 'Backend stopped.' -ForegroundColor Green
}

function Stop-Frontend {
    Stop-PidFileProcess $FrontendPidFile
    $processId = Get-ListenerProcessId ([int]$Settings.FrontendPort)
    if ($processId) { Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue }
    Write-Host 'Frontend stopped.' -ForegroundColor Green
}

function Stop-LocalDevelopment {
    Stop-Frontend
    Stop-Backend
    Write-Host 'Local frontend and backend processes stopped.' -ForegroundColor Green
}

function Initialize-LocalDependencies {
    Assert-StartConfiguration
    Ensure-ContainerRunning $Settings.PostgresContainer
    Ensure-ContainerRunning $Settings.RedisContainer
    Ensure-Database
}

function Restart-Backend {
    Initialize-LocalDependencies
    Stop-Backend
    Start-Backend -ForceBuild
    Show-Status
}

function Restart-Frontend {
    Assert-StartConfiguration
    Stop-Frontend
    Start-Frontend
    Show-Status
}

function Restart-LocalDevelopment {
    Initialize-LocalDependencies
    Stop-Frontend
    Stop-Backend
    Start-Backend -ForceBuild
    Start-Frontend
    Show-Status
}

function Show-Logs {
    Write-Host "Backend log       : $BackendLog"
    Write-Host "Backend error log : $BackendErrorLog"
    Write-Host "Frontend log      : $FrontendLog"
    Write-Host "Frontend error log: $FrontendErrorLog"
    foreach ($path in @($BackendErrorLog, $FrontendErrorLog)) {
        if (Test-Path -LiteralPath $path) {
            Write-Host "`n--- $(Split-Path $path -Leaf) ---" -ForegroundColor Cyan
            Get-Content -LiteralPath $path -Tail 30
        }
    }
}

function Show-InteractiveMenu {
    while ($true) {
        Clear-Host
        Write-Host '==================================================' -ForegroundColor Cyan
        Write-Host '              Sub2API 本地开发' -ForegroundColor Cyan
        Write-Host '==================================================' -ForegroundColor Cyan
        Write-Host ''
        Write-Host '  1. 启动全部服务'
        Write-Host '  2. 重新编译并重启后端'
        Write-Host '  3. 重启前端'
        Write-Host '  4. 重新编译并重启前后端'
        Write-Host '  5. 查看服务状态'
        Write-Host '  6. 查看最近错误日志'
        Write-Host '  7. 停止前端和后端'
        Write-Host '  0. 退出'
        Write-Host ''

        $choice = Read-Host '请选择操作 [0-7]'
        if ($choice -eq '0') {
            return
        }

        try {
            switch ($choice) {
                '1' {
                    Initialize-LocalDependencies
                    Start-Backend
                    Start-Frontend
                    Show-Status
                }
                '2' { Restart-Backend }
                '3' { Restart-Frontend }
                '4' { Restart-LocalDevelopment }
                '5' { Show-Status }
                '6' { Show-Logs }
                '7' { Stop-LocalDevelopment }
                default {
                    Write-Host '输入无效，请输入 0 到 7。' -ForegroundColor Yellow
                    [void](Read-Host '按回车键返回菜单')
                    continue
                }
            }
            Write-Host ''
            Write-Host '操作执行成功。' -ForegroundColor Green
        }
        catch {
            Write-Host ''
            Write-Host "操作执行失败：$($_.Exception.Message)" -ForegroundColor Red
        }

        [void](Read-Host '按回车键返回菜单')
    }
}

function Show-Help {
    @"
Sub2API local development helper

Usage:
  .\tools\local-dev\sub2api-dev.ps1 start [-SkipBuild]
  .\tools\local-dev\sub2api-dev.ps1 restart
  .\tools\local-dev\sub2api-dev.ps1 restart-backend
  .\tools\local-dev\sub2api-dev.ps1 restart-frontend
  .\tools\local-dev\sub2api-dev.ps1 stop
  .\tools\local-dev\sub2api-dev.ps1 status [-Json]
  .\tools\local-dev\sub2api-dev.ps1 logs

Commands:
  start             Start Docker dependencies, create the development database, then start backend and frontend.
  restart           Rebuild the backend and restart both backend and frontend.
  restart-backend   Rebuild and restart only the Go backend. Use this after changing Go code.
  restart-frontend  Restart only the Vite frontend. Vue changes normally hot-reload without this.
  stop              Stop frontend and backend processes started for local development.
  status            Show container, process, port, and health status.
  logs              Show log paths and recent error output.

Local configuration:
  Copy tools\local-dev\local.env.example.ps1 to tools\local-dev\local.env.ps1.
  The local.env.ps1 file is ignored by Git.
"@ | Write-Output
}

switch ($Action) {
    'menu' { Show-InteractiveMenu }
    'start' {
        Initialize-LocalDependencies
        Start-Backend
        Start-Frontend
        Show-Status
    }
    'restart' { Restart-LocalDevelopment }
    'restart-backend' { Restart-Backend }
    'restart-frontend' { Restart-Frontend }
    'stop' { Stop-LocalDevelopment }
    'status' { Show-Status }
    'logs' { Show-Logs }
    'help' { Show-Help }
}
