# Findings & Decisions

## Requirements

- 第一阶段只支持 `sub2api` 类型。
- Sub2API 登录使用 `email + password`。
- 登录 token 必须 Redis 缓存，按 `expires_in` 设置 TTL。
- token 失效后清理缓存、重新登录并重试一次。
- 提供同步全部及账号、分组、余额、成本独立同步。
- 业务数据写入本地数据库，页面查询不直接调用上游。
- 同步记录保留 30 天，详细快照保留 30 天，每日统计保留 365 天，失效账号和分组保留 90 天。
- 新增同步任务和清理任务，统一由供应商自动化任务中心维护。
- 中文硬编码，不引入 i18n。
- 不修改旧上游管理业务代码。

## Research Findings

- 2026-07-16 会话恢复检查确认：本轮同步功能尚未写入生产代码，当前工作区改动仍是此前供应商 CRUD、菜单页面、本地开发脚本及其他已有未提交内容。
- 仓库仅有根目录 `AGENTS.md`，本次涉及的后端、前端文件没有更深层级规则覆盖。
- 当前供应商路由已经挂载供应商 CRUD、类型 CRUD；同步、账号、分组、自动化任务仍需新增独立 Handler 和路由。
- `wire_gen.go` 已包含供应商 CRUD 的手写生成结果，本次依赖注入需同步维护 Wire provider set 与生成文件，避免本地未安装 Wire 时无法编译。
- `SupplierProviderRepository.GetByID` 返回完整加密凭据，适合新同步服务内部取数；面向页面的 `SupplierProviderService.Get/List` 会调用 `redactSupplierProvider`。
- 现有 `supplier_provider_repo.go` 同时承担供应商与供应商类型 CRUD；为避免继续膨胀，本次新建 `supplier_provider_data_repo.go` 和 `supplier_automation_repo.go`，不把同步 SQL 塞回原 CRUD 仓储。
- `177_supplier_management.sql` 已有运行状态、指标快照和同步记录；新迁移需要给同步记录增加 `sync_scope`，并新增账号、分组、每日统计、任务配置、任务运行表。
- 供应商删除是软删除且会物理删除凭据；修改或删除后 token 清理由服务层通过 token cache 显式执行，不能依赖数据库级联。
- 当前类型模板会把 `AvailableGroupsURL` 强制设为 `GroupsURL`，符合用户要求“分组接口与可用分组接口为同一个接口”。
- 项目已依赖 `github.com/redis/go-redis/v9`、`github.com/robfig/cron/v3` 和 `miniredis`，无需新增第三方依赖。
- `leader_lock_cache.go` 已提供 Redis `SET NX` 与 owner compare-and-delete Lua 脚本模式；供应商 token 登录锁和任务锁应采用同样的安全释放方式，但使用独立键前缀。
- 后台服务通常在 `Provide...` 中构造并 `Start()`，在 `cmd/server/wire.go` 的 `provideCleanup` 中 `Stop()`；供应商自动化调度器应遵循同样生命周期。
- `handler.AdminHandlers` 当前仅暴露 `SupplierProvider` 与 `SupplierProviderType`，本次需新增同步数据和自动化 Handler 字段，并同步更新 `ProvideAdminHandlers` 参数与返回结构。
- 旧 `Sub2APIProviderAdapter` 可作为协议解析参考，但新模块不能调用它；其登录请求明确使用 `email/password`，账号响应为 `data.items`，分组响应为 `data` 数组，余额为 `data.balance`，成本为 `data.today_actual_cost`。
- 新客户端需要在旧解析形态之外兼容常见 `success + data.items` 包装，且所有响应读取必须增加 4 MiB 上限，不能照搬旧适配器的无限 `io.ReadAll`。
- `OpsCleanupService` 展示了可重载的 5 字段 Cron、时区、Redis owner 锁和启停方式；供应商任务配置来自独立表，因此调度器可按任务列表重建一个 Cron 实例。
- `SupplierAutomationView.vue`、`SupplierAccountsView.vue`、`SupplierGroupsView.vue` 均以模块级静态数组驱动；本次应整页重写为 API 请求、加载态、空态、错误态和筛选状态。
- `SupplierProvidersView.vue` 已接真实 CRUD 和主体框架样式，只需新增同步按钮、逐项 loading 和同步后刷新，不应整体推翻。
- 共享 `supplier-management.css` 已包含桌面、平板和手机断点，本次应复用现有类并仅补充任务配置、分页和移动端操作所需样式。
- 当前最高迁移编号为 `178`，本次使用 `backend/migrations/179_supplier_provider_sync.sql`。
- 后端已依赖 `go-sqlmock` 与 `miniredis`，Redis 缓存和 SQL 仓储均可做快速单元测试，不需要测试容器。
- 当前分支为 `dev-2.0`，不是 `main/master`；由于供应商基础模块存在大量未跟踪/未提交上下文，本次继续在当前工作区实现，禁止创建会丢失这些上下文的新 worktree。
- `SupplierProvidersView.vue` 的 CRUD 逻辑集中且已有统一 `errorMessage/showToast/loadProviders`，同步按钮应复用这些状态管理约定。
- 旧模块的 URL 组合、HTTP 错误和业务认证失败判定分别位于 `upstream_provider_adapter.go` 与 `upstream_model_square.go`；新客户端可参考判断逻辑但必须保持独立类型和方法。
- 本项目登录响应的 `expires_in` 为秒，供应商客户端解析后应转换为 `time.Duration(expiresIn) * time.Second` 再计算 Redis TTL。
- `SecretEncryptor` 同时提供 `Encrypt`/`Decrypt`，同步服务可直接解密 `PasswordEncrypted`；现有供应商测试 stub 已实现 `Decrypt`。
- 同步冲突应使用项目标准 `infraerrors.Conflict`，Handler 通过 `response.ErrorFrom` 自动输出 409 和 reason。
- 分页接口可直接返回领域层 `{items,total,page,page_size}` 结果，或使用 `response.Paginated`；供应商现有 API 已采用前一种统一结构。
- 分批清理可复用项目已有 `WITH target/doomed ... LIMIT $n` 后 `DELETE ... USING/IN` 模式，并以 `RowsAffected` 判断是否继续下一批。
- 账号/分组 upsert 应在显式事务中执行，网络请求保持在事务外；列表 SQL 延续动态参数位置与 `LIMIT/OFFSET` 约定。
- `backend/internal/handler/auth_handler.go` 的 Sub2API 登录响应包含 `access_token`、`refresh_token`、`expires_in` 和 `token_type`。
- 旧 `Sub2APIProviderAdapter` 只使用进程内 map 缓存 token，支持 401 后重登，但不支持重启、多实例和 TTL。
- 新供应商数据已使用独立表：`supplier_providers`、`supplier_provider_credentials`、`supplier_provider_runtime_stats`、`supplier_provider_metric_snapshots`、`supplier_provider_sync_runs`。
- 当前供应商自动化页面仍为静态演示数据，并且存在中文乱码。
- 当前供应商账号页和分组页也完全使用静态演示数据，并且存在中文乱码。
- 当前供应商后台路由只包含 provider-types 和 providers CRUD，没有同步、账号、分组或自动化任务接口。
- 当前 `SupplierProviderService.Get/List` 会抹除 `PasswordEncrypted`，同步服务必须通过 Repository 获取内部凭据并自行解密，不能调用面向页面的脱敏方法。
- `backend/internal/service/wire.go`、`backend/internal/repository/wire.go`、`backend/internal/handler/wire.go` 和 `backend/cmd/server/wire.go` 是新增服务、缓存、Handler 和调度器生命周期的接入点。
- 项目已有 Redis leader lock 和 5 字段 Cron 调度模式，可以复用通用基础设施。
- 当前本地后端修改 Go 代码后不会热更新，需要执行 `dev.bat restart-backend`。

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| 新增 SupplierProviderTokenCache 接口 | 服务层不直接依赖 Redis 客户端，便于测试 |
| 新增 SupplierSub2APIClient | 将协议解析与同步事务分离 |
| 同步服务按类别拆分方法 | 支持独立按钮和全量编排 |
| 上游请求在事务外完成 | 避免网络请求占用数据库长事务 |
| 同步缺失数据标记 inactive | 保留审计能力，定期物理清理 |
| 自动任务配置存表 | 页面可维护启停、Cron、超时和保留周期 |
| Redis 分布式锁控制登录和任务 | 避免多实例重复登录和重复执行 |
| 所有同步范围共享供应商级锁 | 防止分项、全量和定时同步并发导致误失效或统计覆盖 |
| 登录锁等待期间持续抢锁，不绕过有效锁 | 保证并发首次请求最多一个登录者 |
| 凭据变更前必须成功清理 token | Redis 删除失败时不写入新配置，避免旧 token 命中新凭据 |
| 成本日期统一使用 Asia/Shanghai 当日 | 与用户实际接口时区参数及每日统计一致 |
| 保留 HTTP/HTTPS 兼容，不新增私网阻断 | 用户批准设计明确允许 HTTP/HTTPS，私有部署可能使用内网上游；通过限制响应、重定向和日志降低风险 |
| 金额暂用现有 float64 服务风格 | 本阶段保持现有模块一致，数据库仍使用 NUMERIC；不额外引入 decimal 重构 |

