package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	DefaultSupplierGroupSchedulingElectionTopN = 1
	MaxSupplierGroupSchedulingElectionTopN     = 100

	SupplierGroupSchedulingElectionTestStatusSuccess = "success"
	SupplierGroupSchedulingElectionTestStatusFailed  = "failed"

	SupplierGroupSchedulingElectionActionNone     = "none"
	SupplierGroupSchedulingElectionActionEnabled  = "enabled"
	SupplierGroupSchedulingElectionActionDisabled = "disabled"

	SupplierGroupSchedulingElectionReasonFailed      = "测试失败，关闭调度"
	SupplierGroupSchedulingElectionReasonElected     = "分组内最优，开启调度"
	SupplierGroupSchedulingElectionReasonNotElected  = "非分组最优，关闭调度"
	SupplierGroupSchedulingElectionReasonUntested    = "尚未测试，保持原状"
	SupplierGroupSchedulingElectionReasonWriteFailed = "更新调度状态失败"
	// 下面两条是「失败不再立刻关」后的新出口：
	// 未达阈值时保持原状等下一轮，或分组没有备选账号时保留调度（绝不把分组关成空组）。
	// 未达阈值的理由带上进度（第几次/共几次），否则运维只看到"没关"却不知道还要等几轮。
	SupplierGroupSchedulingElectionReasonFailedPendingFmt = "连续失败 %d/%d 次，未达阈值，暂不关闭"
	// 已经关着的账号也会落到这一条（本任务只关不重开），所以措辞用"保持现状"而不是"保留调度"。
	SupplierGroupSchedulingElectionReasonKeepLastOne = "分组内无备选账号，保持现状待人工确认"

	// SupplierGroupSchedulingElectionCountScoreCap 是"连续成功次数"对综合分的贡献上限。
	// 健康守护的连续成功计数只增不减（失败才归零），不封顶就会让现任赢家永久固化：
	// 50 次与 51 次没有实质区别，不该因此压过一个用时明显更短的账号。
	SupplierGroupSchedulingElectionCountScoreCap = 10

	// SupplierGroupSchedulingElectionScoreEpsilon 是综合分的比较容差。
	// 综合分含浮点除法，不能用 == 判并列，否则"算出来应该一样快"的两个账号会因尾差
	// 落到 ID 兜底之外的分支，破坏黏性。
	SupplierGroupSchedulingElectionScoreEpsilon = 1e-9

	// DefaultSupplierGroupSchedulingElectionCountWeight / ...LatencyWeight
	// 是两项的默认话语权。两项都先归一到 [0,1] 再加权，所以比值就是相对重要性：
	// 1.0 : 0.5 相当于「用时最多抵 Cap/2 = 5 次连续成功」。
	// 用时是单次采样、抖动远大于次数，所以默认仍让它弱于次数——
	// 既能让明显更快的账号翻盘，又不至于一次网络抖动换掉长期稳定的赢家。
	DefaultSupplierGroupSchedulingElectionCountWeight   = 1.0
	DefaultSupplierGroupSchedulingElectionLatencyWeight = 0.5

	// MaxSupplierGroupSchedulingElectionWeight 是权重上限，防止配置误填把某一项无限放大。
	MaxSupplierGroupSchedulingElectionWeight = 100.0

	// DefaultSupplierGroupSchedulingElectionFailureThreshold 是连续多少个调度周期都判失败，才真正关闭调度。
	// 2 是刻意的选择：1 等于旧的「一次失败立刻关」，一次网络抖动就会误伤；
	// 3 以上会让真正坏掉的账号在分组里多活太久。配成 1 可退回旧行为。
	DefaultSupplierGroupSchedulingElectionFailureThreshold = 2
	MaxSupplierGroupSchedulingElectionFailureThreshold     = 100

	// DefaultSupplierGroupSchedulingElectionSwitchMargin 是「换人」的迟滞死区，含义是延迟的相对比例：
	// 挑战者必须在（可靠性校正后的）延迟上比在任者快出这个比例才夺位，否则保持现任。0.15 = 快 15%。
	// 做成延迟比例而不是综合分死区，是因为组内 min-max 归一化会把任意延迟差放大到满量程，
	// 综合分层面的死区在「两个候选」这种最常见的抖动场景里根本不生效（分差恒为 LatencyWeight）。
	DefaultSupplierGroupSchedulingElectionSwitchMargin = 0.15
	// MaxSupplierGroupSchedulingElectionSwitchMargin 是死区上限。配得过大会让在任者近乎永不被换下
	// （除非它测试失败），本质变成「谁先占位谁通吃」，故设 0.95 上限防误配（另有 5% 地板兜底）。
	MaxSupplierGroupSchedulingElectionSwitchMargin = 0.95

	// DefaultSupplierGroupSchedulingElectionLatencyWindowMinutes 是「最近一段时间平均延迟」的时间窗。
	// 延迟改从 supplier_account_health_history 里取最近这么多分钟的成功样本求平均，而不是用
	// accounts.extra 里那个被反复覆盖的单值——单采样抖动远大于均值，是抖动的源头之一。
	DefaultSupplierGroupSchedulingElectionLatencyWindowMinutes = 30
	MaxSupplierGroupSchedulingElectionLatencyWindowMinutes     = 1440

	// DefaultSupplierGroupSchedulingElectionLatencyMinSamples 是信任窗口均值所需的最少成功样本数。
	// 低于它就不信任这个均值（几次采样太抖），回退到 accounts.extra 里的最近单值——
	// 等于退回今天的行为，绝不比现状更差。
	DefaultSupplierGroupSchedulingElectionLatencyMinSamples = 3
	MaxSupplierGroupSchedulingElectionLatencyMinSamples     = 1000

	// supplierGroupElectionSuccessRateFloor 是成功率惩罚的下限（写死，不做成配置——它是防爆保护）。
	// 有效延迟 = 窗口均值 / max(成功率, floor)：只算成功样本会掩盖「偶尔快一下、实则一直在失败」
	// 的账号，用成功率把失败重新计入代价。floor 0.2 表示失败惩罚最多放大 5 倍，防止成功率趋 0 时除爆。
	supplierGroupElectionSuccessRateFloor = 0.2

	// supplierGroupElectionFailedCountExtraKey 是本任务自己维护的「连续失败轮次」。
	// 不能复用健康守护的 supplier_health_guard_failure_count：那个是健康守护自己检测周期里的计数，
	// 而本任务裁决依据的 last_test_status 只有账号测试才会更新，两者不同源、清零时机也不一致。
	supplierGroupElectionFailedCountExtraKey = "supplier_group_election_failed_count"

	// 下面两个是 supplierGroupElectionAccount.hold 的取值：本轮"故意没关"的原因分类。
	// 用机器可读的枚举而不是拿 reason 文案做匹配，免得以后改一句话就把统计打碎。
	supplierGroupElectionHoldPending       = "failed_pending"
	supplierGroupElectionHoldNoAlternative = "no_alternative"
)

