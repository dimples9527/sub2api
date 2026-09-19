## 1. 后端配置字段与接口签名

- [x] 1.1 `service.SupplierAutomationConfig` 新增 `AccountRateGuardDisabledGroupIDs []int64`，JSON 键 `account_rate_guard_disabled_group_ids`（`supplier_automation_service.go:59` 区块）
- [x] 1.2 `SupplierAccountRateGuardRunner.Run` 接口方法加参 `disabledGroupIDs []int64`（:204）
- [x] 1.3 `executeTask` 的 `SupplierAutomationTaskAccountRateGuard` 分支透传 `task.Config.AccountRateGuardDisabledGroupIDs`
- [x] 1.4 确认**无需重跑 Wire**：`NewSupplierAccountRateGuardService` 构造参数未变，依赖图未变（`wire_gen.go` 零差异）

## 2. 分组过滤与跳过留痕

- [x] 2.1 新增 `supplierAccountRateGuardDisabledGroupSet(groupIDs []int64) map[int64]struct{}`：nil/空 → nil；丢弃 `<= 0` 后建集合
- [x] 2.2 `SupplierAccountRateGuardService.Run` 加参并在候选循环前构建关闭集合
- [x] 2.3 `processCandidate` 的 `riskGroups` 收集循环排除关闭分组，同时累计 `skippedGroupCount`
- [x] 2.4 全部风险分组被关闭时写一条 `skipped` 日志（`ErrorMessage` 为「N 个风险分组已关闭倍率守护，本次跳过」）+ `result.Skipped++`
- [x] 2.5 无风险分组（`skippedGroupCount == 0`）时保持返回 `nil, nil`，不产生日志

## 3. 后端测试

- [x] 3.1 改 `supplier_account_rate_guard_test.go` 6 处 `guard.Run` 调用签名（补 `nil`）
- [x] 3.2 改 `supplier_automation_service_test.go` 两个 stub 的 `Run` 签名，并让 `supplierAutomationAccountRateGuardStub` 记录 `disabledGroupIDs`
- [x] 3.3 新增 `newSupplierAccountRateGuardGroupSwitchFixture` 共用夹具（2 个风险分组 + 1 个正常分组）
- [x] 3.4 新增用例：preview 跳过关闭分组、execute 只解绑未关闭分组、全部关闭时写 skipped 且不调 remover、空列表等价于全开、非正数 ID 不误伤、无风险分组不产生日志
- [x] 3.5 新增用例：`PassesDisabledGroupIDsToAccountRateGuard`（透传生效）、`PassesNilWhenNoDisabledGroupConfigured`（默认全开）
- [x] 3.6 `go vet ./internal/service/` 通过；`go test ./internal/service/ -run "SupplierAccountRateGuard|SupplierAutomation"` 中本次相关用例全绿
- [x] 3.7 新增仓储层持久化用例（`supplier_automation_repo_test.go`）：
  - 用**从真实库抓下来的旧 `config_json`** 做夹具，断言旧配置读出的关闭分组为空、且同结构体其他字段未被挤掉
  - 断言新字段确实出现在被整块覆盖写入的 JSON 中，且空列表序列化为 `[]` 而非被省略

## 4. 前端类型与配置默认值

- [x] 4.1 `api/admin/supplierAutomation.ts` 的 `SupplierAutomationConfig` 新增 `account_rate_guard_disabled_group_ids?: number[]`（带注释说明"空列表=全部分组参与"）
- [x] 4.2 `SupplierAutomationView.vue` 的 `editForm.config` 默认值补充该字段为 `[]`
- [x] 4.3 新增 `applyAccountRateGuardDefaults()`（`openEdit` 与 `saveTask` 两处调用），复用 `normalizePositiveAccountIDs` 归一化去重升序后再提交

## 5. 前端分组开关界面

