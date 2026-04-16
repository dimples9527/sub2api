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

校验并重载：

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl enable --now caddy
sudo systemctl reload caddy
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

## 十、建议继续优化的点

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
