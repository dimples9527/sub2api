# CentOS 域名 + HTTPS 优化部署指南（Caddy 方案）

本文适用于以下场景：

- 你已经按 `deploy/DEPLOY_CENTOS_NO_DOMAIN_CN.md` 成功部署
- 当前服务已经可以通过 `http://服务器IP:8080` 访问
- 你现在有域名 `api.sunshinefastlink.top`
- 希望把部署从“直接暴露应用端口”升级为“域名 + HTTPS + 反向代理”

推荐目标架构：

```text
Internet
  -> 80/443
  -> Caddy
  -> 127.0.0.1:8080
  -> Docker sub2api
```

这样做的好处：

- 外网不再直接暴露应用 `8080`
- 自动签发和续期 HTTPS 证书
- WebSocket、SSE、流式响应都由反向代理统一处理
- 后续更方便接入限流、日志、WAF/CDN

## 一、先确认 DNS

先确保域名已经解析到你的服务器公网 IP：

```bash
ping api.sunshinefastlink.top
```

或者：

```bash
nslookup api.sunshinefastlink.top
```

如果解析还没生效，先不要继续申请证书。

## 二、把应用端口收回到本机

编辑：

`/opt/sub2api/runtime/.env`

把：

```env
BIND_HOST=0.0.0.0
SERVER_PORT=8080
```

改成：

```env
BIND_HOST=127.0.0.1
SERVER_PORT=8080
```

这样容器仍然映射到宿主机 `8080`，但只允许本机访问，外网无法直接打到应用。

## 三、建议补一个运行时 config.yaml

如果你后续要用邮件重置密码、支付回调、真实客户端 IP 识别，建议显式配置运行时 `config.yaml`。

创建：

`/opt/sub2api/runtime/config.yaml`

示例：

```yaml
server:
  frontend_url: "https://api.sunshinefastlink.top"
  trusted_proxies:
    - "172.17.0.0/16"
```

说明：

- `frontend_url` 用于生成邮件里的外部链接
- `trusted_proxies` 用于让应用信任反向代理传来的 `X-Forwarded-For`

注意：

- `trusted_proxies` 建议填你 Docker bridge 实际网段；上面 `172.17.0.0/16` 是常见默认值
- 如果你暂时不关心后台里的真实客户端 IP，可以先不配这一项

然后编辑：

`/opt/sub2api/runtime/docker-compose.yml`

把这行取消注释：

```yaml
      - ./config.yaml:/app/data/config.yaml
```

## 四、重建并确认本机访问正常

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
curl http://127.0.0.1:8080/health
```

返回 `200` 说明应用本机访问正常。

此时从外网直接访问 `http://服务器IP:8080` 应该失败或不可达，这是预期结果。

## 五、安装 Caddy

如果服务器还没装 Caddy，可以按官方仓库方式安装。CentOS / Rocky / AlmaLinux 常见命令如下：

```bash
sudo dnf install -y 'dnf-command(copr)'
sudo dnf copr enable @caddy/caddy -y
sudo dnf install -y caddy
```

安装后先不要急着启动，先写配置。

## 六、部署 Caddy 配置

你的仓库里已经有：

`deploy/Caddyfile`

可以直接用它作为基础配置。复制到系统目录：

```bash
sudo cp /opt/sub2api/repo/deploy/Caddyfile /etc/caddy/Caddyfile
```

然后检查其中这几个关键点：

1. 域名是：

```caddyfile
api.sunshinefastlink.top {
```

2. 反向代理目标是：

```caddyfile
reverse_proxy localhost:8080
```

3. 已经包含：

- gzip/zstd 压缩
- 静态资源缓存
- TLS 配置
- 健康检查
- WebSocket/流式请求代理能力
- 访问日志输出到 systemd journal，可用 `journalctl -u caddy` 查看

日志配置应保持为：

```caddyfile
log {
    output stdout
    format json
    level INFO
}
```

不要把访问日志写到 `/var/log/caddy/sub2api.log`。在 CentOS/RHEL 系系统下，Caddy 通常以 `caddy` 用户运行，文件日志容易被目录权限或 SELinux 拦住，导致服务启动失败。