## Issues Encountered

| Issue | Resolution |
|-------|------------|
| 用户曾在对话中发送明文供应商密码 | 不写入任何文件或日志，并提醒用户更换 |
| 供应商自动化页面文本乱码 | 实现阶段完全重写该页面 |
| `git diff --stat` 不显示未跟踪供应商文件 | 同时以 `git status --short` 为准，禁止遗漏或覆盖未跟踪文件 |
| 计划审查发现同步互斥、登录锁等待和清理循环缺口 | 已补充供应商级 Redis 锁、严格等待、批量循环和取消检查 |

## Resources

- `docs/superpowers/specs/2026-07-16-supplier-sub2api-sync-design.md`
- `backend/internal/service/supplier_provider_service.go`
- `backend/internal/repository/supplier_provider_repo.go`
- `backend/internal/service/upstream_provider_sub2api.go`
- `backend/internal/repository/leader_lock_cache.go`
- `frontend/src/views/admin/supplier-management/SupplierAutomationView.vue`
- `frontend/src/views/admin/supplier-management/SupplierAccountsView.vue`
- `frontend/src/views/admin/supplier-management/SupplierGroupsView.vue`
- `backend/internal/server/routes/supplier_management.go`
- `backend/internal/handler/admin/supplier_provider_handler.go`

## Visual/Browser Findings

- 尚未进行浏览器验证。