- [x] 5.1 在编辑弹窗 `supplier_rate_guard` 策略区之后新增 `v-if="editForm.task_code === 'supplier_account_rate_guard'"` 策略区（`sp-rate-guard-scope-card`）
- [x] 5.2 分组列表走 `@/api/admin/groups` 的 `getAllIncludingInactive()`（含停用分组，避免开关状态"看不见但生效"）
- [x] 5.3 复用 `BaseDialog` + 原生 checkbox 列表（照健康守护账号弹窗的形态），样式沿用 `sp-*` 命名（青色系与健康守护的蓝青色区分）
- [x] 5.4 空态文案明确表达"所有分组都参与守护。新增分组也会自动参与。"
- [x] 5.5 使用 `frontend-design` 技能后判断为"在既有管理后台追加配置区块"，沿用健康守护策略区既有视觉范式；只复用现有组件，未修改通用组件
- [x] 5.6 保存/失败反馈复用页面既有的 `showSuccess`/`showError` 与 `extractApiErrorMessage`

## 6. 前端测试

- [x] 6.1 `SupplierAutomationView.spec.ts` 更新 3 个既有守卫断言（`BaseDialog` 4→5、策略区 3→4 并纳入新任务码、原生 `<input>` 1→2 并补新 checkbox 断言）+ 新增 5 个用例
- [x] 6.2 `cd frontend && npx vitest run src/views/admin/supplier-management/SupplierAutomationView.spec.ts` 通过（**93/93**）
- [x] 6.3 `npx vue-tsc --noEmit` 无新增错误

## 7. 验证

- [x] 7.1 `cd backend && go build ./internal/service/` 通过
- [x] 7.2 `cd backend && go build ./...` 通过（另经 `go vet ./internal/repository/ ./internal/service/`）
- [x] 7.3 `cd backend && go test -tags unit ./internal/handler/admin/ -run SupplierAutomation` 通过
- [x] 7.4 持久化链路端到端验证（**因本地库无真实风险分组，改为验证"配置能存进/读回/透传"这条有不确定性的链路**）：
  - 真实 DB（docker 容器内免密 psql，事务 + ROLLBACK）验证 `jsonb` 列可承接 `[81,66]` 与 `[]`，回滚后配置逐字节复原
  - Go 层用真实旧配置夹具验证序列化往返（见 3.7）
  - `service` 层透传已有单元测试覆盖（见 3.5）
  - ⚠️ **未做**：UI 上真实点击"关闭某分组 → 检测预览确认不出现"。原因：本地库当前**无任何真实风险分组**（判定差值恒为负），且 4000 端口实例是带真实上游同步副作用的生产环境，不宜重启。
- [x] 7.5 回归确认：本次相关用例 `SupplierAccountRateGuard` **13/13 通过**；`supplier_account_health_guard` 与 `supplier_rate_guard` 未受影响（service 全量仅剩一个预存在缺陷失败，已用 `git show HEAD:` 基线源码实锤与本改动无关）；仓储包全量通过

## 8. 文档与记忆

- [x] 8.1 修正 `openspec/changes/account-rate-guard-group-switch/` 下 4 份文档为供应商模块口径
- [x] 8.2 修正 `.workbuddy-ai/memory/2026-09-19.md` 中关于模块定位的错误描述（并补充端到端验证记录）
- [x] 8.3 在项目长期记忆中沉淀「三个同名/近名守护链路的区分方法」+ 「4000 实例是活环境」+ 「`config_json` 整块覆盖写的加字段检查清单」

## 9. 待用户确认（不在本次范围）

- [ ] 9.1 是否需要在任务列表的结果详情区把「跳过」指标拆分为"分组关闭导致的跳过"与"其他跳过"
- [ ] 9.2 是否需要对关闭分组做"已解绑分组不回溯"的界面说明加强（当前 spec 已声明不回溯，但前端文案尚未落地）
- [ ] 9.3 是否删除 git 事故时留下的备份目录 `.git.bak-20260919-081802/`（3.5M，已加入 `.gitignore`）
