# CentOS 无域名源码构建部署指南（SCP 方式）

本文适用于以下场景：

- 服务器系统：CentOS
- 已安装 Docker / Docker Compose
- 暂时没有域名，不配置 HTTPS
- 需要使用自己修改后的前端和后端源码
- 不使用 Git，改为通过 `scp` 上传源码
- PostgreSQL / Redis 由 Docker Compose 一并启动

此方案不做前后端分离。前端会在镜像构建阶段自动打包，并嵌入后端服务中，最终只运行一个 `sub2api` Web 服务。

## 部署结构

建议使用以下目录结构：

```text
/opt/sub2api/
├── repo/       # 你的修改后源码
└── runtime/    # 运行目录（compose/.env/数据）
```

## 一、准备目录

```bash
sudo mkdir -p /opt/sub2api
sudo chown -R $USER:$USER /opt/sub2api
cd /opt/sub2api
```

## 二、通过 SCP 上传源码

推荐做法不是直接 `scp -r` 整个项目目录，而是：

1. 在本地先打一个源码包
2. 用 `scp` 上传到服务器
3. 在服务器解压到 `/opt/sub2api/repo`

这样更稳定，也更容易避免把本地无关文件一并传上去。

### 2.1 本地打包源码

在你本地项目根目录执行：

```bash
tar --exclude=.git --exclude=frontend/node_modules --exclude=frontend/dist --exclude=backend/internal/web/dist --exclude=backend/sub2api --exclude=backend/sub2api.exe -czf sub2api-src.tar.gz .
```

如果你在 Windows PowerShell 中操作，并且系统自带 `tar`，也可以直接执行上面的命令。

### 2.2 上传到服务器

在本地执行：

```bash
scp sub2api-src.tar.gz <服务器用户>@<服务器IP>:/opt/sub2api/
```

例如：

```bash
scp sub2api-src.tar.gz root@154.201.90.9:/opt/sub2api/
```

### 2.3 在服务器解压源码

登录服务器后执行：

```bash
cd /opt/sub2api
rm -rf repo
mkdir -p repo
tar -xzf sub2api-src.tar.gz -C repo
```

完成后，源码目录应为：

```text
/opt/sub2api/repo
```

## 三、准备运行目录

```bash
mkdir -p /opt/sub2api/runtime/data
mkdir -p /opt/sub2api/runtime/postgres_data
mkdir -p /opt/sub2api/runtime/redis_data

cp /opt/sub2api/repo/deploy/docker-compose.local.yml /opt/sub2api/runtime/docker-compose.yml
```

## 四、生成密钥

执行两次：

```bash
openssl rand -hex 32
```

用途：

- 第一串用于 `JWT_SECRET`
- 第二串用于 `TOTP_ENCRYPTION_KEY`

## 五、创建运行配置

创建文件 `/opt/sub2api/runtime/.env`：

```bash
cat > /opt/sub2api/runtime/.env <<'EOF'
BIND_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_MODE=release
RUN_MODE=standard
TZ=Asia/Shanghai

POSTGRES_USER=sub2api
POSTGRES_PASSWORD=改成你的数据库强密码
POSTGRES_DB=sub2api
DATABASE_PORT=5432

REDIS_PASSWORD=

ADMIN_EMAIL=改成你的邮箱
ADMIN_PASSWORD=改成你的后台强密码

JWT_SECRET=改成刚刚生成的第一串
TOTP_ENCRYPTION_KEY=改成刚刚生成的第二串
EOF
```

至少需要替换以下值：

- `POSTGRES_PASSWORD`
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`
- `JWT_SECRET`
- `TOTP_ENCRYPTION_KEY`

## 六、创建自定义构建覆盖文件

为了让 Compose 使用你当前服务器上的源码构建镜像，而不是直接拉官方镜像，创建：

`/opt/sub2api/runtime/docker-compose.build.yml`

```bash
cat > /opt/sub2api/runtime/docker-compose.build.yml <<'EOF'
services:
  sub2api:
    image: sub2api-custom:local
    build:
      context: ../repo
      dockerfile: Dockerfile
