# 上游账号同步设计

日期：2026-06-10

## 背景

当前项目已经完成了独立的上游配置管理和上游分组管理：

- 上游配置支持 `sub2api` 和 `newapi` 两类 provider。
- 上游 provider 已经支持默认上游、账号名称前缀、登录测试、密钥获取和倍率解析。
- 上游分组页面只基于默认上游做本地分组匹配。
- 分组匹配支持手动映射优先、规范化名称兜底。
- 本地分组倍率低于上游倍率时，系统可以把本地倍率提高到上游倍率，并记录修改历史。

参考项目 `D:\codingspace\workspace-zhongyj\sub2api` 的同步能力主要包含两类行为：

1. 基于上游 key 的分组倍率修正。
2. 基于上游 key 和本地账号的名称匹配，检查并解绑低倍率本地分组。

第 1 类能力已经迁移到当前项目的上游分组模块。下一步应补齐第 2 类能力，并在此基础上设计“上游 key 到本地账号”的同步入口。这样管理员可以从默认上游拉取 key，预览本地账号差异，再选择执行账号创建、账号分组更新和倍率保护。

## 目标

新增独立的“上游账号同步”能力，第一版只围绕默认上游展开。

具体目标：

- 从默认上游获取统一 key 列表。
- 根据上游配置里的 `account_name_prefix` 生成本地账号名称。
- 按本地账号名称匹配已有账号。
- 按上游分组映射规则匹配本地分组。
- 提供同步预览，展示将创建、将更新、已匹配、跳过和风险项。
- 提供手动执行同步，创建缺失账号并更新同步托管字段。
- 提供账号倍率保护，发现本地账号绑定了低于上游倍率的分组时，预览并可执行解绑。
- 记录最近同步历史和倍率保护修改历史，方便管理员追踪。

非目标：

- 第一版不做后台定时任务。
- 第一版不支持多个上游一起同步账号。
- 第一版不删除本地账号，也不自动禁用上游已删除的 key 对应账号。
- 第一版不改现有账号表结构，优先使用 `extra` 存储上游来源元数据。
- 第一版不覆盖用户手工维护的账号字段，例如并发、优先级、代理、模型映射、限额、临时禁用策略。

## 推荐方案

采用独立服务和独立页面：

- 后端新增 `UpstreamAccountSyncService`。
- 后端新增 `UpstreamAccountSyncHandler`。
- 前端新增管理员菜单“上游账号”。
- 页面路径建议为 `/admin/upstream-management/accounts`。
- API 统一放在 `/admin/upstream-management/accounts/*` 下。

该方案保持现有上游配置、上游分组、账号管理的边界清晰。上游账号同步依赖已有服务能力，但不把同步逻辑塞进现有账号管理或分组管理页面。

## 账号匹配规则

本地账号名称由默认上游配置和上游 key 生成：

```text
local_account_name = account_name_prefix + upstream_key.key_name
```

匹配时使用两层规则：

1. 精确名称匹配。
2. 规范化名称匹配，规则与当前上游分组匹配保持一致：去除首尾空白、合并连续空白、转小写。

如果同一个规范化名称匹配到多个本地账号，视为冲突。预览阶段展示冲突，不自动更新这些账号。

如果上游 key 名称为空，跳过该 key 并记录跳过原因。

## 分组匹配规则

每个上游 key 带有上游分组名称和上游倍率。同步时复用当前上游分组模块的匹配规则：

1. 手动映射优先。
2. 本地分组规范化名称匹配兜底。

如果上游分组无法匹配本地分组：

- 预览中展示为“未匹配分组”。
- 不创建本地账号。
- 不更新已有本地账号的分组绑定。
- 不执行倍率保护。

该规则避免把上游账号同步到错误分组。

## 创建账号规则

当上游 key 没有匹配到本地账号，且其上游分组已匹配到本地分组时，预览展示为“将创建账号”。

执行同步时创建账号：

- `name`：`account_name_prefix + key_name`
- `platform`：`openai`
- `type`：`apikey`
- `status`：使用账号服务默认值，即 `active`
- `credentials.api_key`：上游 key 原始值
- `credentials.base_url`：默认上游 `base_url`
- `group_ids`：匹配到的本地分组 ID
- `concurrency`：使用系统创建账号默认值
- `priority`：使用系统创建账号默认值
- `extra.upstream_provider_slug`：默认上游 slug
- `extra.upstream_provider_name`：默认上游名称
- `extra.upstream_provider_type`：默认上游类型
- `extra.upstream_key_name`：上游 key 名称
- `extra.upstream_group_name`：上游分组名称
- `extra.upstream_rate_multiplier`：上游倍率
- `extra.upstream_synced_at`：同步执行时间

