# 上游管理设计

日期：2026-06-10

## 背景

参考项目 `D:\codingspace\workspace-zhongyj\sub2api` 已经实现了一套上游管理能力，核心入口是后台菜单“上游分组”。当前项目 `sub2api-2.0` 还没有 `model-square` 或 `upstream-management` 模块；现有后台路由、前端路由和菜单都未包含这套功能。

参考项目里的上游管理不是单纯列表页，而是把几个运维动作放在同一页：

- 查看上游可用分组，并和本地分组按名称匹配。
- 将未匹配的上游分组同步成本地分组。
- 查看上游秘钥摘要，识别上游分组是否有可用秘钥。
- 同步或检查上游倍率，提示本地倍率低于上游倍率的风险。
- 运行账号倍率守护，发现本地账号绑定了低倍率分组时解除绑定。
- 展示近 90 分钟监控趋势，用于判断分组近期可用性。

当前 2.0 项目的后端路由采用模块化注册方式，前端后台页面也已按功能模块组织。适合新增一个独立的“上游管理”模块，而不是把能力继续混进现有分组管理页。

## 目标

第一版目标是把“上游来源和本地资源的对账能力”引入 2.0，并为后续扩展成完整上游运营台留出边界。

具体目标：

- 管理员能看到上游分组和本地分组的匹配状态。
- 管理员能从上游分组创建本地分组，减少手工录入。
- 管理员能看到上游秘钥和倍率风险，避免本地分组倍率低于上游成本。
- 管理员能手动执行倍率守护的模拟检查；正式自动执行作为后续阶段。
- 前后端新增功能尽量集中在新文件和新模块，减少对现有分组、账号、设置代码的侵入。

非目标：

- 第一版不重做现有“分组管理”和“账号管理”的核心数据模型。
- 第一版不强制迁移所有旧 `model-square` 命名，兼容别名可作为过渡层存在。
- 第一版不把上游源配置做成复杂的多供应商 CRUD；先支持读取配置和必要的测试入口。
- 第一版不把参考项目的大型页面原样照搬为最终形态。

## 方案比较

### 方案 A：最小迁移参考项目页面

直接迁移参考项目的 `UpstreamGroupsView.vue`、前端 API 封装、后端 `ModelSquareHandler`、路由和测试。

优点是交付最快，参考项目已有行为可以复用。缺点是参考页面职责较多，`ModelSquareHandler` 命名和职责也偏历史包袱；迁入后会给 2.0 留下一个大而混杂的模块，后续继续扩展会更难。

### 方案 B：独立上游管理模块，第一阶段迁移核心能力

新增独立的 `upstream-management` 后台模块。第一阶段只落地“分组对账、同步本地分组、秘钥摘要、倍率风险检查、手动模拟守护”，页面按模块化组件拆开。后端保留清晰的 `UpstreamManagementHandler` 或 `UpstreamManagementService` 边界，内部可以复用参考项目算法。

优点是结构适合 2.0，风险可控，也能满足“尽量不改现有代码和文件”。缺点是比整页照搬多一点整理成本。

推荐采用方案 B。

### 方案 C：先做后端抽象，再做 UI

先抽象上游供应商、认证、分组、秘钥、倍率守护等后端接口，前端暂时只做简单调试页。

优点是长期架构最稳，适合未来接入很多上游平台。缺点是第一版可见价值较慢，而且在没有实际 UI 使用反馈前，抽象容易过度。

## 推荐设计

采用方案 B：独立上游管理模块，第一阶段只迁移核心对账能力。

后台菜单新增“上游管理”，首个页面为“上游分组”。后续如果需要扩展，可以在同一模块下增加“上游源”“倍率守护”“秘钥摘要”等子页。第一版仍可以把这些区域放在一个页面内，但代码上应拆成小组件，避免形成参考项目那样的单文件大页面。

建议路由：

- `/admin/upstream-management/groups`：上游分组对账页。
- `/admin/model-square/groups`：兼容重定向到上游分组页，仅用于历史链接。
- 可选 `/model-square`：如果后续需要用户侧模型广场，再单独设计；第一版不纳入。

建议菜单：

- 管理员后台菜单新增“上游管理”或“上游分组”。
- 如果第一版只有一个页面，菜单可直接指向 `/admin/upstream-management/groups`。
- 如果确定后续会继续扩展，菜单使用可展开分组，第一项为“上游分组”。

## 页面结构

“上游分组”页分为五个区域。

### 1. 顶部操作区

包含标题、刷新按钮、倍率风险检查按钮、说明性状态。这里显示当前上游配置是否完整，例如 Base URL、Groups URL、Keys URL 是否可用。配置不完整时，页面不崩溃，展示明确错误并引导到系统设置。

