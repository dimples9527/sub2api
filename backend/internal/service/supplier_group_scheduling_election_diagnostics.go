package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 分组择优调度的「诊断快照」。
//
// 为什么要有它：这个任务的业务执行过程一行日志都不打（只有调度器级 slog），
// 排查「某个分组为什么开着好几个账号 / 为什么没切换」只能翻 supplier_automation_runs 的
// result_detail；而那里的 items 只记录「发生了变化 / 被锁定保住 / 被闸门拦住」的账号，
// 光看它推不出「某组当前到底开着几个」—— 那个数只存在于 accounts.schedulable。
//
// 所以这里把排查所需的四类上下文一次性拼成**纯文本**，由管理员一键复制带走：
//  ① 任务配置（TopN、锁定名单、必需模型等）
//  ② 分组现状（成员数 / 已开启数 / TopN），实时取自 accounts.schedulable
//  ③ 开启中的账号及其所属分组 —— 判断「跨组共享成员」的唯一依据
//  ④ 最近 N 轮的逐账号、逐分组决策（elected / rank / locked / required_models）
//
// 为什么输出纯文本而不是结构化 JSON：使用者的动作是「把它拷走贴给别人（或 AI）看」，
// JSON 还得对方自己再解析一遍；文本自解释、可直接搜索，也不受前端渲染能力限制。
// 同理，字段顺序与标题都刻意写死，避免以后为了"好看"把它改成难读的样子。

// SupplierGroupSchedulingElectionDiagnosticsRow 是「账号 → 所属分组」的展开行。
// 刻意用扁平行而不是嵌套结构：一个查询就能同时算出「每组开启数」与「每账号的所属分组」，
// 两个聚合都在 Go 侧做 —— 既不用发两次查询，也不会出现两边口径不一致。
type SupplierGroupSchedulingElectionDiagnosticsRow struct {
	AccountID   int64
	AccountName string
	Platform    string
	Schedulable bool
	GroupID     int64
	GroupName   string
}

// SupplierGroupSchedulingElectionDiagnosticsStore 由 supplier_provider_data_repo 实现。
// 走窄接口断言而不是给 SupplierAutomationService 加构造参数 —— 加参数会让 Wire 生成代码失效。
type SupplierGroupSchedulingElectionDiagnosticsStore interface {
	ListGroupSchedulingElectionDiagnosticsRows(ctx context.Context) ([]SupplierGroupSchedulingElectionDiagnosticsRow, error)
}

const (
	// SupplierGroupSchedulingElectionDiagnosticsDefaultRunLimit 是默认回溯的运行轮数。
	SupplierGroupSchedulingElectionDiagnosticsDefaultRunLimit = 3
	// MaxSupplierGroupSchedulingElectionDiagnosticsRunLimit 是回溯轮数上限。
	// 快照是给人读的，轮数太多会把真正要看的那几轮淹掉；要更长历史应该去看调度切换日志。
	MaxSupplierGroupSchedulingElectionDiagnosticsRunLimit = 10
)

// supplierGroupElectionDiagnosticsZone 固定用东八区渲染时间。
// 服务器时区可能是 UTC，而管理员看到的运行时间都是本地时间；快照要贴给别人看，
// 时区必须在文本里自证（表头已带 +08:00 偏移），不能依赖阅读者所在时区。
var supplierGroupElectionDiagnosticsZone = time.FixedZone("UTC+8", 8*3600)