校验并重载：

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl enable --now caddy
sudo systemctl reload caddy
```

如果 `caddy validate` 或 `systemctl start caddy` 报错类似：

```text
setting up custom log 'log0': opening log writer using &logging.FileWriter{Filename:"/var/log/caddy/sub2api.log"}
```

说明当前 `/etc/caddy/Caddyfile` 还在写文件日志。先确认旧配置是否存在：

```bash
sudo grep -n "/var/log/caddy/sub2api.log\|output file\|output stdout" /etc/caddy/Caddyfile
```

如果看到 `output file /var/log/caddy/sub2api.log`，先备份，再把文件日志替换成 `stdout`：

```bash
sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak.$(date +%Y%m%d%H%M%S)
sudo perl -0pi -e 's/output file \/var\/log\/caddy\/sub2api\.log\s*\{\s*roll_size\s+50mb\s*roll_keep\s+10\s*roll_keep_for\s+720h\s*\}/output stdout/s' /etc/caddy/Caddyfile
sudo grep -n "/var/log/caddy/sub2api.log\|output file\|output stdout" /etc/caddy/Caddyfile
```

替换后，日志块应为：

```caddyfile
log {
    output stdout
    format json
    level INFO
}
```

然后重新校验并启动：

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl restart caddy
```

如果仍然启动失败，查看完整日志：

```bash
sudo systemctl status caddy --no-pager -l
sudo journalctl -u caddy -n 80 --no-pager
```

如果是首次启动，Caddy 会自动申请 HTTPS 证书。

## 七、开放 80/443，关闭公网 8080

如果你使用 firewalld：

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --remove-port=8080/tcp
sudo firewall-cmd --reload
```

如果云厂商还有安全组，也要同步：

- 放行 `80/tcp`
- 放行 `443/tcp`
- 关闭 `8080/tcp`

## 八、验证访问

浏览器访问：

```text
https://api.sunshinefastlink.top
```

命令行验证：

```bash
curl -I https://api.sunshinefastlink.top/health
```

检查 Caddy 日志：

```bash
sudo journalctl -u caddy -f
```

检查应用日志：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml logs -f sub2api
```

## 九、后续更新方式

以后你的更新流程基本不变，仍然是：

1. 本地重新打包源码
2. `scp` 上传到服务器
3. 覆盖 `/opt/sub2api/repo`
4. 在 `/opt/sub2api/runtime` 重新执行：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

只要：

- `runtime/.env` 继续保持 `BIND_HOST=127.0.0.1`
- `Caddy` 持续代理到 `localhost:8080`

你的域名层不需要每次重配。

## 十、迁移服务器时的无感切换方式

如果你已经在旧服务器上用 Caddy 跑着域名，现在要切换到新服务器，不要只改 DNS 后等待解析。DNS 有缓存，部分用户会继续访问旧 IP，时间从几分钟到数小时不等，取决于 TTL、运营商缓存和客户端缓存。

更稳的切换拓扑是：

```text
DNS 还缓存旧 IP 的用户
  -> 旧服务器 Caddy
  -> 新服务器:8080
  -> 新服务器应用和数据库

DNS 已解析到新 IP 的用户
  -> 新服务器 Caddy
  -> 127.0.0.1:8080
  -> 新服务器应用和数据库
```

也就是说，切换期间旧服务器只保留 Caddy，旧应用停止写入；所有请求最终都打到新服务器。

### 10.1 新服务器先启动应用

先按 `deploy/DEPLOY_CENTOS_NO_DOMAIN_CN.md` 完成新服务器部署和数据迁移，并确保新服务器本机正常：

```bash
curl http://127.0.0.1:8080/health
```

迁移期间，如果旧服务器 Caddy 要通过公网访问新服务器的 `8080`，新服务器 `.env` 需要临时允许外部访问：

```env
BIND_HOST=0.0.0.0
SERVER_PORT=8080
```

然后只放行旧服务器访问新服务器的 `8080`，不要对全网开放：

```bash
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="OLD_IP/32" port port="8080" protocol="tcp" accept'
sudo firewall-cmd --reload
```

云厂商安全组也要同步放行：仅允许 `OLD_IP` 访问新服务器 `8080/tcp`。

### 10.2 新服务器配置正式 Caddy

新服务器的 `/etc/caddy/Caddyfile` 仍然使用正式域名：

```caddyfile
YOUR_DOMAIN {
    reverse_proxy localhost:8080

    log {
        output stdout
        format json
        level INFO
    }
}
```

把 `YOUR_DOMAIN` 替换成你的域名，例如 `api.sunshinefastlink.top`。

校验并启动：

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl enable --now caddy
sudo systemctl reload caddy
```

如果 DNS 此时还指向旧服务器，新服务器 Caddy 可能暂时申请不到证书，这是正常现象。等 DNS A 记录切到新 IP，并且新服务器 `80/443` 已放行后，Caddy 才能通过 ACME 验证并签发证书。不要在短时间内反复删除 Caddy 证书存储目录重试，以免触发签发频率限制。

### 10.3 停止旧服务器应用，只保留旧 Caddy

在旧服务器停止应用和数据库容器，避免旧数据库继续被写入：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml down
```

如果你还有最终增量数据要同步，必须在旧应用停止后再做最后一次同步。同步完成后，不要再启动旧应用。

### 10.4 把旧服务器 Caddy 临时反代到新服务器

