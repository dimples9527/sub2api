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

第一版目标是先把“上游配置管理”引入 2.0。管理员可以维护不同类型的上游来源，保存登录所需信息，并通过统一接口测试登录、获取密钥和归一化密钥数据。分组对账、倍率风险和账号倍率守护建立在这套配置与 provider adapter 之上，作为后续阶段扩展。

具体目标：

- 管理员能在独立“上游管理”模块中维护上游配置。
- 第一批支持 `sub2api` 和 `newapi` 两种类型，后续可以新增类型。
- 每个上游配置维护其登录和密钥获取所需信息。
- 后端提供统一登录测试接口和统一密钥获取接口。
- `sub2api` 获取密钥只调用一个密钥接口。
- `newapi` 获取密钥需要调用密钥接口和分组接口，并在后端合并为统一密钥视图。
- 前后端新增功能尽量集中在新文件和新模块；现有文件只做路由注册、菜单入口、依赖注入、API 导出这类必要接线。

非目标：

- 第一版不重做现有“分组管理”和“账号管理”的核心数据模型。
- 第一版不强制迁移所有旧 `model-square` 命名，兼容别名可作为过渡层存在。
- 第一版不做分组对账、同步本地分组、倍率守护的完整页面；这些依赖上游配置管理稳定后再推进。
- 第一版不把参考项目的大型页面原样照搬为最终形态。
- 第一版不改造现有系统设置页来承载上游配置；上游配置应放在新模块中。

## 方案比较

### 方案 A：最小迁移参考项目页面

直接迁移参考项目的 `UpstreamGroupsView.vue`、前端 API 封装、后端 `ModelSquareHandler`、路由和测试。

优点是交付最快，参考项目已有行为可以复用。缺点是参考页面职责较多，`ModelSquareHandler` 命名和职责也偏历史包袱；迁入后会给 2.0 留下一个大而混杂的模块，后续继续扩展会更难。

### 方案 B：独立上游管理模块，第一阶段先做配置管理

新增独立的 `upstream-management` 后台模块。第一阶段只落地“上游配置管理、登录测试、密钥获取测试、统一密钥视图”。后端使用 provider adapter 抽象不同上游类型，先实现 `sub2api` 和 `newapi` 两个 adapter。分组对账和倍率守护作为第二阶段以后使用这些 adapter。

优点是结构适合 2.0，风险可控，也最符合“尽量新写代码，不动现有代码和文件”。缺点是分组对账页面不会第一时间完整可用，需要分阶段交付。

推荐采用方案 B。

### 方案 C：先做后端抽象，再做 UI

先抽象上游供应商、认证、分组、秘钥、倍率守护等后端接口，前端暂时只做简单调试页。

优点是长期架构最稳，适合未来接入很多上游平台。缺点是第一版可见价值较慢，而且在没有实际 UI 使用反馈前，抽象容易过度。

## 推荐设计

采用方案 B：独立上游管理模块，第一阶段先做上游配置管理。

后台菜单新增“上游管理”，首个页面为“上游配置”。它维护多个上游 provider，每个 provider 有类型、基础地址、登录信息、密钥接口、分组接口等字段。后续同一模块下再增加“上游分组”“密钥摘要”“倍率守护”等页面。

建议路由：

- `/admin/upstream-management/providers`：上游配置管理页。
- `/admin/upstream-management/groups`：上游分组对账页，第二阶段启用。
- `/admin/model-square/groups`：后续兼容重定向到上游分组页，仅用于历史链接。
- 可选 `/model-square`：如果后续需要用户侧模型广场，再单独设计；第一版不纳入。

建议菜单：

- 管理员后台菜单新增“上游管理”。
- 第一阶段菜单项指向 `/admin/upstream-management/providers`。
- 如果使用展开分组，第一项为“上游配置”，第二阶段再加入“上游分组”。

## 低侵入约束

实现时优先新增代码，现有文件只允许做必要接入：

- 路由注册文件只增加一行或一个函数调用。
- 依赖注入文件只增加新 handler/service 的接线。
- 侧边栏只增加一个菜单入口。
- 前端 API barrel 只导出新模块。
- 不把新业务逻辑塞进现有 `groups.ts`、`SettingsView.vue`、`GroupHandler` 或 `GroupService`。
- 如果必须复用现有能力，通过接口调用或小型 adapter 连接，不直接改写原有实现。