// BuildGroupSchedulingElectionDiagnostics 生成诊断快照文本。
func (s *SupplierAutomationService) BuildGroupSchedulingElectionDiagnostics(ctx context.Context, runLimit int) (string, error) {
	if runLimit <= 0 {
		runLimit = SupplierGroupSchedulingElectionDiagnosticsDefaultRunLimit
	}
	if runLimit > MaxSupplierGroupSchedulingElectionDiagnosticsRunLimit {
		runLimit = MaxSupplierGroupSchedulingElectionDiagnosticsRunLimit
	}

	var b strings.Builder
	b.WriteString("=== 分组择优调度 · 诊断快照 ===\n")
	b.WriteString("生成时间: " + time.Now().In(supplierGroupElectionDiagnosticsZone).Format("2006-01-02 15:04:05 -07:00") + "\n")
	b.WriteString("任务: " + SupplierAutomationTaskGroupElection + "\n")

	topN := DefaultSupplierGroupSchedulingElectionTopN
	// topNByGroup 是「每组开启账号数」的分组级覆盖，section [2] 按它算每组的生效 TopN。
	var topNByGroup map[int64]int
	task, err := s.repo.GetTask(ctx, SupplierAutomationTaskGroupElection)
	if err != nil {
		return "", fmt.Errorf("读取分组择优任务配置失败: %w", err)
	}
	b.WriteString("\n[1] 任务配置\n")
	if task == nil {
		b.WriteString("  （未找到任务记录）\n")
	} else {
		cfg := task.Config
		if cfg.GroupElectionTopN > 0 {
			topN = cfg.GroupElectionTopN
		}
		topNByGroup = cfg.GroupElectionTopNByGroup
		b.WriteString(fmt.Sprintf("  enabled=%t  cron=%s  timeout=%ds\n", task.Enabled, task.CronExpression, task.TimeoutSeconds))
		b.WriteString(fmt.Sprintf("  top_n=%d（每组开启账号数）\n", cfg.GroupElectionTopN))
		b.WriteString("  top_n_by_group=" + supplierGroupElectionDiagnosticsTopNByGroup(cfg.GroupElectionTopNByGroup) + "\n")
		b.WriteString(fmt.Sprintf("  count_weight=%s  latency_weight=%s  switch_margin=%s  count_score_cap=%d\n",
			supplierGroupElectionDiagnosticsFloat(cfg.GroupElectionCountWeight),
			supplierGroupElectionDiagnosticsFloat(cfg.GroupElectionLatencyWeight),
			supplierGroupElectionDiagnosticsFloat(cfg.GroupElectionSwitchMargin),
			cfg.GroupElectionCountScoreCap))
		b.WriteString(fmt.Sprintf("  failure_threshold=%d  latency_window_minutes=%d  latency_min_samples=%d\n",
			cfg.GroupElectionFailureThreshold, cfg.GroupElectionLatencyWindowMinutes, cfg.GroupElectionLatencyMinSamples))
		b.WriteString(fmt.Sprintf("  keep_healthy_incumbent_global=%t\n", cfg.GroupElectionKeepHealthyIncumbentGlobal))
		b.WriteString("  keep_healthy_incumbent_group_ids=" + supplierGroupElectionDiagnosticsInt64List(cfg.GroupElectionKeepHealthyIncumbentGroupIDs) + "\n")
		b.WriteString("  keep_healthy_incumbent_excluded_group_ids=" + supplierGroupElectionDiagnosticsInt64List(cfg.GroupElectionKeepHealthyIncumbentExcludedGroupIDs) + "\n")
		b.WriteString("  disabled_group_ids=" + supplierGroupElectionDiagnosticsInt64List(cfg.GroupElectionDisabledGroupIDs) + "\n")
		b.WriteString(fmt.Sprintf("  dry_run=%t  dry_run_group_ids=%s\n",
			cfg.GroupElectionDryRun, supplierGroupElectionDiagnosticsInt64List(cfg.GroupElectionDryRunGroupIDs)))
		b.WriteString("  required_models=" + supplierGroupElectionDiagnosticsRequiredModels(cfg.GroupElectionRequiredModels) + "\n")
		b.WriteString(fmt.Sprintf("  priority_enabled_global=%t  priority_enabled_group_ids=%s\n",
			cfg.GroupElectionPriorityEnabledGlobal, supplierGroupElectionDiagnosticsInt64List(cfg.GroupElectionPriorityEnabledGroupIDs)))
	}

	store, ok := s.dataRepo.(SupplierGroupSchedulingElectionDiagnosticsStore)
	if !ok {
		return "", fmt.Errorf("supplier group scheduling election diagnostics store is required")
	}
	rows, err := store.ListGroupSchedulingElectionDiagnosticsRows(ctx)
	if err != nil {
		return "", fmt.Errorf("读取分组账号快照失败: %w", err)
	}

	type diagnosticsGroupAgg struct {
		groupID     int64
		groupName   string
		memberCount int
		openCount   int
	}
	type diagnosticsAccountAgg struct {
		accountID   int64
		accountName string
		platform    string
		schedulable bool
		groupNames  []string
	}
	groupAggs := make(map[int64]*diagnosticsGroupAgg)
	groupOrder := make([]int64, 0)
	accountAggs := make(map[int64]*diagnosticsAccountAgg)
	accountOrder := make([]int64, 0)
	for _, row := range rows {
		group := groupAggs[row.GroupID]
		if group == nil {
			group = &diagnosticsGroupAgg{groupID: row.GroupID, groupName: row.GroupName}
			groupAggs[row.GroupID] = group
			groupOrder = append(groupOrder, row.GroupID)
		}
		group.memberCount++
		if row.Schedulable {
			group.openCount++
		}
		account := accountAggs[row.AccountID]
		if account == nil {
			account = &diagnosticsAccountAgg{
				accountID:   row.AccountID,
				accountName: row.AccountName,
				platform:    row.Platform,
				schedulable: row.Schedulable,
			}
			accountAggs[row.AccountID] = account
			accountOrder = append(accountOrder, row.AccountID)
		}
		account.groupNames = append(account.groupNames, fmt.Sprintf("%d %s", row.GroupID, row.GroupName))
	}
	sort.Slice(groupOrder, func(i, j int) bool { return groupOrder[i] < groupOrder[j] })
	sort.Slice(accountOrder, func(i, j int) bool { return accountOrder[i] < accountOrder[j] })

	// effectiveTopN 与后端 topNForGroup 同口径：分组级覆盖优先，否则全局 TopN。
	// 表头写「全局默认」而各行 TopN 列写生效值，超员判定也按生效值——否则被单独调过 TopN 的分组
	// 会被错判成超员。
	effectiveTopN := func(groupID int64) int {
		if n, ok := topNByGroup[groupID]; ok && n > 0 {
			return n
		}
		return topN
	}

	b.WriteString(fmt.Sprintf("\n[2] 分组现状（实时，取自 accounts.schedulable；全局默认 TopN=%d，各行 TopN 为生效值）\n", topN))
	b.WriteString(fmt.Sprintf("  参与择优的分组共 %d 个\n", len(groupOrder)))
	b.WriteString("  group_id | 分组名 | 成员 | 已开启 | TopN | 状态\n")
	overflowCount := 0
	for _, groupID := range groupOrder {
		group := groupAggs[groupID]
		effTopN := effectiveTopN(group.groupID)
		status := "-"
		if group.openCount > effTopN {
			status = fmt.Sprintf("超员 +%d", group.openCount-effTopN)
			overflowCount++
		}
		b.WriteString(fmt.Sprintf("  %d | %s | %d | %d | %d | %s\n",
			group.groupID, group.groupName, group.memberCount, group.openCount, effTopN, status))
	}
	b.WriteString(fmt.Sprintf("  → 超员分组数: %d\n", overflowCount))

	b.WriteString("\n[3] 开启中的账号（schedulable=true）及其所属分组\n")
	b.WriteString("  账号ID | 名称 | 平台 | 所属分组\n")
	openAccountCount := 0
	for _, accountID := range accountOrder {
		account := accountAggs[accountID]
		if !account.schedulable {
			continue
		}
		openAccountCount++
		b.WriteString(fmt.Sprintf("  %d | %s | %s | %s\n",
			account.accountID, account.accountName, account.platform, strings.Join(account.groupNames, " / ")))
	}
	if openAccountCount == 0 {
		b.WriteString("  （无）\n")
	}
	b.WriteString(fmt.Sprintf("  → 全库开启账号数: %d\n", openAccountCount))

	b.WriteString(fmt.Sprintf("\n[4] 最近 %d 轮决策明细（只含发生变更 / 被锁定保住 / 被闸门拦住的账号）\n", runLimit))
	runs, err := s.repo.ListRuns(ctx, SupplierAutomationRunListParams{
		TaskCode: SupplierAutomationTaskGroupElection,
		Page:     1,
		PageSize: runLimit,
	})
	if err != nil {
		return "", fmt.Errorf("读取分组择优运行记录失败: %w", err)
	}
	if len(runs.Items) == 0 {
		b.WriteString("  （无运行记录）\n")
	}
	for _, run := range runs.Items {
		startedAt := run.StartedAt.In(supplierGroupElectionDiagnosticsZone).Format("2006-01-02 15:04:05")
		detail := run.ResultDetail
		if detail == nil || detail.GroupElection == nil {
			b.WriteString(fmt.Sprintf("\n--- [%s] run=%d trigger=%s status=%s （无分组择优明细）---\n",
				startedAt, run.ID, run.TriggerSource, run.Status))
			continue
		}
		election := detail.GroupElection
		b.WriteString(fmt.Sprintf("\n--- [%s] run=%d trigger=%s status=%s 开%d/关%d/未变%d ---\n",
			startedAt, run.ID, run.TriggerSource, run.Status,
			election.EnabledCount, election.DisabledCount, election.UnchangedCount))
		for _, item := range election.Items {
			b.WriteString(fmt.Sprintf("  %d %s  bef=%t→aft=%t action=%s reason=%s\n",
				item.AccountID, item.AccountName,
				item.SchedulableBefore, item.SchedulableAfter, item.Action, item.Reason))
			for _, decision := range item.GroupDecisions {
				b.WriteString("      · " + supplierGroupElectionDiagnosticsDecision(decision) + "\n")
			}
		}
	}
	return b.String(), nil
}

