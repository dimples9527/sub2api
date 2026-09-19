## Context

需求与动机见 proposal.md。与方案相关的现状（均已核对源码）：

- **配置承载**：任务配置在 `supplier_automation_tasks.config_json`，Go 侧结构体是 `service.SupplierAutomationConfig`（`supplier_automation_service.go:59`）。它同时承载数据清理保留天数、分组倍率守护快照时效、健康守护的全部账号级配置，以及运行态回写的 `AccountHealthGuardCursorAccountID`。
- **配置读写**：`SupplierAutomationService.UpdateTask`（:292）→ `validateSupplierAutomationTask` → `validateSupplierAccountHealthGuardSelection` → `repo.UpdateTask` → `reloader.Reload`。**整体覆盖**：前端 `saveTask` 提交的 config 必须包含全部字段，否则会被静默清空（健康守护用 `applyAccountHealthGuardDefaults()` 兜底，是既有范式）。
- **执行链路**：`RunWithMode`（:349）抢任务锁 → 等上游拉取锁（`supplierAutomationTaskRequiresUpstreamFetchLock` 对 `AccountRateGuard` 返回 true）→ `executeTask`（:514）按 `task.TaskCode` 分派。
- **判定收口**：`SupplierAccountRateGuardService.processCandidate`（`supplier_account_rate_guard.go:217`）。它遍历 `candidate.Groups`，按 `effectiveRate - group.RateMultiplier > 0.0000001` 收集 `riskGroups`（:233-238）。**`SupplierAccountRateGuardGroup.ID` 就是本地分组 `groups.id`。**
- **候选来源**：`ListAccountRateGuardCandidates`（`supplier_account_rate_guard_repo.go`）从 `supplier_provider_accounts` 出发 `LEFT JOIN account_groups` + `LEFT JOIN groups local_group`，**当前没有任何分组过滤条件**——所以过滤必须放在 Go 侧。
- **三条容易混淆的守护链路**：

  | 链路 | 任务码 / 配置键 | 分组级开关 |
  | --- | --- | --- |
  | 分组倍率守护 | `supplier_rate_guard`（`SupplierRateGuardConfig{MaxSnapshotAge}`） | **已有** `supplier_provider_groups.rate_guard_enabled`（迁移 226） |
  | 账号倍率守护 | `supplier_account_rate_guard` | **无（本次要做）** |
  | 账号健康守护 | `supplier_account_health_guard` | 另一套账号多选体系 |

  另有上游管理模块的 `UpstreamAccountRateGuardPanel.vue` / `upstream_account_rate_guard_config` **面板标题也叫「账号倍率守护」，但与本次无关**。

- **按实体多选的既有范式**：健康守护的 `AccountHealthGuardAccountIDs []int64`（:75）→ `normalizeSupplierAccountHealthGuardAccountIDs`（`supplier_account_health_guard.go:731`，去重 + 过滤 `<= 0` + 排序）→ 前端 `BaseDialog` + 原生 checkbox 列表 + `normalizePositiveAccountIDs`（`SupplierAutomationView.vue:2311`）。**分组开关照抄这一整套形态即可**。
- **跳过留痕机制**：`SupplierAccountRateGuardLogResultSkipped = "skipped"` 是既有枚举（`supplier_account_rate_guard.go:30`），前端 `SupplierAccountRateGuardUnbindLog.result` 也已含 `'skipped'`。既有三种 skip 原因：未匹配/多重匹配、倍率非法、执行时绑定已消失。
- **前端现状**：`SupplierAutomationView.vue`（约 5100 行）的任务操作区在第 97 行按 `task_code === 'supplier_account_rate_guard'` 显示专属按钮组；编辑弹窗的策略区按任务码分支（第 243 行分组倍率守护、第 253 行健康守护、第 283 行数据清理）——**账号倍率守护目前没有策略区，新区块插在第 251 行后**。
- **测试范式**：后端 `supplier_account_rate_guard_test.go` 有完整的 stub 套件（syncer / repo / remover）；前端 `SupplierAutomationView.spec.ts` 是**源码文本断言测试**（`readFileSync` + `toContain`），新增标识符必须同步补断言，否则"改了但测试没跟着动"。

## Goals / Non-Goals

**Goals:**

- 分组开关的语义是「该分组是否参与守护」，默认参与；只有显式关闭才产生行为差异。
- 判定收口在**唯一一处** `processCandidate`：让关闭分组在进入风险分组集合时就消失，而不是在解绑动作前临时判断，避免出现"计数说解绑了但实际没解绑"这类不一致。
- 配置向后兼容：字段缺失、值为 null/空、含非法 ID 都必须安全降级，且不影响既有三种 skip 语义。
- 跳过必须可追溯：全部风险分组被关闭时留一条 `skipped` 日志，而不是静默什么都不做。
- **不触发 Wire 重生成**：只改接口方法签名，不动构造函数参数。

**Non-Goals:**

- 不改动 `supplier_rate_guard`（分组倍率守护）链路——它已有 `rate_guard_enabled` 分组开关（迁移 226），守护对象是上游分组映射，语义不同。
- 不改动上游管理模块的账号倍率守护面板与 `upstream_account_rate_guard_config`。
- 不改动健康守护的账号多选体系。
- 不为关闭分组做"已解绑分组回滚"或"跳过时保留原绑定"的特殊处理：本次只做**不主动解绑**，不回溯历史动作。
- 不改变"上游倍率 > 分组倍率"的判定阈值与比较方式。
- 不引入任务级开关与分组级开关的优先级冲突处理（任务级 `Enabled=false` 时本就不执行）。