EOF
```

## 七、处理 SELinux（CentOS 常见）

如果服务器启用了 SELinux，建议先执行：

```bash
sudo chcon -Rt svirt_sandbox_file_t /opt/sub2api/runtime || true
```

如果后续还要挂载其他目录，也要对对应目录做相同处理。

## 八、启动服务

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

如果构建停在下面这个错误：

```text
ERR_PNPM_IGNORED_BUILDS Ignored build scripts: esbuild, vue-demi
Run "pnpm approve-builds" to pick which dependencies should be allowed to run scripts.
```

不要在服务器里手动执行 `pnpm approve-builds`。这是交互式命令，不适合 Docker 构建。

正确处理方式是：确认你上传到服务器的源码里包含最新的根目录 `Dockerfile`、`frontend/package.json` 和 `frontend/.npmrc`，然后重新执行“日常更新”里的打包、上传、解压和重建流程。

如果构建日志仍然显示类似下面的信息：

```text
frontend-builder 7/7
Dockerfile:32
COPY frontend/ ./
RUN pnpm run build
Could not resolve "../../../../docs/legal/admin-compliance.zh.md?raw"
```

说明服务器上的 `/opt/sub2api/repo/Dockerfile` 还是旧版本，或者你上传的源码包不是最新代码。先在服务器执行：

```bash
cd /opt/sub2api/repo
grep -n "COPY docs/legal\|pnpm@9\|RUN pnpm run build" Dockerfile
ls -l docs/legal
```

正常应该能看到：

```text
COPY docs/legal/ /app/docs/legal/
RUN pnpm run build
```

并且 `docs/legal` 目录下应该有 `admin-compliance.zh.md` 和 `admin-compliance.en.md`。如果看不到，回到本地项目根目录重新打包并上传，不要复用旧的 `sub2api-src.tar.gz`。

如果服务器仍然使用了旧构建缓存，可以强制重建一次：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml build --no-cache sub2api
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d
```

查看状态：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml ps
```

查看日志：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml logs -f sub2api
```

## 九、验证服务

先在服务器本机验证：

```bash
curl http://127.0.0.1:8080/health
```

如果返回 `200`，说明服务已正常启动。

## 十、开放防火墙

开放 `8080` 端口：

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

然后通过浏览器访问：

```text
http://你的服务器IP:8080
```

## 服务器迁移和数据目录迁移

如果你已经在一台服务器上跑起来，现在要迁移到另一台服务器，核心原则是：**最终只能有一个应用实例接收写入请求**。不要让旧服务器和新服务器同时对各自的本地数据库写入，否则会产生两份不一致的数据，后续很难合并。

需要迁移的关键路径是：

```text
/opt/sub2api/repo
/opt/sub2api/runtime/.env
/opt/sub2api/runtime/config.yaml
/opt/sub2api/runtime/data
/opt/sub2api/runtime/postgres_data
/opt/sub2api/runtime/redis_data
```

其中：

- `repo` 是源码，可重新上传最新源码包，也可以从旧服务器复制
- `.env` 包含数据库密码、管理员账号、JWT 密钥、TOTP 加密密钥，必须保留
- `config.yaml` 如果你启用了运行时配置，也要一起迁移
- `data` 是应用本地数据
- `postgres_data` 是 PostgreSQL 物理数据目录
- `redis_data` 是 Redis 持久化数据目录

### 迁移推荐流程

1. 在新服务器按本文档先准备好 Docker、目录、源码和运行配置。
2. 先不要让新服务器对外提供正式流量。
3. 在旧服务器停止写入，可以临时关闭旧应用或进入维护窗口。
4. 停止旧服务器容器：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml down
```

5. 在旧服务器打包运行数据：

```bash
cd /opt/sub2api
tar --ignore-failed-read -czf sub2api-runtime-data.tar.gz \
  runtime/.env \
  runtime/config.yaml \
  runtime/data \
  runtime/postgres_data \
  runtime/redis_data
```

如果你的部署没有 `runtime/config.yaml`，`--ignore-failed-read` 会跳过它。

6. 上传到新服务器：

```bash
scp /opt/sub2api/sub2api-runtime-data.tar.gz <新服务器用户>@<新服务器IP>:/opt/sub2api/
```

7. 在新服务器停止容器，再备份并替换运行数据。不要直接删除整个 `runtime` 目录，因为里面还有新服务器使用的 `docker-compose.yml` 和 `docker-compose.build.yml`：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml down

cd /opt/sub2api
BACKUP_DIR="runtime.bak.$(date +%Y%m%d%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp -a runtime/.env "$BACKUP_DIR"/ 2>/dev/null || true
cp -a runtime/config.yaml "$BACKUP_DIR"/ 2>/dev/null || true
cp -a runtime/data "$BACKUP_DIR"/ 2>/dev/null || true
cp -a runtime/postgres_data "$BACKUP_DIR"/ 2>/dev/null || true
cp -a runtime/redis_data "$BACKUP_DIR"/ 2>/dev/null || true

rm -rf runtime/data runtime/postgres_data runtime/redis_data
tar -xzf sub2api-runtime-data.tar.gz -C /opt/sub2api
```