// SupplierGroupSchedulingElectionConfig 是分组择优调度任务的配置。
type SupplierGroupSchedulingElectionConfig struct {
	// TopN 是每个分组允许保持开启调度的最优账号数量，默认 1（严格单活）。
	TopN int `json:"group_scheduling_election_top_n"`
	// DisabledGroupIDs 存"被关闭择优"的分组 ID，空列表表示全部分组都参与。
	// 存关闭项而不是开启项，是为了让"新增分组默认参与"无需数据迁移。
	DisabledGroupIDs []int64 `json:"group_scheduling_election_disabled_group_ids"`
	// CountWeight / LatencyWeight 是"连续成功次数"与"测试用时"在综合分里的相对话语权。
	// 两项都先各自归一到 [0,1] 再加权，所以它们的比值就是两者的相对重要性：
	// 例 1.0 : 0.5 表示用时的影响上限是次数的一半（相当于"用时最多抵 Cap/2 次连续成功"）。
	// 0 或缺失都回落到默认值；LatencyWeight 显式配 0 即退化为"只看次数"的旧行为。
	CountWeight   float64 `json:"group_scheduling_election_count_weight"`
	LatencyWeight float64 `json:"group_scheduling_election_latency_weight"`
	// FailureThreshold 是连续多少个调度周期都判失败才关闭调度，默认 2。
	// 配成 1 即退回「一次失败立刻关」的旧行为；0 或缺失都按默认值处理。
	FailureThreshold int `json:"group_scheduling_election_failure_threshold"`
	// SwitchMargin 是换人的迟滞死区，取延迟的相对比例（0.15 = 挑战者要快 15% 才换人），默认 0.15；
	// 0 或缺失回落默认值。想几乎关掉迟滞请填一个极小正数（如 0.0001），不要把 0 当开关用——那会被当成缺失。
	SwitchMargin float64 `json:"group_scheduling_election_switch_margin"`
	// LatencyWindowMinutes 是「最近平均延迟」的时间窗（分钟），默认 30；0 或缺失回落默认值。
	LatencyWindowMinutes int `json:"group_scheduling_election_latency_window_minutes"`
	// LatencyMinSamples 是信任窗口均值所需的最少成功样本数，默认 3；不足则回退单值。
	LatencyMinSamples int `json:"group_scheduling_election_latency_min_samples"`
	// KeepHealthyIncumbentGroupIDs 存「启用在任者健康锁定」的分组 ID（opt-in，与 DisabledGroupIDs 相反极性）。
	// 列表里的分组：只要它「当前开启调度的账号」测试都正常（且没有开着却失败的），就锁定——保留这些在任账号、
	// 不做任何择优与换人，直接跳过。空列表=所有分组都正常择优（默认），新增分组默认不锁定，安全。
	// 只在在任者健康时锁定：一旦有开着的账号测试失败，该分组仍走正常择优交给失败闸门处理，绝不锁死一个坏分组。
	KeepHealthyIncumbentGroupIDs []int64 `json:"group_scheduling_election_keep_healthy_incumbent_group_ids"`
	// RequiredModelsByGroup 是「分组必须能服务的模型」清单：group_id → 模型名列表（空=该组无强制要求）。
	// 择优选出赢家后对每个必需模型做覆盖兜底：赢家里若没有账号支持它，就在组内**健康**账号中补选综合分最高的
	// 支持者 union-enable（哪怕它本不是最优）——保证换人不会把某个必需模型换没了。必需模型是硬底线，
	// 连「在任者健康锁定」的分组也会补齐。若支持该模型的账号当前全部测试失败/根本没有，则不硬留失败账号
	// （让正常赢家生效、允许切到能用的账号），只记一条告警待人工恢复。判定「是否支持」复用运行时
	// account.IsModelSupported，与网关逐请求过滤同源。
	RequiredModelsByGroup map[int64][]string `json:"group_scheduling_election_required_models"`
}

// SupplierGroupSchedulingElectionMember 是仓储层返回的一条"分组×账号"成员行。
type SupplierGroupSchedulingElectionMember struct {
	GroupID        int64
	GroupName      string
	AccountID      int64
	AccountName    string
	Platform       string
	Schedulable    bool
	LastTestStatus string
	HealthyCount   int
	LastTestedAt   time.Time
	// LastTestLatencyMs 是最近一次成功测试的耗时（毫秒），0 表示没有可用数据。
	// 它与 LastTestStatus / LastTestedAt 同源（都由账号测试写入），失败时不覆盖旧值，
	// 因此"筛进 success 的账号"通常都带得上耗时。
	LastTestLatencyMs int64
	// FailedCount 是本任务自己累计的"连续失败轮次"，来自 accounts.extra。
	// 它不来自测试系统：last_test_status 只有账号测试才更新，而没有任何自动任务会跑账号测试，
	// 所以"连续失败几次"必须由本任务按自己的执行周期来数。
	FailedCount int
	// 下面三项来自 supplier_account_health_history 在时间窗内的聚合，用来把延迟从"单次采样"
	// 升级为"最近一段时间的平均"，并让被剔除的失败样本通过成功率重新计入代价。
	//   AvgLatencyMs        —— 窗口内成功样本（healthy/slow 且 latency>0）的平均延迟，0 表示窗口无数据。
	//   LatencySuccessCount —— 窗口内成功样本数，用于判断均值是否可信（不足 MinSamples 则回退单值）。
	//   LatencyTotalCount   —— 窗口内总样本数（含 failed），成功率 = 成功数 / 总数，不受各账号巡检频率差异影响。
	AvgLatencyMs        int64
	LatencySuccessCount int
	LatencyTotalCount   int
	// AccountType / ModelMapping / Extra 只服务于「必需模型覆盖」：用它们在择优里重建一个最小 Account，
	// faithfully 复用 account.IsModelSupported 判断该账号是否支持某模型（与网关逐请求过滤同源）。
	// ModelMapping 只取 credentials.model_mapping 子对象（不拉 token）；Extra 里带 openai_passthrough 等
	// 影响模型判定的开关。空 mapping = 支持所有模型，与运行时口径一致。
	AccountType  string
	ModelMapping map[string]any
	Extra        map[string]any
}