## Decisions

### D1：开关以分组 ID 黑名单存在任务配置 JSON 中

字段名 `account_rate_guard_disabled_group_ids`，Go 类型 `[]int64`。**空数组或缺失 = 全部分组参与守护**（默认开启）。

- 备选一：白名单。在"默认开启"语义下白名单会退化成"上线时必须把所有分组勾一遍"，且新增分组会自动变成不参与——与需求相反，否决。
- 备选二：给 `groups` 表加布尔列（如 `account_rate_guard_enabled`）。语义更实体化，但需要迁移 + 打通分组仓储，且分组删除后会留下脏配置；本次只是"任务配置层面的开关"，沿用 `config_json` 改动面最小，**无需数据库迁移**。
- 备选三：命名 `ignored_group_ids`。更短，但 `disabled` 与既有 skip 原因文案（"已关闭倍率守护"）措辞一致，可读性更好。

### D2：归一化放在读取时做，而不是写配置时强校验

新增 `supplierAccountRateGuardDisabledGroupSet(groupIDs []int64) map[int64]struct{}`：nil 或空列表直接返回 nil；否则丢弃 `<= 0` 的 ID 后建集合。

- 不在 `UpdateTask` 里拒绝非法 ID，因为既有账号 ID 列表走的就是"归一化时静默丢弃"策略，两处保持一致比引入一套新校验更不容易出错。
- 用 `map` 而非切片做成员判定：分组数量可能上百，`processCandidate` 是逐候选调用的热点路径。
- 返回 `nil` 而不是空 map：`_, ok := nilMap[k]` 在 Go 里安全返回 `ok=false`，省一次分配。

### D3：过滤收口在 `processCandidate` 的 `riskGroups` 收集循环

在既有阈值判断之后、`append` 之前追加关闭判断：

```go
for _, group := range candidate.Groups {
    if effectiveRate-group.RateMultiplier <= 0.0000001 {
        continue
    }
    if _, disabled := disabledGroups[group.ID]; disabled {
        skippedGroupCount++
        continue
    }
    riskGroups = append(riskGroups, group)
}
```

这样 `riskGroups` 天然不含关闭分组，后续 preview 的 `Planned` 日志、execute 的 `groupIDs` 入参、`result.RiskGroups` / `UnboundGroups` 计数、`DisabledAccounts` 判定**全部自动正确**，不需要各自打补丁。

### D4：全部风险分组被关闭时写一条 skipped 日志

`len(riskGroups) == 0 && skippedGroupCount > 0` 时，写一条 `base` 日志（`Result=skipped`，`ErrorMessage` 为「N 个风险分组已关闭倍率守护，本次跳过」）并 `result.Skipped++`。

- 为什么必须留痕：否则用户会看到"检查了账号但什么都没发生"，无从判断是开关生效了还是守护压根没跑。
- 为什么只在 `skippedGroupCount > 0` 时写：本来就没有风险分组（`skippedGroupCount == 0`）时保持现状返回 `nil, nil`，避免日志被"无风险"刷屏。
- 为什么用一条汇总日志而不是每个分组一条：这些分组是**被配置主动排除**的，不是异常；逐分组刷 `skipped` 会把真正需要关注的三种既有 skip 原因淹掉。日志粒度与既有"未匹配/倍率非法"（按账号一条）保持一致。
- `ErrorMessage` 必须与既有三种原因措辞可区分，便于日志检索。

### D5：接口加参而不是给 Service 注入配置仓储

`SupplierAccountRateGuardRunner.Run` 加 `disabledGroupIDs []int64` 参数，由 `executeTask` 从 `task.Config` 透传。

- 备选：给 `SupplierAccountRateGuardService` 注入 `SupplierAutomationTaskRepository`，让它自己读配置。否决——`NewSupplierAccountRateGuardService` 加参数会改动依赖图，**必须重跑 Wire**；而且会让守护服务反向依赖自动化任务模块，耦合方向错误。
- `SupplierRateGuardRunner` 已是"传 config"形态（`executeTask` 里 `s.rateGuard.Run(ctx, SupplierRateGuardConfig{...}, time.Now())`），本次改动与它对齐。

### D6：前端分组列表走 `@/api/admin/groups` 的 `getAll()`

`GET /admin/groups/all` 返回 `AdminGroup[]`（含 `id/name/platform/rate_multiplier/status`），与账号绑定的 `group_id` **同一口径**。

- 备选：复用本页面的 `listSupplierGroups`（`supplierProviderData.ts`）。但它是**供应商视角的分组映射**，语义与本地分组开关不一致（一个本地分组可能对应多个上游账号的不同映射），容易让用户误以为是对供应商分组设开关。否决。
- 需注意 `getAll()` 默认只返回启用分组，若用户已关闭某些分组的开关后又把分组停用，该配置会变成"悬空 ID"——后端归一化会保留它（只丢弃 `<= 0`），行为是"分组恢复启用后继续跳过"，符合直觉。
