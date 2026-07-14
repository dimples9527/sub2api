# Sub2API 本地联调脚本

该目录用于复用当前 Windows + Docker Desktop 本地联调环境，避免每次重新查找 PostgreSQL、Redis、Go 路径和启动参数。

## 当前环境

- PostgreSQL 容器：`pgsql-local-5433`
- PostgreSQL 地址：`127.0.0.1:5433`
- 开发数据库：`sub2api_dev`
- 开发数据库用户：`sub2api_dev`
- Redis 容器：`redis-local`
- Redis 地址：`127.0.0.1:6379`
- 后端地址：`http://127.0.0.1:4000`
- 前端地址：`http://127.0.0.1:5173`
- 默认管理员邮箱：`admin@sub2api.local`

密码和本机 Go 路径保存在 `local.env.ps1`。该文件已被 Git 忽略，不得提交。

## 使用方式

在仓库根目录执行：

```powershell
# 启动或复用 Docker 依赖，幂等创建数据库，然后启动前后端
.\tools\local-dev\sub2api-dev.ps1 start

# 后端已构建且只想快速重启时
.\tools\local-dev\sub2api-dev.ps1 start -SkipBuild

# 查看容器、端口、进程和健康状态
.\tools\local-dev\sub2api-dev.ps1 status

# 输出机器可读状态
.\tools\local-dev\sub2api-dev.ps1 status -Json

# 查看日志路径和最近错误
.\tools\local-dev\sub2api-dev.ps1 logs

# 停止本地前后端，不停止 PostgreSQL 和 Redis
.\tools\local-dev\sub2api-dev.ps1 stop
```

## 首次配置

如果 `local.env.ps1` 不存在：

```powershell
Copy-Item .\tools\local-dev\local.env.example.ps1 .\tools\local-dev\local.env.ps1
```

然后填写：

- `DatabasePassword`
- `AdminPassword`
- `GoExe`，仓库需要指定兼容版本的 Go 可执行文件

脚本会自动完成：

1. 检查并启动现有 PostgreSQL、Redis 容器。
2. 幂等创建 PostgreSQL 角色和 `sub2api_dev` 数据库。
3. 将运行数据、配置、PID 和日志写入 `.dev/sub2api-local`。
4. 构建并启动 Go 后端。
5. 启动 Vite 前端。
6. 等待后端 `/health` 和前端首页可访问。

首次登录新数据库后，后台会要求管理员确认合规声明。该确认必须在浏览器中由使用者完成。

## 验证脚本

```powershell
.\tools\local-dev\sub2api-dev.tests.ps1
```
