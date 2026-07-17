# Sub2API 本地开发启动工具

该工具用于 Windows + Docker Desktop 开发环境，可以统一启动 PostgreSQL、Redis、Go 后端和 Vite 前端。

## 最简单的使用方式

在仓库根目录双击：

```text
dev.bat
```

在菜单中按数字选择操作，不需要输入 PowerShell 命令。

常用选择：

- `1`：首次启动或全部服务未运行时使用。
- `2`：修改 Go 后端代码后使用，会重新执行 `go build` 并重启后端。
- `3`：重启 Vite 前端。通常修改 Vue、TypeScript 或 CSS 后会自动热更新，不需要选择此项。
- `4`：重新编译后端并重启前后端。
- `5`：查看 PostgreSQL、Redis、后端和前端状态。
- `6`：查看最近的前后端错误日志。
- `7`：停止前端和后端，PostgreSQL 与 Redis 容器会继续保留。

## 开发代码如何生效

### 前端

前端使用 Vite 开发服务器：

- 修改 `.vue`、`.ts`、`.css` 后通常会自动热更新。
- 页面没有更新时，可以在 `dev.bat` 中选择 `3` 重启前端。

### 后端

后端运行的是编译后的 `.dev/sub2api-local/sub2api.exe`：

- 修改 Go 代码后不会自动生效。
- 在 `dev.bat` 中选择 `2`，脚本会停止旧进程、重新编译并启动新后端。
- 不要在需要更新 Go 代码时使用 `-SkipBuild`。

## 首次配置

如果 `tools/local-dev/local.env.ps1` 不存在，复制示例文件：

```powershell
Copy-Item .\tools\local-dev\local.env.example.ps1 .\tools\local-dev\local.env.ps1
```

至少填写：

- `DatabasePassword`
- `AdminPassword`
- `GoExe`

`GoExe` 示例：

```powershell
GoExe = 'C:\Program Files\Go\bin\go.exe'
```

`local.env.ps1` 已被 Git 忽略，里面可以保存本机开发密码，但不要手动提交该文件。

## 默认地址

- 前端：`http://127.0.0.1:5173`
- 后端：`http://127.0.0.1:4000`
- PostgreSQL：`127.0.0.1:5433`
- Redis：`127.0.0.1:6379`
- 开发数据库：`sub2api_dev`

运行状态、PID、生成的后端程序和日志保存在：

```text
.dev/sub2api-local/
```

## 命令行方式

双击 `dev.bat` 之外，也可以直接使用：

```powershell
.\dev.bat start
.\dev.bat restart-backend
.\dev.bat restart-frontend
.\dev.bat restart
.\dev.bat status
.\dev.bat logs
.\dev.bat stop
```

底层 PowerShell 命令仍然可用：

```powershell
.\tools\local-dev\sub2api-dev.ps1 start
.\tools\local-dev\sub2api-dev.ps1 restart-backend
```

## 验证脚本

```powershell
.\tools\local-dev\sub2api-dev.tests.ps1
```