type SupplierGroupSchedulingElectionRepository interface {
	// latencyWindowMinutes 决定「最近平均延迟」聚合的时间窗；<=0 时由仓储回落到默认窗口。
	ListGroupSchedulingElectionMembers(ctx context.Context, latencyWindowMinutes int) ([]SupplierGroupSchedulingElectionMember, error)
}

type supplierGroupSchedulingElectionAccountStore interface {
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
	// UpdateExtra 用于回写「连续失败轮次」计数。
	// 给本接口加方法不会改变 ProvideSupplierGroupSchedulingElectionService 的参数列表，
	// 因此 Wire 生成的代码无需重新生成（AccountRepository 本来就有这个方法）。
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type SupplierGroupSchedulingElectionRunner interface {
	Run(ctx context.Context, config SupplierGroupSchedulingElectionConfig, now time.Time) (SupplierGroupSchedulingElectionResult, error)
}

type SupplierGroupSchedulingElectionService struct {
	repository   SupplierGroupSchedulingElectionRepository
	accountStore supplierGroupSchedulingElectionAccountStore
}

func NewSupplierGroupSchedulingElectionService(repository SupplierGroupSchedulingElectionRepository, accountStore supplierGroupSchedulingElectionAccountStore) *SupplierGroupSchedulingElectionService {
	return &SupplierGroupSchedulingElectionService{repository: repository, accountStore: accountStore}
}

// SupplierGroupSchedulingElectionGroupDetail 汇总单个分组的裁决结果。
type SupplierGroupSchedulingElectionGroupDetail struct {
	GroupID       int64   `json:"group_id"`
	GroupName     string  `json:"group_name"`
	MemberCount   int     `json:"member_count"`
	SuccessCount  int     `json:"success_count"`
	FailedCount   int     `json:"failed_count"`
	UntestedCount int     `json:"untested_count"`
	WinnerCount   int     `json:"winner_count"`
	WinnerIDs     []int64 `json:"winner_ids,omitempty"`
}

// SupplierGroupSchedulingElectionAccountItem 是发生调度变更（或写库失败）的账号明细。
type SupplierGroupSchedulingElectionAccountItem struct {
	AccountID    int64  `json:"account_id"`
	AccountName  string `json:"account_name"`
	Platform     string `json:"platform,omitempty"`
	TestStatus   string `json:"test_status,omitempty"`
	HealthyCount int    `json:"healthy_count"`
	// LatencyMs 是最近一次成功测试的耗时（毫秒），只用于让管理员看懂"为什么是它当选"：
	// 综合分里用时的权重是可配的，明细里却看不到用时，赢家就没法被解释。
	// 0 表示没有可用数据，此时省略该字段 —— 旧的运行记录里没有它，前端按"—"处理。
	LatencyMs         int64   `json:"latency_ms,omitempty"`
	SchedulableBefore bool    `json:"schedulable_before"`
	SchedulableAfter  bool    `json:"schedulable_after"`
	Action            string  `json:"action"`
	Reason            string  `json:"reason,omitempty"`
	GroupIDs          []int64 `json:"group_ids,omitempty"`
	ErrorMessage      string  `json:"error_message,omitempty"`
}

type SupplierGroupSchedulingElectionResult struct {
	TopN             int `json:"top_n"`
	GroupCount       int `json:"group_count"`
	AccountCount     int `json:"account_count"`
	EnabledCount     int `json:"enabled_count"`
	DisabledCount    int `json:"disabled_count"`
	UnchangedCount   int `json:"unchanged_count"`
	SkippedCount     int `json:"skipped_count"`
	FailedWriteCount int `json:"failed_write_count"`
	// PendingCount / KeptCount 是被闸门拦住、本轮"故意没关"的账号数。
	// 单独计数而不是只留在明细里，是因为运行列表默认只显示一行摘要——
	// 「连续失败待观察」和「分组只剩它、需人工确认」这两种状态必须能被一眼看到。
	PendingCount int                                          `json:"pending_count"`
	KeptCount    int                                          `json:"kept_count"`
	// RequiredModelUncoveredCount / RequiredModelWarnings 记录「分组配置了必需模型、但当前没有健康账号能提供它」
	// 的情况。这不是失败（本轮仍让能用的账号生效），但必须被看见——否则某个模型静默断供、无人知晓。
	RequiredModelUncoveredCount int                                                    `json:"required_model_uncovered_count"`
	RequiredModelWarnings       []SupplierGroupSchedulingElectionRequiredModelWarning `json:"required_model_warnings,omitempty"`
	Groups                      []SupplierGroupSchedulingElectionGroupDetail          `json:"groups"`
	Items                       []SupplierGroupSchedulingElectionAccountItem          `json:"items"`
}

// SupplierGroupSchedulingElectionRequiredModelWarning 是一条「某分组的必需模型当前无健康账号可提供」的告警。
// 触发条件：分组配置了必需模型 M，赢家里没有账号支持它，且组内支持 M 的账号当前测试都失败/根本没有。
// 此时不硬留失败账号（允许切到能用的健康账号），只发这条告警并携带失败支持者 ID，方便人工定位恢复。
type SupplierGroupSchedulingElectionRequiredModelWarning struct {
	GroupID          int64   `json:"group_id"`
	GroupName        string  `json:"group_name,omitempty"`
	Model            string  `json:"model"`
	FailedAccountIDs []int64 `json:"failed_account_ids,omitempty"`
}

// 调度切换日志的方向。明细里能靠 schedulable_before/after 自己推，但列表要能按方向筛，
// 推出来的值必须有一个稳定命名，否则前后端各推一份迟早对不上。
const (
	SupplierGroupSchedulingElectionChangeDirectionEnabled  = "enabled"
	SupplierGroupSchedulingElectionChangeDirectionDisabled = "disabled"
)

// SupplierGroupSchedulingElectionChangeLog 是一条「某账号的调度开关真的被拨动了」的记录。
// 与运行明细的区别：运行明细是「一次任务执行」视角（含未变更、写库失败），
// 这里是「一次开关变化」视角，**只保留 schedulable_before <> schedulable_after 的条目**。
type SupplierGroupSchedulingElectionChangeLog struct {
	RunID       int64     `json:"run_id"`
	RunStatus   string    `json:"run_status"`
	ChangedAt   time.Time `json:"changed_at"`
	AccountID   int64     `json:"account_id"`
	AccountName string    `json:"account_name"`
	Platform    string    `json:"platform,omitempty"`
	TestStatus  string    `json:"test_status,omitempty"`
	// HealthyCount / LatencyMs 是切换发生那一刻的竞选依据，
	// 事后回看「为什么当时选了它而不是另一个」全靠这两个值。
	HealthyCount      int      `json:"healthy_count"`
	LatencyMs         int64    `json:"latency_ms,omitempty"`
	SchedulableBefore bool     `json:"schedulable_before"`
	SchedulableAfter  bool     `json:"schedulable_after"`
	Direction         string   `json:"direction"`
	Action            string   `json:"action"`
	Reason            string   `json:"reason,omitempty"`
	ErrorMessage      string   `json:"error_message,omitempty"`
	GroupIDs          []int64  `json:"group_ids,omitempty"`
	GroupNames        []string `json:"group_names,omitempty"`
}

type SupplierGroupSchedulingElectionChangeLogListParams struct {
	GroupID   int64
	AccountID int64
	// RunID 锁定到某一次择优运行（任务批次），只看这批被拨动的账号；0 = 不按批次筛。
	RunID int64
	// Search 按账号名模糊匹配（分组管理页从某个分组进入时不带它，任务中心页全局看时用）。
	Search      string
	Direction   string
	StartedFrom *time.Time
	StartedTo   *time.Time
	Page        int
	PageSize    int
}

type SupplierGroupSchedulingElectionChangeLogListResult struct {
	Items    []SupplierGroupSchedulingElectionChangeLog `json:"items"`
	Total    int64                                      `json:"total"`
	Page     int                                        `json:"page"`
	PageSize int                                        `json:"page_size"`
}

// SupplierGroupSchedulingElectionChangeLogStore 由已注入的 dataRepo 断言得到。
// 不进 SupplierProviderDataRepository 大接口，是为了让"谁依赖这份日志"保持可见：
// 只有自动化任务中心会调它，不该让所有持有 dataRepo 的地方都被动实现一遍。
type SupplierGroupSchedulingElectionChangeLogStore interface {
	ListGroupSchedulingElectionChangeLogs(ctx context.Context, params SupplierGroupSchedulingElectionChangeLogListParams) (SupplierGroupSchedulingElectionChangeLogListResult, error)
}

// supplierGroupElectionAccount 是把同一账号在多个分组里的成员行聚合后的视图。
type supplierGroupElectionAccount struct {
	id                int64
	name              string
	platform          string
	schedulableBefore bool
	testStatus        string
	healthyCount      int
	lastTestedAt      time.Time
	// latencyMs 是最近一次成功测试的耗时。同一账号跨多个分组时每行取值相同（都来自 accounts.extra），
	// 但只在当前为 0 时补填，避免"某一行恰好没读到"就把已有值覆盖掉。
	latencyMs   int64
	failedCount int
	groupIDs    []int64
	// winner 表示该账号在其所属的任意分组里被选为最优（union-enable）。
	winner bool
	// noAlternative 表示：关掉这个账号之后，它所属的分组里**至少有一个**将没有任何成功账号可开启。
	// 命中就保留它的调度（哪怕它当前是 failed）—— 把分组关成空组比留着一个坏账号更糟，
	// 前者是明确故障，后者至少还有恢复的可能。
	noAlternative bool
	// hold 非空表示这个账号本轮"故意没关"，取值见 supplierGroupElectionHold* 常量。
	// 非空即 notable：只靠汇总数字的话，运维看到"关闭 0 个"会以为任务没跑，
	// 而实际恰恰是最需要被看见的情况——有账号连续失败但被闸门拦住了。
	hold string
}

func (s *SupplierGroupSchedulingElectionService) Run(ctx context.Context, config SupplierGroupSchedulingElectionConfig, now time.Time) (SupplierGroupSchedulingElectionResult, error) {
	config = normalizeSupplierGroupSchedulingElectionConfig(config)
	if s == nil || s.repository == nil || s.accountStore == nil {
		return SupplierGroupSchedulingElectionResult{}, errors.New("分组择优调度依赖未初始化")
	}
	members, err := s.repository.ListGroupSchedulingElectionMembers(ctx, config.LatencyWindowMinutes)
	if err != nil {
		return SupplierGroupSchedulingElectionResult{}, err
	}

	disabledGroups := make(map[int64]struct{}, len(config.DisabledGroupIDs))
	for _, groupID := range config.DisabledGroupIDs {
		disabledGroups[groupID] = struct{}{}
	}
	keepHealthyGroups := make(map[int64]struct{}, len(config.KeepHealthyIncumbentGroupIDs))
	for _, groupID := range config.KeepHealthyIncumbentGroupIDs {
		keepHealthyGroups[groupID] = struct{}{}
	}

	// 1) 按分组归拢成员，同时聚合账号级视图（同一账号可能横跨多个分组）。
	membersByGroup := make(map[int64][]SupplierGroupSchedulingElectionMember)
	groupNames := make(map[int64]string)
	accounts := make(map[int64]*supplierGroupElectionAccount)
	groupOrder := make([]int64, 0)
	for _, member := range members {
		if _, skip := disabledGroups[member.GroupID]; skip {
			continue
		}
		if _, seen := membersByGroup[member.GroupID]; !seen {
			groupOrder = append(groupOrder, member.GroupID)
		}
		membersByGroup[member.GroupID] = append(membersByGroup[member.GroupID], member)
		groupNames[member.GroupID] = member.GroupName

		// 显示用延迟：窗口成功样本足够就用平均，否则回退最近单值——同一账号跨分组取值一致。
		baseLatency, _ := supplierGroupSchedulingElectionLatency(member, config.LatencyMinSamples)
		account := accounts[member.AccountID]
		if account == nil {
			account = &supplierGroupElectionAccount{
				id:                member.AccountID,
				name:              member.AccountName,
				platform:          member.Platform,
				schedulableBefore: member.Schedulable,
				testStatus:        strings.TrimSpace(member.LastTestStatus),
				healthyCount:      member.HealthyCount,
				lastTestedAt:      member.LastTestedAt,
				latencyMs:         baseLatency,
				failedCount:       member.FailedCount,
			}
			accounts[member.AccountID] = account
		}
		if account.latencyMs == 0 {
			account.latencyMs = baseLatency
		}
		account.groupIDs = append(account.groupIDs, member.GroupID)
	}
	sort.Slice(groupOrder, func(i, j int) bool { return groupOrder[i] < groupOrder[j] })

	result := SupplierGroupSchedulingElectionResult{
		TopN:         config.TopN,
		GroupCount:   len(groupOrder),
		AccountCount: len(accounts),
		Groups:       make([]SupplierGroupSchedulingElectionGroupDetail, 0, len(groupOrder)),
		Items:        make([]SupplierGroupSchedulingElectionAccountItem, 0),
	}

	// 2) 逐分组选出最优（前 N），并集式标记赢家（union-enable）。
	for _, groupID := range groupOrder {
		detail := SupplierGroupSchedulingElectionGroupDetail{
			GroupID:   groupID,
			GroupName: groupNames[groupID],
		}
		groupMembers := membersByGroup[groupID]
		// recordWinner 统一「标记账号赢家（union，跨组累积）+ 记入本组明细」。
		// 明细里始终追加（同一账号跨多组时每组都应列出），故不因 account.winner 已置而跳过追加。
		recordWinner := func(accountID int64) {
			if account := accounts[accountID]; account != nil {
				account.winner = true
			}
			detail.WinnerCount++
			detail.WinnerIDs = append(detail.WinnerIDs, accountID)
		}

		successMembers := make([]SupplierGroupSchedulingElectionMember, 0)
		for _, member := range groupMembers {
			detail.MemberCount++
			switch strings.TrimSpace(member.LastTestStatus) {
			case SupplierGroupSchedulingElectionTestStatusSuccess:
				detail.SuccessCount++
				successMembers = append(successMembers, member)
			case SupplierGroupSchedulingElectionTestStatusFailed:
				detail.FailedCount++
			default:
				detail.UntestedCount++
			}
		}
		// 闸门一：一个成功账号都没有的分组里，失败账号一个都不能关——
		// 关掉最后一个等于把这个分组关成空组，落到调度上就是请求全量失败；
		// 留着一个坏账号至少还有恢复的可能，所以交给人工确认而不是自动关。
		if len(successMembers) == 0 {
			for _, member := range groupMembers {
				if strings.TrimSpace(member.LastTestStatus) != SupplierGroupSchedulingElectionTestStatusFailed {
					continue
				}
				if account := accounts[member.AccountID]; account != nil {
					account.noAlternative = true
				}
			}
		}

		// 综合分依赖组内上下文（同平台内谁最快），必须按组算一次——锁定路径、正常择优、
		// 必需模型补选三处都复用它，保证「谁更优」的口径一致。
		electionScores := supplierGroupSchedulingElectionScores(successMembers, config.CountWeight, config.LatencyWeight, config.LatencyMinSamples, config.SwitchMargin)

		// 在任者健康锁定：仅对被 opt-in 的分组生效，且该组「当前开着的账号」测试都正常（且没有开着却失败的），
		// 就保留这些在任账号、跳过择优与换人。只在健康时锁定——有开着的账号失败仍走下面的正常择优，
		// 交给失败闸门处理，绝不把一个坏分组锁死。
		locked := false
		if _, keepHealthy := keepHealthyGroups[groupID]; keepHealthy {
			scheduledHealthy := make([]int64, 0)
			scheduledFailed := false
			for _, member := range groupMembers {
				if !member.Schedulable {
					continue
				}
				switch strings.TrimSpace(member.LastTestStatus) {
				case SupplierGroupSchedulingElectionTestStatusSuccess:
					scheduledHealthy = append(scheduledHealthy, member.AccountID)
				case SupplierGroupSchedulingElectionTestStatusFailed:
					scheduledFailed = true
				}
			}
			if len(scheduledHealthy) > 0 && !scheduledFailed {
				for _, accountID := range scheduledHealthy {
					recordWinner(accountID)
				}
				locked = true
			}
		}

		// 正常择优：锁定未生效时才做。综合分排序后取前 N。
		if !locked {
			sort.SliceStable(successMembers, func(i, j int) bool {
				return supplierGroupSchedulingElectionMemberLess(successMembers[i], successMembers[j], electionScores)
			})
			winners := config.TopN
			if winners > len(successMembers) {
				winners = len(successMembers)
			}
			for index := 0; index < winners; index++ {
				recordWinner(successMembers[index].AccountID)
			}
		}

		// 必需模型覆盖兜底：赢家没覆盖的必需模型，补选一个健康支持者 union-enable；全失败则告警。
		// 必需模型是硬底线，锁定组也执行——它只增开支持者、不动在任赢家，不破坏锁定语义。
		applySupplierGroupRequiredModelCoverage(
			groupID, groupNames[groupID], config.RequiredModelsByGroup[groupID],
			groupMembers, successMembers, electionScores, detail.WinnerIDs, recordWinner, &result,
		)

		result.Groups = append(result.Groups, detail)
	}

	// 3) 账号级最终裁决：失败先过两道闸门；成功赢家开、成功落选关；未测过跳过。
	accountIDs := make([]int64, 0, len(accounts))
	for accountID := range accounts {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i] < accountIDs[j] })

	for _, accountID := range accountIDs {
		account := accounts[accountID]
		item := SupplierGroupSchedulingElectionAccountItem{
			AccountID:         account.id,
			AccountName:       account.name,
			Platform:          account.platform,
			TestStatus:        account.testStatus,
			HealthyCount:      account.healthyCount,
			LatencyMs:         account.latencyMs,
			SchedulableBefore: account.schedulableBefore,
			SchedulableAfter:  account.schedulableBefore,
			Action:            SupplierGroupSchedulingElectionActionNone,
			GroupIDs:          account.groupIDs,
		}

		desiredSchedulable, action, reason := supplierGroupSchedulingElectionDecide(account, config.FailureThreshold)
		item.SchedulableAfter = desiredSchedulable
		item.Action = action
		item.Reason = reason

		if account.testStatus != SupplierGroupSchedulingElectionTestStatusSuccess &&
			account.testStatus != SupplierGroupSchedulingElectionTestStatusFailed {
			result.SkippedCount++
			// 未测过的账号不产生调度变更，也不计入明细，避免噪声。
			continue
		}

		// 连续失败轮次必须在本轮就落地：下一轮判定读的就是这个数。
		if err := s.persistSupplierGroupElectionFailedCount(ctx, account, config.FailureThreshold); err != nil {
			// 计数只是辅助账本，回写失败不该改变本轮的调度裁决；
			// 也不计入 FailedWriteCount（那一项是"调度状态写库失败"，会影响任务整体状态），
			// 但必须体现在明细里，否则这个失败就真的没人看得见。
			item.ErrorMessage = fmt.Sprintf("连续失败计数回写失败：%v", err)
		}

		if item.SchedulableAfter == item.SchedulableBefore {
			result.UnchangedCount++
			switch account.hold {
			case supplierGroupElectionHoldPending:
				result.PendingCount++
			case supplierGroupElectionHoldNoAlternative:
				result.KeptCount++
			}
			if account.hold != "" || item.ErrorMessage != "" {
				result.Items = append(result.Items, item)
			}
			continue
		}

		if err := s.accountStore.SetSchedulable(ctx, account.id, item.SchedulableAfter); err != nil {
			item.SchedulableAfter = item.SchedulableBefore
			item.Action = SupplierGroupSchedulingElectionActionNone
			item.Reason = SupplierGroupSchedulingElectionReasonWriteFailed
			item.ErrorMessage = err.Error()
			result.FailedWriteCount++
			result.Items = append(result.Items, item)
			continue
		}

		switch item.Action {
		case SupplierGroupSchedulingElectionActionEnabled:
			result.EnabledCount++
		case SupplierGroupSchedulingElectionActionDisabled:
			result.DisabledCount++
		}
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// supplierGroupSchedulingElectionDecide 给出账号的目标调度状态。
// 规则（复用 last_test_status 与 healthy_count，不重测）：
//   - failed  → 两道闸门都过了才关：先确认分组不会被关成空组，再确认连续失败已达阈值
//   - success → 在任一所属分组是最优则开；否则关（落选）
//   - 其它（未测过） → 保持原状
//
// 为什么 failed 不再是"无条件关"：last_test_status 只有账号测试才写，而一次网络抖动、
// 一次上游限流都会写 failed；同时没有任何自动任务会跑账号测试，所以一旦关掉很可能再也没人
// 把它翻回 success。一次失败就关 = 用一次采样赌一个账号的生死，代价远大于收益。
func supplierGroupSchedulingElectionDecide(account *supplierGroupElectionAccount, failureThreshold int) (bool, string, string) {
	switch account.testStatus {
	case SupplierGroupSchedulingElectionTestStatusFailed:
		// 闸门一（保底）：关掉它之后分组里再没有可开启的成功账号 —— 绝不关。
		// 这条不看失败次数：没有备选账号时，关与不关的区别是"全量失败"和"可能慢但能用"。
		if account.noAlternative {
			account.hold = supplierGroupElectionHoldNoAlternative
			return account.schedulableBefore, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonKeepLastOne
		}
		// 闸门二（防抖）：连续失败累计到阈值才关，让单次抖动有翻盘的机会。
		if pending := account.failedCount + 1; pending < failureThreshold {
			account.hold = supplierGroupElectionHoldPending
			return account.schedulableBefore, SupplierGroupSchedulingElectionActionNone,
				fmt.Sprintf(SupplierGroupSchedulingElectionReasonFailedPendingFmt, pending, failureThreshold)
		}
		if account.schedulableBefore {
			return false, SupplierGroupSchedulingElectionActionDisabled, SupplierGroupSchedulingElectionReasonFailed
		}
		return false, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonFailed
	case SupplierGroupSchedulingElectionTestStatusSuccess:
		if account.winner {
			if !account.schedulableBefore {
				return true, SupplierGroupSchedulingElectionActionEnabled, SupplierGroupSchedulingElectionReasonElected
			}
			return true, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonElected
		}
		if account.schedulableBefore {
			return false, SupplierGroupSchedulingElectionActionDisabled, SupplierGroupSchedulingElectionReasonNotElected
		}
		return false, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonNotElected
	default:
		return account.schedulableBefore, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonUntested
	}
}

// persistSupplierGroupElectionFailedCount 回写本任务自己累计的「连续失败轮次」。
//
// 为什么不直接读健康守护的 failure_count：两者的检测周期与判据都不同——
// 本任务裁决用的是 last_test_status（只有账号测试才写），而失败计数必须与"本任务的判定轮次"
// 对齐，否则阈值就没有意义（比如健康守护一轮就跑十次，两次就到阈值了）。
func (s *SupplierGroupSchedulingElectionService) persistSupplierGroupElectionFailedCount(ctx context.Context, account *supplierGroupElectionAccount, failureThreshold int) error {
	next := 0
	if account.testStatus == SupplierGroupSchedulingElectionTestStatusFailed {
		next = account.failedCount + 1
		// 封顶到阈值：再往上加不改变任何判定结果，只会让 extra 无限膨胀。
		if next > failureThreshold {
			next = failureThreshold
		}
	}
	if next == account.failedCount {
		return nil
	}
	if err := s.accountStore.UpdateExtra(ctx, account.id, map[string]any{supplierGroupElectionFailedCountExtraKey: next}); err != nil {
		return err
	}
	account.failedCount = next
	return nil
}

// supplierGroupSchedulingElectionScores 给同一分组内的候选账号算综合分，越大越优。
//
// 综合分 = CountWeight × 次数分 + LatencyWeight × 用时分，两项各自先归一到 [0,1]。
// 次数是"资历"、用时是"当前表现"，量纲不同，只有归一后加权才能合成一个比较依据；
// 也正因归一了，两个权重的比值就是两者的相对话语权，配置起来不需要理解量纲。
//
// 用时只在**同平台**账号之间比较：不同平台的基线延迟可以相差数倍，
// 直接比毫秒会让低延迟平台的账号长期垄断赢家，那不是择优。
// 凡无法比较的情形（跨平台、没数据、同平台只有一个样本、用时全都一样）
// 一律给中性分 0.5，让这一项不影响相对顺序，胜负交回次数决定。
// supplierGroupSchedulingElectionLatency 返回该成员用于「显示」与「评分」的两个延迟值。
//
//	base    —— 窗口内成功样本数达到 minSamples 时取窗口平均延迟，否则回退到最近一次单值
//	           （accounts.extra 里的 last_test_latency_ms）。这是真实延迟，用于日志显示。
//	scoring —— 在 base 之上按成功率惩罚：base / max(成功率, floor)。只算成功样本会掩盖
//	           「偶尔快一下、实则一直在失败」的账号，用成功率把被剔除的失败重新计入代价。
//	           惩罚只在信任窗口均值（有窗口总样本数）时生效；回退单值时没有成功率可用，不惩罚。
func supplierGroupSchedulingElectionLatency(member SupplierGroupSchedulingElectionMember, minSamples int) (base int64, scoring int64) {
	trusted := member.LatencySuccessCount >= minSamples && member.AvgLatencyMs > 0 && member.LatencyTotalCount > 0
	if !trusted {
		return member.LastTestLatencyMs, member.LastTestLatencyMs
	}
	base = member.AvgLatencyMs
	successRate := float64(member.LatencySuccessCount) / float64(member.LatencyTotalCount)
	if successRate < supplierGroupElectionSuccessRateFloor {
		successRate = supplierGroupElectionSuccessRateFloor
	}
	scoring = int64(math.Round(float64(base) / successRate))
	return base, scoring
}

func supplierGroupSchedulingElectionScores(members []SupplierGroupSchedulingElectionMember, countWeight, latencyWeight float64, latencyMinSamples int, switchMargin float64) map[int64]float64 {
	// 迟滞死区做在延迟上而不是综合分上：组内 min-max 归一化会把任意大小的延迟差放大到满量程
	// （两个候选时分差恒为 latencyWeight），综合分层面的死区因此形同虚设。改为把「在任者(已开启)」
	// 的评分延迟按 (1-switchMargin) 折算，让它显得更快，于是挑战者必须在延迟上快出 switchMargin 比例
	// 才能在归一化后反超——这才是能压住两个正常账号反复对拍的真·死区。折算后仍留 5% 地板防止归零。
	marginFactor := 1.0 - switchMargin
	if marginFactor < 0.05 {
		marginFactor = 0.05
	}
	// 评分用的有效延迟：窗口平均（可信时）+ 成功率惩罚，否则回退单值；在任者再叠加迟滞折算。
	effectiveLatency := make(map[int64]int64, len(members))
	for _, member := range members {
		_, scoring := supplierGroupSchedulingElectionLatency(member, latencyMinSamples)
		if member.Schedulable && scoring > 0 {
			scoring = int64(math.Round(float64(scoring) * marginFactor))
		}
		effectiveLatency[member.AccountID] = scoring
	}

	minLatency := make(map[string]int64)
	maxLatency := make(map[string]int64)
	latencySamples := make(map[string]int)
	for _, member := range members {
		latency := effectiveLatency[member.AccountID]
		if latency <= 0 {
			continue
		}
		latencySamples[member.Platform]++
		if seen, ok := minLatency[member.Platform]; !ok || latency < seen {
			minLatency[member.Platform] = latency
		}
		if seen, ok := maxLatency[member.Platform]; !ok || latency > seen {
			maxLatency[member.Platform] = latency
		}
	}

	countCap := float64(SupplierGroupSchedulingElectionCountScoreCap)
	scores := make(map[int64]float64, len(members))
	for _, member := range members {
		normalizedCount := float64(member.HealthyCount) / countCap
		if normalizedCount > 1 {
			normalizedCount = 1
		}
		latency := effectiveLatency[member.AccountID]
		normalizedLatency := 0.5
		// 至少两个样本才谈得上"谁更快"；极差为 0 说明大家一样快，同样比不出高下。
		if latency > 0 && latencySamples[member.Platform] >= 2 {
			if span := maxLatency[member.Platform] - minLatency[member.Platform]; span > 0 {
				normalizedLatency = float64(maxLatency[member.Platform]-latency) / float64(span)
			}
		}
		scores[member.AccountID] = countWeight*normalizedCount + latencyWeight*normalizedLatency
	}
	return scores
}

// supplierGroupSchedulingElectionMemberLess 定义"更优"排序：综合分高者优先。
// 迟滞死区不在这里做（会被组内 min-max 归一化放大而失效，两候选时分差恒为 latencyWeight），
// 而是在 supplierGroupSchedulingElectionScores 里对在任者的延迟按比例折算实现——见那里的注释。
// 综合分并列时优先保留「当前已开启调度」的账号（黏性），再按最近测试时间、账号 ID 兜底，保证确定性。
func supplierGroupSchedulingElectionMemberLess(a, b SupplierGroupSchedulingElectionMember, scores map[int64]float64) bool {
	scoreA, scoreB := scores[a.AccountID], scores[b.AccountID]
	if math.Abs(scoreA-scoreB) > SupplierGroupSchedulingElectionScoreEpsilon {
		return scoreA > scoreB
	}
	if a.Schedulable != b.Schedulable {
		return a.Schedulable
	}
	if !a.LastTestedAt.Equal(b.LastTestedAt) {
		return a.LastTestedAt.After(b.LastTestedAt)
	}
	return a.AccountID < b.AccountID
}

// applySupplierGroupRequiredModelCoverage 保证分组配置的每个「必需模型」都至少有一个赢家能提供。
// 覆盖判定与网关运行时同源（account.IsModelSupported）：赢家里已有账号支持该模型则跳过；否则在组内
// **健康(success)**账号里补选综合分最高的支持者 union-enable（哪怕它本不是 TopN 最优）。若支持它的账号
// 当前全部失败/根本没有，就不硬留失败账号（让已选出的健康赢家生效、允许切到能用的账号），只记一条告警、
// 携带失败支持者 ID 待人工恢复。只增开支持者、不动已选赢家，因此对「在任者健康锁定」的分组同样安全。
func applySupplierGroupRequiredModelCoverage(
	groupID int64,
	groupName string,
	requiredModels []string,
	groupMembers []SupplierGroupSchedulingElectionMember,
	successMembers []SupplierGroupSchedulingElectionMember,
	electionScores map[int64]float64,
	initialWinnerIDs []int64,
	recordWinner func(int64),
	result *SupplierGroupSchedulingElectionResult,
) {
	if len(requiredModels) == 0 {
		return
	}
	memberByID := make(map[int64]SupplierGroupSchedulingElectionMember, len(groupMembers))
	for _, member := range groupMembers {
		memberByID[member.AccountID] = member
	}
	winnerSet := make(map[int64]struct{}, len(initialWinnerIDs))
	for _, id := range initialWinnerIDs {
		winnerSet[id] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requiredModels))
	for _, model := range requiredModels {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, dup := seen[model]; dup {
			continue
		}
		seen[model] = struct{}{}

		// 已有赢家支持该模型？
		covered := false
		for id := range winnerSet {
			if member, ok := memberByID[id]; ok && supplierGroupElectionMemberSupportsModel(member, model) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}

		// 在健康账号里补选综合分最高的支持者（跳过已是赢家的）。
		var best *SupplierGroupSchedulingElectionMember
		for i := range successMembers {
			candidate := successMembers[i]
			if _, isWinner := winnerSet[candidate.AccountID]; isWinner {
				continue
			}
			if !supplierGroupElectionMemberSupportsModel(candidate, model) {
				continue
			}
			if best == nil || supplierGroupSchedulingElectionMemberLess(candidate, *best, electionScores) {
				picked := candidate
				best = &picked
			}
		}
		if best != nil {
			recordWinner(best.AccountID)
			winnerSet[best.AccountID] = struct{}{}
			continue
		}

		// 没有健康支持者：收集当前失败的支持者 ID（方便人工恢复），记一条告警。
		failedIDs := make([]int64, 0)
		for _, member := range groupMembers {
			if strings.TrimSpace(member.LastTestStatus) != SupplierGroupSchedulingElectionTestStatusFailed {
				continue
			}
			if supplierGroupElectionMemberSupportsModel(member, model) {
				failedIDs = append(failedIDs, member.AccountID)
			}
		}
		result.RequiredModelUncoveredCount++
		result.RequiredModelWarnings = append(result.RequiredModelWarnings, SupplierGroupSchedulingElectionRequiredModelWarning{
			GroupID:          groupID,
			GroupName:        groupName,
			Model:            model,
			FailedAccountIDs: failedIDs,
		})
	}
}