// supplierGroupElectionDiagnosticsDecision 把「某账号在某分组的裁决依据」压成一行。
// 只输出能解释结果的那几个字段（名次 / 入选 / 锁定 / 补选），评分分量与权重留在 result_detail 里 ——
// 快照是给人快速定位用的，堆满浮点分量反而看不清重点。
func supplierGroupElectionDiagnosticsDecision(decision SupplierGroupSchedulingElectionDecisionDetail) string {
	rank := "rank=-"
	if decision.Scored {
		rank = fmt.Sprintf("rank=%d/%d", decision.Rank, decision.RankTotal)
	}
	parts := []string{
		fmt.Sprintf("%s(%d)", decision.GroupName, decision.GroupID),
		rank,
		fmt.Sprintf("top_n=%d", decision.TopN),
		fmt.Sprintf("elected=%t", decision.Elected),
		fmt.Sprintf("locked=%t", decision.Locked),
	}
	if decision.TestFailed {
		parts = append(parts, "test_failed=true")
	}
	if decision.NoAlternative {
		parts = append(parts, "no_alternative=true")
	}
	if len(decision.RequiredModels) > 0 {
		parts = append(parts, "required_models=["+strings.Join(decision.RequiredModels, ",")+"]")
	}
	return strings.Join(parts, " ")
}