## 页面结构

第一阶段页面是“上游配置”页，分为四个区域。

### 1. 顶部操作区

包含标题、新增上游按钮、刷新按钮和简要状态。状态显示已启用 provider 数量、最近测试结果和配置缺失数量。

### 2. 上游配置列表

筛选项：

- 搜索：匹配名称、slug、Base URL。
- 类型：全部、Sub2API、NewAPI。
- 状态：全部、启用、停用、配置不完整。

列表列：

- 名称。
- 类型。
- Slug。
- Base URL。
- 登录方式摘要。
- 密钥接口状态。
- 最近测试结果。
- 启用状态。
- 操作。

### 3. 配置编辑抽屉或弹窗

通用字段：

- 类型：`sub2api`、`newapi`。
- Slug：稳定标识，保存后不建议频繁修改。
- 名称。
- 启用状态。
- Base URL。
- 登录 URL。
- 密钥 URL。
- 分组 URL。
- 邮箱。
- 用户名。
- 密码。
- 账号名称前缀。
- 临时禁用分钟数。

类型差异：

- `sub2api`：密钥获取只需要密钥 URL；登录可选。如果配置了邮箱或密码，则先登录并携带 token；未配置登录信息时直接请求密钥 URL。
- `newapi`：登录必需；密钥获取需要密钥 URL 和分组 URL 两个接口。后端用密钥接口返回的 key 所属分组名，去分组接口返回的倍率表里匹配倍率，再合并成统一密钥项。

密码处理：

- 前端不回显密码明文。
- 编辑时留空表示保留原密码。
- 提供 `password_configured` 状态。
- 测试结果和日志不得输出密码、token、cookie、完整 key。

### 4. 测试面板

每个 provider 提供“测试连接”操作。测试结果按阶段展示：

- 登录阶段。
- 密钥接口阶段。
- 分组接口阶段，仅 `newapi` 必需。
- 解析阶段。

测试成功后显示归一化后的样例 key：

- key 名称。
- 上游分组名称。
- 上游倍率。
- 来源 provider。

样例只展示前若干条，并对敏感字段脱敏。

## Provider Adapter 设计

后端定义统一 provider adapter 接口，业务层只依赖统一结果，不直接关心具体上游类型。

建议能力：

- `ValidateConfig(provider)`：校验该类型必填字段。
- `Login(ctx, provider)`：返回会话信息，可为空。
- `FetchKeys(ctx, provider)`：返回统一密钥项。
- `Test(ctx, provider)`：返回分阶段测试结果。

统一密钥项字段：

- `provider_slug`
- `provider_name`
- `provider_type`
- `key_name`
- `group_name`
- `rate_multiplier`
- `raw_status`
- `raw_group_id`

`sub2api` adapter：

1. 如果配置了邮箱或密码，调用登录接口。
2. 调用密钥接口。
3. 解析返回数据中的 key、group、rate multiplier。
4. 输出统一密钥项。

`newapi` adapter：

1. 使用 username 或 email 加 password 调用登录接口。
2. 从登录响应和 Set-Cookie 中建立会话。
3. 调用密钥接口，获取 key 列表及其 group 名称。
4. 调用分组接口，获取 group 名称到 ratio 的映射。
5. 按原始 group 名称和规范化 group 名称双重匹配。
6. 只有匹配到倍率的 key 才输出统一密钥项；未匹配项进入 warnings。

后续新增 provider 类型时，只新增 adapter 和类型注册，不改调用方流程。

## 后端设计

新增后台路由注册函数：

- `registerUpstreamManagementRoutes(admin, h)`

建议接口：

- `GET /admin/upstream-management/providers`
- `POST /admin/upstream-management/providers`
- `PUT /admin/upstream-management/providers/:slug`
- `DELETE /admin/upstream-management/providers/:slug`
- `POST /admin/upstream-management/providers/:slug/test`
- `GET /admin/upstream-management/providers/:slug/keys`
- `GET /admin/upstream-management/groups`
- `GET /admin/upstream-management/key-summary`
- `GET /admin/upstream-management/rate-warnings`
- `POST /admin/upstream-management/sync`
- `POST /admin/upstream-management/account-rate-guard/run`
- `GET /admin/upstream-management/account-rate-guard/status`
- `GET /admin/upstream-management/monitor-status`
- `POST /admin/upstream-management/providers/test`，兼容临时测试未保存表单数据，可选。