8. 如果启用了 SELinux，恢复目录标签：

```bash
sudo chcon -Rt svirt_sandbox_file_t /opt/sub2api/runtime || true
```

9. 启动新服务器：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
curl http://127.0.0.1:8080/health
```

10. 确认新服务器后台、登录、关键接口和数据都正常后，再切换入口流量。

### 新服务器已经启动后，还能不能直接替换数据库文件

可以替换，但不能在 PostgreSQL 正在运行时直接覆盖 `postgres_data`。

正确做法是先停掉新服务器容器，再替换完整目录：

旧服务器先在 PostgreSQL 停止后打包：

```bash
cd /opt/sub2api/runtime
tar -czf /opt/sub2api/old-postgres-data.tar.gz postgres_data
```

上传到新服务器后执行：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml down

cd /opt/sub2api/runtime
mv postgres_data postgres_data.bak.$(date +%Y%m%d%H%M%S)
tar -xzf /opt/sub2api/old-postgres-data.tar.gz -C /opt/sub2api/runtime

sudo chcon -Rt svirt_sandbox_file_t /opt/sub2api/runtime/postgres_data || true

docker compose -f docker-compose.yml -f docker-compose.build.yml up -d postgres
docker compose -f docker-compose.yml -f docker-compose.build.yml logs -f postgres
```

确认 PostgreSQL 正常后，再启动完整服务：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d
curl http://127.0.0.1:8080/health
```

注意：

- `postgres_data` 必须是旧服务器停止 PostgreSQL 后复制出来的完整目录
- 新旧服务器使用的 PostgreSQL 主版本必须一致；本文档当前 compose 使用 `postgres:18-alpine`
- 如果 PostgreSQL 主版本不同，优先使用 `pg_dump` / `pg_restore` 做逻辑迁移，不要直接复制物理目录
- 替换前保留 `postgres_data.bak.*`，观察期结束后再清理
- 迁移窗口内不要让旧应用继续写旧数据库，也不要让新应用先写新数据库后又覆盖数据库目录

### 更稳妥的数据库迁移方式

如果你不确定 PostgreSQL 版本是否一致，或旧服务器数据目录状态不明确，更推荐逻辑备份：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml exec postgres \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -f /tmp/sub2api.dump
docker cp sub2api-postgres:/tmp/sub2api.dump /opt/sub2api/sub2api.dump
```

再把 `/opt/sub2api/sub2api.dump` 上传到新服务器，用 `pg_restore` 恢复。逻辑备份比直接复制 `postgres_data` 慢一些，但跨小版本、排查问题和恢复时更稳。

## 日常更新

以后每次更新代码，按下面流程执行。

### 1. 本地重新打包

```bash
tar --exclude=.git \
    --exclude=frontend/node_modules \
    --exclude=frontend/dist \
    --exclude=backend/internal/web/dist \
    --exclude=backend/sub2api \
    --exclude=backend/sub2api.exe \
    -czf sub2api-src.tar.gz .
```

### 2. 本地上传到服务器

```bash
scp sub2api-src.tar.gz <服务器用户>@<服务器IP>:/opt/sub2api/
```

### 3. 服务器替换源码并重建

```bash
cd /opt/sub2api
rm -rf repo
mkdir -p repo
tar -xzf sub2api-src.tar.gz -C repo

cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

这会重新构建镜像，并将你的前端修改一起打包进最终服务。

## 回滚

如果某次更新有问题，最简单的回滚方式是：

1. 在你本地找到上一个可用版本的源码
2. 重新打包 `sub2api-src.tar.gz`
3. 再次通过 `scp` 上传
4. 在服务器覆盖 `repo` 并重新构建

也就是重复“日常更新”这一节的流程，但上传的是旧版本源码包。

## 后续升级到域名和 HTTPS

当前方案适合“先用 IP 跑起来”。

等你后续有域名后，建议这样调整：

1. 将 `.env` 中的 `BIND_HOST` 从 `0.0.0.0` 改为 `127.0.0.1`
2. 外部增加 Caddy 或 Nginx 反向代理到 `127.0.0.1:8080`
3. 开放 `80/443`
4. 配置域名和 HTTPS
5. 关闭公网 `8080`

这样可以从当前部署平滑升级，无需重做应用层部署。

## 备注

- 本方案默认使用 `docker-compose.local.yml`，数据存放在本地目录，便于备份和迁移
- 本方案不会做前后端分离部署
- 前端个性化修改会在镜像构建时自动被打包进最终服务
- 推荐上传源码包，不要上传本地 `node_modules`、构建产物或 `.git` 目录