第一版创建为 OpenAI API Key 账号，是因为当前上游 key 本质是通过上游 OpenAI 兼容接口转发。后续如果要支持 Anthropic、Gemini 或其他平台，需要在 provider 配置中新增目标平台字段，再扩展创建规则。

## 更新账号规则

当上游 key 匹配到唯一的本地账号时，预览展示为“已匹配账号”或“将更新账号”。

第一版只允许更新同步托管字段：

- `credentials.api_key`
- `credentials.base_url`
- `group_ids`
- `extra.upstream_*`

不覆盖以下字段：

- `name`
- `notes`
- `platform`
- `type`
- `proxy_id`
- `concurrency`
- `priority`
- `rate_multiplier`
- `load_factor`
- `status`
- `schedulable`
- 限额相关 `extra`
- RPM 和窗口成本相关 `extra`
- 模型映射相关 `credentials`
- 自定义错误码和临时禁用规则

如果已匹配账号不是 `platform=openai` 且 `type=apikey`，预览展示为风险项，执行时跳过。这样避免把上游 key 写进 OAuth 或其他平台账号。

## 倍率保护规则

倍率保护沿用参考项目的核心逻辑，并落在独立服务中。

对每个已匹配的上游 key 和本地账号：

1. 读取上游 key 的 `rate_multiplier`。
2. 读取本地账号当前绑定的分组及本地倍率。
3. 找出所有本地倍率低于上游倍率的分组。
4. 预览中展示风险项。
5. 执行时从账号分组绑定中移除这些低倍率分组。

如果本地账号没有任何分组，或者没有可识别的分组倍率，不执行解绑，只在预览中提示。

如果账号同时绑定了高倍率分组和低倍率分组，只解绑低倍率分组，保留满足上游倍率要求的分组。

如果解绑后账号没有剩余分组，仍允许保存空分组绑定。此行为与参考项目一致，表示该账号不再暴露给低倍率本地分组使用。

## 预览结果结构

建议返回结构：

```json
{
  "default_provider": {},
  "summary": {
    "upstream_key_count": 0,
    "matched_account_count": 0,
    "create_count": 0,
    "update_count": 0,
    "skip_count": 0,
    "conflict_count": 0,
    "rate_violation_count": 0
  },
  "items": [],
  "warnings": [],
  "records": []
}
```

每个 item 建议包含：

- `action`：`create`、`update`、`noop`、`skip`、`conflict`
- `provider_slug`
- `provider_name`
- `upstream_key_name`
- `local_account_name`
- `matched_account_id`
- `matched_account_name`
- `upstream_group_name`
- `upstream_rate_multiplier`
- `local_group_id`
- `local_group_name`
- `local_rate_multiplier`
- `rate_violation`
- `unbound_group_ids`
- `unbound_group_names`
- `skip_reason`
- `conflict_account_ids`

## API 设计

新增接口：

- `GET /admin/upstream-management/accounts/sync-preview`
- `POST /admin/upstream-management/accounts/sync`
- `GET /admin/upstream-management/accounts/sync-records`

`sync-preview` 只读取上游和本地数据，不写入任何内容。

`sync` 执行手动同步。请求体建议：

```json
{
  "create_missing": true,
  "update_existing": true,
  "apply_rate_guard": true
}
```

默认建议：

- `create_missing = true`
- `update_existing = true`
- `apply_rate_guard = true`

`sync-records` 返回最近 100 条同步记录。记录存储可以先使用 setting key，避免新增表结构。

建议 setting key：

- `upstream_account_sync_records`

## 页面设计

新增管理员菜单“上游账号”，放在“上游配置”和“上游分组”之后。

页面区域：

1. 顶部默认上游摘要，展示 provider 名称、类型、Base URL、账号前缀。
2. 操作区，提供“刷新预览”和“执行同步”。
3. 预览统计，展示 key 总数、将创建、将更新、跳过、冲突、倍率风险。
4. 同步明细表，按 action 分组或筛选。
5. 倍率风险区域，突出展示将解绑的低倍率分组。
6. 同步记录区域，展示最近操作时间、创建数、更新数、解绑数、错误信息。