### 2. 筛选与摘要

筛选项：

- 搜索：匹配上游分组名、本地分组名、描述。
- 平台：OpenAI、Claude、Gemini、Antigravity、其他。
- 匹配状态：全部、已匹配、未匹配。
- 风险状态：全部、有秘钥无本地分组、本地倍率低于上游倍率、无可用秘钥。

摘要卡片：

- 上游分组总数。
- 已匹配本地分组数。
- 未匹配数。
- 有倍率风险数。

### 3. 分组对账表

核心列：

- 操作：同步本地分组、查看秘钥。
- 上游 ID。
- 上游分组名称。
- 平台。
- 上游倍率。
- 上游状态。
- 上游秘钥状态。
- 本地分组名称。
- 本地倍率。
- 近 90 分钟趋势。
- 描述。
- 更新时间。

同步本地分组时打开确认弹窗，展示将要创建的字段：名称、平台、倍率、状态、描述。创建成功后刷新对账数据。

### 4. 秘钥摘要

从上游秘钥接口聚合分组下的 key 信息。第一版只展示脱敏名称或数量，不展示完整秘钥。入口可以是表格里的“上游秘钥”状态按钮。

状态建议：

- 有可用秘钥。
- 无秘钥。
- 拉取失败。
- 未配置 Keys URL。

### 5. 倍率守护

第一版只提供“模拟检查”。它检查上游 key 对应的分组倍率和本地账号绑定分组倍率：

- 如果本地账号绑定的最低分组倍率低于上游分组倍率，标记为风险项。
- 模拟检查只返回影响范围，不写数据库。
- 正式解绑、自动定时运行、审计保留策略进入第二阶段。

页面展示最近一次模拟结果：检查 key 数、命中账号数、风险项数、建议解绑的分组。

## 后端设计

新增后台路由注册函数：

- `registerUpstreamManagementRoutes(admin, h)`

建议接口：

- `GET /admin/upstream-management/groups`
- `GET /admin/upstream-management/key-summary`
- `GET /admin/upstream-management/rate-warnings`
- `POST /admin/upstream-management/sync`
- `POST /admin/upstream-management/account-rate-guard/run`
- `GET /admin/upstream-management/account-rate-guard/status`
- `GET /admin/upstream-management/monitor-status`
- `POST /admin/upstream-management/providers/test`

兼容接口：

- `/admin/model-square/groups`
- `/admin/model-square/key-summary`
- `/admin/model-square/rate-warnings`
- `/admin/model-square/sync`

后端边界建议：

- `UpstreamManagementHandler`：只负责 HTTP 参数、响应和错误码。
- `UpstreamManagementService`：负责上游请求、认证、分组合并、秘钥摘要、倍率风险计算。
- `UpstreamProviderClient`：封装不同上游类型的登录、请求、解析。
- `AccountRateGuardService`：独立处理倍率守护逻辑，第一版只实现 dry-run。

参考项目的 `ModelSquareHandler` 可以作为算法来源，但不建议整体复制成最终命名。可以先在实现阶段保留少量历史别名，API 对外使用 `upstream-management` 命名。

## 数据流

分组对账：

1. 前端请求 `GET /admin/upstream-management/groups`。
2. 后端读取上游配置。
3. 后端请求上游 groups endpoint。
4. 后端读取本地分组。
5. 按规范化名称匹配上游分组和本地分组。
6. 返回合并后的对账列表。

同步本地分组：

1. 前端选择未匹配上游分组并确认。
2. 后端根据上游分组创建本地分组。
3. 创建时复用现有分组校验规则。
4. 返回创建结果并刷新列表。

秘钥摘要：

1. 后端请求上游 keys endpoint。
2. 按上游分组聚合 key。
3. 返回分组维度摘要，前端仅展示数量和脱敏名称。

倍率守护 dry-run：

1. 后端请求上游 keys endpoint。
2. 后端读取本地账号和分组绑定。
3. 按账号名或配置的前缀策略匹配上游 key。
4. 比较本地绑定分组最低倍率和上游分组倍率。
5. 返回风险项，不写数据库。

## 错误处理

- 上游配置缺失：返回明确的配置错误，前端展示“去系统设置配置”入口。
- 上游登录失败：返回认证失败，不暴露密码或 token。
- 上游接口超时：返回超时错误，前端保留已加载数据并提示刷新失败。
- 上游返回结构不兼容：返回解析错误和来源阶段，例如 groups、keys、login。
- 本地创建分组冲突：返回已有分组信息，前端提示刷新或跳转本地分组。
- dry-run 风险检查失败：区分配置错误、上游错误、本地数据错误。