兼容接口：

- `/admin/model-square/groups`
- `/admin/model-square/key-summary`
- `/admin/model-square/rate-warnings`
- `/admin/model-square/sync`

后端边界建议：

- `UpstreamProviderHandler`：只负责 provider 配置 CRUD、测试、密钥查看的 HTTP 层。
- `UpstreamProviderService`：负责配置存取、密码保留、启停状态、调用 adapter。
- `UpstreamProviderAdapterRegistry`：按 provider type 分发到具体 adapter。
- `Sub2APIProviderAdapter`：封装 Sub2API 登录和单接口密钥解析。
- `NewAPIProviderAdapter`：封装 NewAPI 登录、密钥接口、分组接口和合并逻辑。
- `UpstreamManagementService`：第二阶段再负责分组合并、秘钥摘要、倍率风险计算。
- `AccountRateGuardService`：独立处理倍率守护逻辑；进入倍率守护阶段时先只实现 dry-run。

参考项目的 `ModelSquareHandler` 可以作为算法来源，但不建议整体复制成最终命名。可以先在实现阶段保留少量历史别名，API 对外使用 `upstream-management` 命名。

## 数据流

配置管理：

1. 前端请求 provider 列表。
2. 后端从上游管理配置存储读取 provider 数组。
3. 后端脱敏返回，密码只返回 `password_configured`。
4. 前端新增或编辑 provider。
5. 后端按类型校验必填字段，并处理密码留空保留逻辑。
6. 后端保存配置。

测试 provider：

1. 前端点击测试。
2. 后端读取已保存 provider，或接收临时 provider 表单。
3. 后端根据 type 选择 adapter。
4. adapter 执行登录、密钥获取、分组获取、解析。
5. 后端返回分阶段结果、样例密钥和 warnings。

统一密钥获取：

1. 前端或后续业务请求 `GET /admin/upstream-management/providers/:slug/keys`。
2. 后端读取 provider 配置。
3. 后端调用对应 adapter 的 `FetchKeys`。
4. adapter 返回统一密钥项。
5. 后端脱敏后返回。

分组对账，第二阶段：

1. 后端基于 provider adapter 获取统一密钥项和上游分组信息。
2. 后端读取本地账号和分组绑定。
3. 按规范化名称匹配上游分组和本地分组。
4. 返回合并后的对账列表。

## 错误处理

- 上游配置缺失：返回明确的配置错误，前端在配置表单字段上标注。
- 上游登录失败：返回认证失败，不暴露密码或 token。
- 上游接口超时：返回超时错误，前端保留已加载数据并提示刷新失败。
- 上游返回结构不兼容：返回解析错误和来源阶段，例如 groups、keys、login。
- `newapi` 密钥和分组无法合并：返回 warnings，保留能匹配到倍率的 key。
- 本地创建分组冲突：返回已有分组信息，前端提示刷新或跳转本地分组。
- dry-run 风险检查失败：区分配置错误、上游错误、本地数据错误。

所有后端日志必须脱敏，不记录完整 API key、密码、token、cookie。

## 前端文件边界

建议新增：

- `frontend/src/views/admin/upstream-management/UpstreamProvidersView.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamProviderTable.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamProviderEditor.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamProviderTestPanel.vue`
- `frontend/src/views/admin/upstream-management/UpstreamGroupsView.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamGroupsTable.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamGroupSyncDialog.vue`
- `frontend/src/views/admin/upstream-management/components/UpstreamKeySummaryDialog.vue`
- `frontend/src/views/admin/upstream-management/components/AccountRateGuardPanel.vue`
- `frontend/src/api/admin/upstreamProviders.ts`
- `frontend/src/api/admin/upstreamManagement.ts`，第二阶段用于分组对账等聚合能力。

需要少量修改：

- `frontend/src/router/index.ts`：新增后台路由和后续兼容重定向。
- `frontend/src/components/layout/AppSidebar.vue`：新增后台菜单项。
- `frontend/src/api/admin/index.ts`：导出 `upstreamProvidersAPI`；第二阶段再导出聚合用的 `upstreamManagementAPI`。
- `frontend/src/i18n/locales/zh.ts` 和 `en.ts`：新增菜单和页面文案。

不建议把上游管理 API 继续塞进 `frontend/src/api/admin/groups.ts`，避免污染现有本地分组 API。