执行同步前，前端弹出确认框，列出将创建账号数、将更新账号数、将解绑分组数。冲突项和跳过项不会被执行。

## 后端文件边界

建议新增：

- `backend/internal/service/upstream_account_sync.go`
- `backend/internal/service/upstream_account_sync_test.go`
- `backend/internal/handler/admin/upstream_account_sync_handler.go`
- `backend/internal/handler/admin/upstream_account_sync_handler_test.go`

需要少量修改：

- `backend/internal/handler/handler.go`：注册新 handler。
- `backend/internal/handler/wire.go`：注入新 service 和 handler。
- `backend/internal/server/routes/upstream_management.go`：注册账号同步路由。

不建议修改：

- 不把同步逻辑写进 `account_handler.go`。
- 不把同步逻辑写进 `upstream_management.go` 的分组对比核心流程。
- 不修改账号实体结构和数据库 schema。

## 前端文件边界

建议新增：

- `frontend/src/views/admin/upstream-management/UpstreamAccountsView.vue`
- `frontend/src/api/admin/upstreamAccountSync.ts`

需要少量修改：

- `frontend/src/router/index.ts`：增加 `/admin/upstream-management/accounts`。
- `frontend/src/components/layout/AppSidebar.vue`：增加“上游账号”菜单。
- `frontend/src/api/admin/index.ts`：导出新 API。
- `frontend/src/i18n/locales/zh.ts` 和 `frontend/src/i18n/locales/en.ts`：增加菜单和页面文案。

## 错误处理

- 默认上游未配置：返回明确错误，前端引导去“上游配置”设置默认上游。
- 默认上游禁用：允许预览失败并提示先启用上游。
- 上游登录失败：返回登录阶段错误，不泄露密码、token、cookie。
- 上游 key 获取失败：返回 key 阶段错误。
- 上游分组未匹配：跳过对应 key，并展示原因。
- 本地账号名称冲突：跳过对应 key，并展示冲突账号。
- 本地账号平台或类型不兼容：跳过更新，并展示风险原因。
- 本地账号更新失败：停止执行并记录错误，已完成的操作保留在结果中。

所有日志和响应样例都不得包含完整 API key、密码、token 或 cookie。

## 测试策略

后端服务测试：

- 无默认上游时预览返回配置错误。
- 上游 key 能按前缀匹配本地账号。
- 手动分组映射优先于名称匹配。
- 未匹配分组的 key 被跳过。
- 本地账号名称冲突时跳过更新。
- 创建缺失账号时写入 OpenAI API Key 凭据和上游来源 `extra`。
- 更新已有账号时只更新托管字段。
- 非 OpenAI API Key 账号不会被写入上游 key。
- 倍率保护能解绑低倍率分组并保留高倍率分组。
- 同步记录最多保留 100 条。

后端 handler 测试：

- `sync-preview` 不写入账号和 setting。
- `sync` 能按请求参数启用或关闭创建、更新、倍率保护。
- 响应不包含完整 API key。

前端测试：

- 页面能展示默认上游摘要。
- 页面能展示预览统计和明细。
- 执行同步前展示确认信息。
- 冲突项、跳过项、倍率风险项有明确状态。

验证命令建议：

- `go test ./backend/internal/service ./backend/internal/handler/admin`
- `pnpm.cmd test:run`
- `pnpm.cmd run build`

## 实施顺序

1. 新增后端服务测试，先覆盖预览计算规则。
2. 实现 `UpstreamAccountSyncService` 的预览逻辑。
3. 新增执行同步测试。
4. 实现创建账号、更新托管字段、倍率保护和记录写入。
5. 新增 handler 和路由测试。
6. 接入前端 API。
7. 新增“上游账号”页面。
8. 跑后端服务测试、前端测试和构建。

## 待确认决策

以下决策已经在第一版设计中固定：

- 第一版只使用默认上游。
- 第一版创建 OpenAI API Key 账号。
- 第一版不删除本地账号。
- 第一版不自动禁用上游已删除 key 对应的本地账号。
- 第一版优先使用 `extra` 记录上游来源，不新增数据库字段。