所有后端日志必须脱敏，不记录完整 API key、密码、token、cookie。

## 前端文件边界

建议新增：

- `frontend/src/views/admin/upstream-management/UpstreamGroupsView.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamGroupsTable.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamGroupSyncDialog.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamKeySummaryDialog.vue`
- `frontend/src/views/admin/upstream-management/components/AccountRateGuardPanel.vue`
- `frontend/src/api/admin/upstreamManagement.ts`

需要少量修改：

- `frontend/src/router/index.ts`：新增后台路由和兼容重定向。
- `frontend/src/components/layout/AppSidebar.vue`：新增后台菜单项。
- `frontend/src/api/admin/index.ts`：导出 `upstreamManagementAPI`。
- `frontend/src/i18n/locales/zh.ts` 和 `en.ts`：新增菜单和页面文案。

不建议把上游管理 API 继续塞进 `frontend/src/api/admin/groups.ts`，避免污染现有本地分组 API。

## 后端文件边界

建议新增：

- `backend/internal/handler/admin/upstream_management_handler.go`
- `backend/internal/service/upstream_management.go`
- `backend/internal/service/upstream_provider_client.go`
- `backend/internal/service/account_rate_guard.go`
- `backend/internal/server/routes/upstream_management.go`

需要少量修改：

- `backend/internal/handler/handler.go`：AdminHandlers 增加上游管理 handler。
- `backend/internal/handler/wire.go`：注入上游管理 handler。
- `backend/internal/server/routes/admin.go`：注册上游管理路由。
- `backend/internal/config/config.go` 和设置服务：如果当前没有上游管理配置，需要新增配置读取和 DB 设置映射。

如果为了快速落地需要复用参考项目的 `ModelSquareHandler`，也应在第一轮实现后安排一次命名和职责收敛，避免长期保留历史模块名。

## 测试策略

后端测试：

- 上游 groups 正常返回时，能合并本地分组匹配状态。
- 上游 groups 返回空列表时，返回空数组。
- 上游认证失败时，返回明确错误。
- 同步本地分组时，复用本地分组校验。
- key-summary 能正确按分组聚合。
- rate-warnings 能识别本地倍率低于上游倍率。
- account-rate-guard dry-run 不写数据库。

前端测试：

- API 封装请求路径正确。
- 路由 `/admin/upstream-management/groups` 可访问且需要管理员权限。
- 兼容路由 `/admin/model-square/groups` 重定向正确。
- 页面能展示加载、空状态、错误状态、已匹配和未匹配状态。
- 同步弹窗提交成功后刷新列表。
- dry-run 面板能展示风险项。

验证命令建议：

- `go test ./backend/internal/handler/admin ./backend/internal/service ./backend/internal/server/routes`
- `pnpm.cmd test:run`
- `pnpm.cmd run build`

## 分阶段计划

### 阶段 1：上游分组对账

新增后端 groups 接口、前端 API、路由、菜单和上游分组页面。只读上游分组并展示本地匹配状态。

### 阶段 2：同步本地分组与秘钥摘要

增加从未匹配上游分组创建本地分组的流程。增加 key-summary 接口和秘钥摘要弹窗。

### 阶段 3：倍率风险和手动 dry-run

增加倍率风险接口和账号倍率守护 dry-run。前端展示风险项和建议动作，但不自动写数据库。

### 阶段 4：上游源配置和正式守护

把上游源配置从系统设置中独立出来，支持测试连接。根据实际使用情况，再决定是否启用正式解绑、定时执行和审计持久化。

## 风险与取舍

- 参考项目逻辑集中在大 handler 和大页面中，直接迁移速度快但维护风险高。
- 当前 2.0 项目已有复杂后台模块，新增独立文件边界可以降低回归风险。
- 第一版 dry-run 优先，避免自动解绑账号分组造成不可逆运营影响。
- 需要确认上游接口返回结构是否稳定；如果不稳定，后端解析层必须更宽容并返回阶段化错误。
- 如果后续要支持多个上游源，第一版的接口返回结构应预留 `provider_slug`、`provider_name` 字段。

## 验收标准

- 管理员能从后台进入“上游分组”页面。
- 页面能展示上游分组和本地分组匹配状态。
- 未配置上游时页面能展示明确错误，而不是空白或崩溃。
- 未匹配上游分组能创建为本地分组。
- 上游秘钥摘要不泄露完整秘钥。
- 倍率守护 dry-run 不写数据库，只返回风险结果。
- 新增代码集中在上游管理模块，现有分组/账号/设置代码只做必要接入修改。