## 后端文件边界

建议新增：

- `backend/internal/handler/admin/upstream_provider_handler.go`
- `backend/internal/service/upstream_provider.go`
- `backend/internal/service/upstream_provider_adapter.go`
- `backend/internal/service/upstream_provider_sub2api.go`
- `backend/internal/service/upstream_provider_newapi.go`
- `backend/internal/service/upstream_management.go`，第二阶段用于聚合对账能力。
- `backend/internal/service/account_rate_guard.go`
- `backend/internal/server/routes/upstream_management.go`

需要少量修改：

- `backend/internal/handler/handler.go`：AdminHandlers 增加上游管理 handler。
- `backend/internal/handler/wire.go`：注入上游管理 handler。
- `backend/internal/server/routes/admin.go`：注册上游管理路由。
- 配置存储接入文件：优先新增上游 provider 独立存储；如果必须接入现有设置服务，只做读取/写入入口，不把 provider 逻辑写入设置服务。

如果为了快速落地需要复用参考项目的 `ModelSquareHandler`，也应在第一轮实现后安排一次命名和职责收敛，避免长期保留历史模块名。

## 测试策略

后端测试：

- provider CRUD 能保存、更新、删除、启停配置。
- 密码留空更新时保留原密码。
- `sub2api` adapter 能登录可选、单接口拉取密钥并解析统一密钥项。
- `newapi` adapter 能登录后分别拉取密钥和分组，并合并倍率。
- `newapi` key 找不到对应 group ratio 时进入 warnings。
- 所有测试结果不包含密码、token、cookie、完整 key。
- 未知 provider type 返回明确错误。

前端测试：

- API 封装请求路径正确。
- 路由 `/admin/upstream-management/providers` 可访问且需要管理员权限。
- 上游配置页能展示加载、空状态、错误状态、启用和停用状态。
- 编辑表单按 provider type 切换必填字段。
- 测试面板能展示登录、密钥、分组、解析阶段。
- 密码字段不回显明文，保存后清空输入。

验证命令建议：

- `go test ./backend/internal/handler/admin ./backend/internal/service ./backend/internal/server/routes`
- `pnpm.cmd test:run`
- `pnpm.cmd run build`

## 分阶段计划

### 阶段 1：上游配置管理

新增独立上游管理菜单和“上游配置”页。支持 provider CRUD、启停、密码保留、连接测试、统一密钥获取。实现 `sub2api` 和 `newapi` adapter。

### 阶段 2：上游分组对账与秘钥摘要

基于 provider adapter 的统一密钥项和分组数据，增加上游分组对账页和 key-summary。先只读展示匹配状态。

### 阶段 3：同步本地分组、倍率风险和手动 dry-run

增加从未匹配上游分组创建本地分组的流程。增加倍率风险接口和账号倍率守护 dry-run。前端展示风险项和建议动作，但不自动写数据库。

### 阶段 4：正式守护与更多上游类型

根据实际使用情况，再决定是否启用正式解绑、定时执行和审计持久化。后续新增 provider 类型时，只增加 adapter。

## 风险与取舍

- 参考项目逻辑集中在大 handler 和大页面中，直接迁移速度快但维护风险高。
- 当前 2.0 项目已有复杂后台模块，新增独立文件边界可以降低回归风险。
- 现有文件完全不改不现实；路由、菜单、依赖注入仍需要少量接线。设计目标是把接线压到最小。
- 倍率守护阶段 dry-run 优先，避免自动解绑账号分组造成不可逆运营影响。
- 需要确认上游接口返回结构是否稳定；如果不稳定，后端解析层必须更宽容并返回阶段化错误。
- 如果后续要支持多个上游源，第一版的接口返回结构应预留 `provider_slug`、`provider_name` 字段。

## 验收标准

- 管理员能从后台进入“上游配置”页面。
- 页面能新增、编辑、启停、删除 `sub2api` 和 `newapi` provider。
- 密码不回显；留空保存时保留已配置密码。
- `sub2api` provider 能通过一个密钥接口获取并解析统一密钥项。
- `newapi` provider 能登录后通过密钥接口和分组接口合并出统一密钥项。
- 测试连接能展示登录、密钥、分组、解析阶段结果。
- 测试结果和日志不泄露密码、token、cookie、完整 key。
- 新增代码集中在上游管理模块，现有文件只做必要接线修改。
