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
