## Purpose

规定供应商模块的「供应商账号倍率守护」自动化任务（任务码 `supplier_account_rate_guard`）按本地分组控制适用范围的行为：哪些分组参与自动守护、关闭分组在预览与执行中的表现、以及配置的默认值与兼容性。

## ADDED Requirements

### Requirement: 按分组控制是否参与守护

系统 MUST 支持按本地分组（`groups` 表，即账号绑定的 `group_id`）配置是否参与账号倍率守护。配置中未出现该分组时 MUST 视为参与；被显式关闭的分组 MUST 不参与守护。

配置项 MUST 为任务配置里的分组 ID 列表 `account_rate_guard_disabled_group_ids`，MUST 只包含正整数 ID。空列表或字段缺失 MUST 等价于"所有分组都参与守护"。

#### Scenario: 未配置任何分组

- **WHEN** 任务配置中不存在 `account_rate_guard_disabled_group_ids` 字段（或为空列表）
- **THEN** 所有分组 MUST 参与守护，行为与本次改动前一致

#### Scenario: 关闭某个分组

- **WHEN** `account_rate_guard_disabled_group_ids` 包含分组 A 的 ID
- **THEN** 分组 A MUST 不参与守护，其他分组 MUST 不受影响

#### Scenario: 配置含非正数 ID

- **WHEN** `account_rate_guard_disabled_group_ids` 为 `[3, 0, 3, -1, 5]`
- **THEN** 系统 MUST 只把 `3` 与 `5` 视为关闭，MUST NOT 因 `0` / `-1` 误伤任何分组

### Requirement: 关闭分组的账号不被解绑

账号倍率守护在判定"哪些分组需要因上游倍率高于分组倍率而解绑"时，MUST 排除所有已关闭的分组。被排除的分组 MUST NOT 出现在解绑结果中，也 MUST NOT 影响其他分组的解绑判定。

#### Scenario: 账号绑定了开启与关闭两种分组

- **WHEN** 某账号绑定分组 A（参与）与分组 B（关闭），且上游倍率均高于两者
- **THEN** 系统 MUST 只解绑分组 A
- **THEN** 分组 B 的绑定 MUST 保持不变

#### Scenario: 账号只绑定关闭的分组

- **WHEN** 某账号仅绑定关闭的分组，且上游倍率高于该分组
- **THEN** 系统 MUST 不调用解绑，MUST NOT 产生任何解绑记录
- **THEN** 该账号的绑定集合 MUST 保持不变

#### Scenario: 预览与执行一致

- **WHEN** 在关闭分组存在的前提下先执行检测预览、再执行真正的守护
- **THEN** 预览中 MUST NOT 出现该分组的解绑计划
- **THEN** 执行结果 MUST NOT 包含该分组的解绑记录

### Requirement: 关闭分组的跳过可追溯

当某账号的风险分组**全部**因关闭而被跳过时，系统 MUST 记录一条可在解绑日志中查到的 `skipped` 记录，原因文案 MUST 说明是分组已关闭倍率守护。该记录 MUST NOT 被计入解绑数量或风险分组数量，也 MUST NOT 计入失败数。

某账号本就没有风险分组时（与开关无关），系统 MUST NOT 产生跳过记录。

#### Scenario: 全部风险分组被关闭

- **WHEN** 某账号的风险分组均已关闭
- **THEN** 系统 MUST 生成一条 `skipped` 结果记录，原因文案 MUST 表明分组已关闭倍率守护
- **THEN** 该记录 MUST NOT 计入 `risk_groups` 或 `unbound_groups`
- **THEN** 该次判定 MUST NOT 计入失败数

#### Scenario: 部分风险分组被关闭

- **WHEN** 某账号同时存在参与的风险分组与已关闭的风险分组
- **THEN** 系统 MUST 只对参与的分组产生解绑记录
- **THEN** `risk_groups` MUST 只统计参与的分组

#### Scenario: 无风险分组

- **WHEN** 某账号没有任何分组满足"上游倍率高于分组倍率"
- **THEN** 系统 MUST NOT 产生任何日志，行为与本次改动前一致

### Requirement: 配置读写保持向后兼容

读取任务配置时，系统 MUST 对缺失、`null` 或含非法值的 `account_rate_guard_disabled_group_ids` 安全降级，MUST NOT 因该字段导致任务执行失败。写入配置时，系统 MUST 保持其余任务配置字段（保留天数、分组倍率守护快照时效、健康守护全部配置）的既有行为不变。

#### Scenario: 旧版本配置读取

- **WHEN** 数据库中存有本次改动前写入的 `config_json`
- **THEN** 系统 MUST 正常读取，关闭分组列表视为空，守护行为不变

#### Scenario: 整体覆盖写入时不丢失字段

- **WHEN** 管理员在前端修改间隔或其他配置后保存任务
- **THEN** 已配置的 `account_rate_guard_disabled_group_ids` MUST NOT 被静默清空
- **THEN** 反之，保存关闭分组时其余配置字段也 MUST 保持原值

### Requirement: 管理端可维护分组开关

管理端供应商自动化任务编辑弹窗在「供应商账号倍率守护」任务下 MUST 提供维护"不参与守护的分组"的界面，展示当前关闭的分组数量，并在列表为空时明确表达"所有分组都参与守护"。

界面 MUST 复用项目既有组件，MUST NOT 为此修改通用组件。分组列表 MUST 来自本地分组（`GET /admin/groups/all`），与账号绑定的 `group_id` 同一口径。

#### Scenario: 新增关闭分组

- **WHEN** 管理员在同一界面选中分组并保存
- **THEN** 保存成功后 MUST 提示成功，并刷新摘要数量

#### Scenario: 清空关闭列表

- **WHEN** 管理员清空所有已选分组并保存
- **THEN** 系统 MUST 恢复为"所有分组都参与守护"

### Requirement: 不影响其他守护链路

本次改动 MUST NOT 改变分组倍率守护（任务码 `supplier_rate_guard`，已有 `supplier_provider_groups.rate_guard_enabled`）、账号健康守护（任务码 `supplier_account_health_guard`）以及上游管理模块的账号倍率守护面板的行为。

#### Scenario: 分组倍率守护不受影响

- **WHEN** 管理员关闭账号倍率守护的某分组，同时分组倍率守护处于启用状态
- **THEN** 分组倍率守护的执行结果 MUST NOT 因本次改动产生差异