在旧服务器编辑 `/etc/caddy/Caddyfile`，把同一个域名的反代目标改为新服务器 IP：

```caddyfile
YOUR_DOMAIN {
    reverse_proxy http://NEW_IP:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
        header_up X-Forwarded-Host {host}
    }

    log {
        output stdout
        format json
        level INFO
    }
}
```

注意不要写成：

```caddyfile
reverse_proxy https://YOUR_DOMAIN
```

因为旧服务器自己解析 `YOUR_DOMAIN` 时，可能仍然解析回旧 IP，导致 Caddy 反代到自己，形成循环。

重载旧服务器 Caddy：

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

验证旧服务器能把请求转到新服务器：

```bash
curl -H 'Host: YOUR_DOMAIN' http://127.0.0.1/health
```

### 10.5 修改 DNS 到新服务器 IP

确认新服务器和旧服务器 Caddy 都能访问后，再把 DNS A 记录从旧 IP 改到新 IP。

切换前确认新服务器已经放行：

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

建议：

- 切换前提前把 DNS TTL 调低，例如 `60` 或 `120` 秒
- 切换当天保留旧服务器 Caddy 至少 24 小时
- 观察旧服务器 Caddy 日志，如果仍有访问，说明仍有用户或解析器命中旧 IP

查看旧服务器 Caddy 日志：

```bash
sudo journalctl -u caddy -f
```

### 10.6 两台服务器配置同一个 Caddy 域名会不会有问题

两台服务器的 Caddy 可以同时配置同一个域名。真正决定请求进哪台服务器的是 DNS 解析结果和客户端缓存：

- 解析到旧 IP 的用户会进旧 Caddy，再被转发到新服务器
- 解析到新 IP 的用户会进新 Caddy，直接访问新服务器本机应用
- 两边都配置同一个域名，不会让请求自动分叉，也不会让 Caddy 彼此抢流量

需要注意的是证书签发。Caddy 会按域名自动申请证书，如果你反复重装、清空 Caddy 存储目录并重复申请，可能触发 CA 频率限制。迁移时尽量保留 Caddy 存储，避免短时间重复删证书、重申请。

### 10.7 观察期结束后的收尾

确认 DNS 已基本收敛、旧服务器 Caddy 日志没有正常用户请求后，再做收尾：

1. 新服务器把 `.env` 改回只监听本机：

```env
BIND_HOST=127.0.0.1
SERVER_PORT=8080
```

2. 重启新服务器应用：

```bash
cd /opt/sub2api/runtime
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d
```

3. 删除新服务器上临时放行旧服务器访问 `8080` 的防火墙和安全组规则。

4. 旧服务器可以停掉 Caddy，或保留一段时间只做兜底代理。

### 10.8 回滚方式

如果新服务器出现问题，回滚思路是把入口重新指回旧服务器，但前提是旧服务器数据仍然可用且没有被新旧两边同时写乱。

推荐回滚步骤：

1. 停止新服务器应用，防止继续写新数据库。
2. 如果旧服务器数据库仍是切换前的完整数据，启动旧服务器应用。
3. 把旧服务器 Caddy 从 `reverse_proxy http://NEW_IP:8080` 改回 `reverse_proxy localhost:8080`。
4. 把 DNS A 记录改回旧 IP。
5. 检查旧服务器 `/health`、登录后台和关键业务数据。

如果新服务器已经产生了新的有效数据，不能简单回滚到旧数据库，否则会丢数据。此时应先导出或合并新数据，再决定恢复方案。

## 十一、建议继续优化的点

如果你准备长期跑生产，建议再补这几项：

1. 开启 `server.frontend_url`
   这样密码重置、支付回调等外部链接会使用你的正式域名。

2. 明确 `server.trusted_proxies`
   这样后台记录的客户端 IP 更准确。

3. 收紧 URL 白名单
   可在 `config.yaml` 中开启 `security.url_allowlist.enabled`，仅允许必要上游域名。

4. 做数据备份
   至少备份：
   - `/opt/sub2api/runtime/data`
   - `/opt/sub2api/runtime/postgres_data`
   - `/opt/sub2api/runtime/redis_data`

5. 前置 CDN / WAF
   如果后续流量增大，可以把 Caddy 前面再接 Cloudflare 或云厂商 WAF。

## 最小改造总结

如果你想按最小改动完成升级，实际上只要做这 4 件事：

1. 把 `/opt/sub2api/runtime/.env` 的 `BIND_HOST` 改成 `127.0.0.1`
2. 安装并启用 Caddy
3. 使用域名 `api.sunshinefastlink.top` 反代到 `localhost:8080`
4. 开放 `80/443`，关闭公网 `8080`

这样就已经比“直接暴露 8080”更稳、更安全，也更方便长期运维。