// supplierGroupElectionMemberSupportsModel 复用运行时 account.IsModelSupported 判断成员是否支持某模型。
// 重建一个最小 Account（platform/type/model_mapping/extra）——这些正是 IsModelSupported 读取的字段，
// 因此判定与网关逐请求过滤同源。空 mapping = 支持所有模型。
func supplierGroupElectionMemberSupportsModel(member SupplierGroupSchedulingElectionMember, model string) bool {
	account := &Account{
		Platform: member.Platform,
		Type:     member.AccountType,
		Extra:    member.Extra,
	}
	if member.ModelMapping != nil {
		account.Credentials = map[string]any{"model_mapping": member.ModelMapping}
	}
	return account.IsModelSupported(model)
}

func normalizeSupplierGroupSchedulingElectionConfig(config SupplierGroupSchedulingElectionConfig) SupplierGroupSchedulingElectionConfig {
	if config.TopN <= 0 {
		config.TopN = DefaultSupplierGroupSchedulingElectionTopN
	}
	if config.TopN > MaxSupplierGroupSchedulingElectionTopN {
		config.TopN = MaxSupplierGroupSchedulingElectionTopN
	}
	config.DisabledGroupIDs = uniquePositiveInt64s(config.DisabledGroupIDs)
	sort.Slice(config.DisabledGroupIDs, func(i, j int) bool {
		return config.DisabledGroupIDs[i] < config.DisabledGroupIDs[j]
	})
	config.KeepHealthyIncumbentGroupIDs = uniquePositiveInt64s(config.KeepHealthyIncumbentGroupIDs)
	sort.Slice(config.KeepHealthyIncumbentGroupIDs, func(i, j int) bool {
		return config.KeepHealthyIncumbentGroupIDs[i] < config.KeepHealthyIncumbentGroupIDs[j]
	})
	// 旧的 config_json 里没有权重字段，反序列化后是 0；而 JSON 无法区分"字段缺失"与"显式填 0"，
	// 所以 0 一律按"未配置"回落到默认值 —— 否则老任务升级后会静默把用时权重当成 0，
	// 新功能等于没启用。想压低用时的影响请填一个小的正数（如 0.1），不要把 0 当开关用。
	if config.CountWeight <= 0 {
		config.CountWeight = DefaultSupplierGroupSchedulingElectionCountWeight
	}
	if config.LatencyWeight <= 0 {
		config.LatencyWeight = DefaultSupplierGroupSchedulingElectionLatencyWeight
	}
	if config.CountWeight > MaxSupplierGroupSchedulingElectionWeight {
		config.CountWeight = MaxSupplierGroupSchedulingElectionWeight
	}
	if config.LatencyWeight > MaxSupplierGroupSchedulingElectionWeight {
		config.LatencyWeight = MaxSupplierGroupSchedulingElectionWeight
	}
	// 与权重同理：旧 config_json 里没有这个字段，反序列化后是 0，一律按默认值处理。
	// 想退回「一次失败立刻关」请显式配 1，而不是把 0 当开关用。
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultSupplierGroupSchedulingElectionFailureThreshold
	}
	if config.FailureThreshold > MaxSupplierGroupSchedulingElectionFailureThreshold {
		config.FailureThreshold = MaxSupplierGroupSchedulingElectionFailureThreshold
	}
	// 迟滞死区、平均窗口、最少样本三项同样遵循「0 或缺失回落默认值」：
	// 老任务的 config_json 里没有它们，反序列化都是 0，回落后即拿到防抖后的新默认行为。
	if config.SwitchMargin <= 0 {
		config.SwitchMargin = DefaultSupplierGroupSchedulingElectionSwitchMargin
	}
	if config.SwitchMargin > MaxSupplierGroupSchedulingElectionSwitchMargin {
		config.SwitchMargin = MaxSupplierGroupSchedulingElectionSwitchMargin
	}
	if config.LatencyWindowMinutes <= 0 {
		config.LatencyWindowMinutes = DefaultSupplierGroupSchedulingElectionLatencyWindowMinutes
	}
	if config.LatencyWindowMinutes > MaxSupplierGroupSchedulingElectionLatencyWindowMinutes {
		config.LatencyWindowMinutes = MaxSupplierGroupSchedulingElectionLatencyWindowMinutes
	}
	if config.LatencyMinSamples <= 0 {
		config.LatencyMinSamples = DefaultSupplierGroupSchedulingElectionLatencyMinSamples
	}
	if config.LatencyMinSamples > MaxSupplierGroupSchedulingElectionLatencyMinSamples {
		config.LatencyMinSamples = MaxSupplierGroupSchedulingElectionLatencyMinSamples
	}
	config.RequiredModelsByGroup = normalizeSupplierGroupElectionRequiredModels(config.RequiredModelsByGroup)
	return config
}

// normalizeSupplierGroupElectionRequiredModels 清洗「分组必需模型」：丢弃非正 groupID，
// 模型名 trim、去空、按小写去重保序；某分组清洗后为空则整条丢弃（等于该组无强制要求）。
// 返回 nil 而非空 map，让「没配置」与「配了空」在下游一致（len==0 都不触发覆盖）。
func normalizeSupplierGroupElectionRequiredModels(raw map[int64][]string) map[int64][]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[int64][]string, len(raw))
	for groupID, models := range raw {
		if groupID <= 0 {
			continue
		}
		seen := make(map[string]struct{}, len(models))
		cleaned := make([]string, 0, len(models))
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			key := strings.ToLower(model)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			cleaned = append(cleaned, model)
		}
		if len(cleaned) > 0 {
			out[groupID] = cleaned
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