func supplierGroupElectionDiagnosticsInt64List(values []int64) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatInt(value, 10))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func supplierGroupElectionDiagnosticsFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

// supplierGroupElectionDiagnosticsTopNByGroup 把「每组开启账号数」的分组级覆盖压成紧凑文本，
// 形如 {12:2 34:5}，按 groupID 升序，空覆盖输出 {}。口径与 required_models 的格式化保持一致。
func supplierGroupElectionDiagnosticsTopNByGroup(topNByGroup map[int64]int) string {
	if len(topNByGroup) == 0 {
		return "{}"
	}
	groupIDs := make([]int64, 0, len(topNByGroup))
	for groupID := range topNByGroup {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	parts := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		parts = append(parts, fmt.Sprintf("%d:%d", groupID, topNByGroup[groupID]))
	}
	return "{" + strings.Join(parts, " ") + "}"
}

func supplierGroupElectionDiagnosticsRequiredModels(models map[int64][]string) string {
	if len(models) == 0 {
		return "{}"
	}
	groupIDs := make([]int64, 0, len(models))
	for groupID := range models {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	parts := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		parts = append(parts, fmt.Sprintf("%d:[%s]", groupID, strings.Join(models[groupID], ",")))
	}
	return "{" + strings.Join(parts, " ") + "}"
}
