## Why

供应商模块的「供应商账号倍率守护」自动化任务（任务码 `supplier_account_rate_guard`，入口在 `frontend/src/views/admin/supplier-management/SupplierAutomationView.vue`）目前只有**任务级**开关：整个任务要么开、要么关。

当上游账号的倍率高于某些本地分组的分组倍率时，守护会把这些账号从这些分组解绑。但实际运营中存在一类分组，其定价本身就是刻意压低或另有用途的——不应参与自动倍率守护。当前只能整任务关闭，无法表达"整个分组都不参与"，粒度太粗。

需要把开关粒度下沉到**分组**：按分组决定是否参与守护，关闭的分组在执行守护时被跳过，默认全部参与。

> ⚠️ 与「分组倍率守护」不是同一条链路。分组倍率守护（任务码 `supplier_rate_guard`，配置键 `SupplierRateGuardConfig{RateGuardMaxSnapshotAge}`）已经有分组级开关 `supplier_provider_groups.rate_guard_enabled`（迁移 226）；本次要做的是**账号倍率守护**，两者互不影响。

## What Changes

- 在账号倍率守护任务配置中新增**分组级开关**，语义为「该分组是否参与自动倍率守护」。
- 存储落在既有任务配置 `supplier_automation_tasks.config_json` 中，新增 `account_rate_guard_disabled_group_ids: number[]`：**列表为空（含字段缺失）= 全部分组参与**，即默认开启；出现在列表中的分组 = 关闭，守护时跳过。
- 守护执行时跳过关闭分组：这些分组不再进入风险判定、不产生解绑、不计入 `risk_groups` / `unbound_groups`。
- 前端供应商自动化任务编辑弹窗在「供应商账号倍率守护」任务下新增分组开关区，以分组多选维护该列表；文案显式说明"留空表示所有分组都参与守护"。
- 关闭分组的跳过**留有痕迹**：当某个账号的风险分组**全部**被关闭时，写一条 `skipped` 结果日志并在结果中计 `skipped`，原因文案为「N 个风险分组已关闭倍率守护，本次跳过」，便于回溯"为什么这个分组没被处理"。

## Capabilities

### Modified Capabilities
- `account-rate-guard-group-scope`：供应商账号倍率守护的分组适用范围配置（新增能力域，见 spec）。

## Impact

- **数据库**：无迁移。配置沿用 `supplier_automation_tasks.config_json`（自由 JSON），新增字段由 `SupplierAutomationConfig` 承载。
- **后端**：
  - `service.SupplierAutomationConfig` 新增 `AccountRateGuardDisabledGroupIDs []int64`（JSON 键 `account_rate_guard_disabled_group_ids`）。
  - `SupplierAccountRateGuardRunner.Run` 接口方法加参 `disabledGroupIDs []int64`，`executeTask` 的 `SupplierAutomationTaskAccountRateGuard` 分支透传 `task.Config.AccountRateGuardDisabledGroupIDs`。
  - `SupplierAccountRateGuardService.Run` 与 `processCandidate` 同步改签名；`processCandidate` 在收集 `riskGroups` 时排除关闭分组，全部被排除且原本存在风险分组时写一条 `skipped` 日志。
  - 新增 `supplierAccountRateGuardDisabledGroupSet` 辅助函数：nil / 空列表 → 空集合（即全部参与），并丢弃非正数 ID。
- **Wire 依赖注入**：**无需重跑 Wire**。本次只改接口方法签名，`NewSupplierAccountRateGuardService` 的 4 个构造参数不变，`ProvideSupplierAutomationService`（`supplier_account_health_guard_wiring.go`）的依赖图不变。
- **前端**：
  - `api/admin/supplierAutomation.ts` 的 `SupplierAutomationConfig` 新增字段。
  - `SupplierAutomationView.vue` 新增分组开关区、`editForm.config` 默认值、保存校验与源码断言测试。
  - 分组数据源复用 `@/api/admin/groups` 的 `getAll()`（`GET /admin/groups/all`，返回 `id/name/platform/rate_multiplier/status`），与账号绑定的 `group_id` 口径一致。
- **行为变化**：存量配置因列表为空而保持"全部分组参与"，升级后行为不变；只有管理员显式关闭某分组后才会产生差异。
- **不影响**：上游管理模块的 `UpstreamAccountRateGuardPanel.vue` / `upstream_account_rate_guard_config`（同名「账号倍率守护」面板）与分组倍率守护任务。
