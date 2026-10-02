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
	// SupplierGroupSchedulingElectionReasonOverCapacity 是「本组在任账号数超过上限、多出来的被收敛掉」的关闭原因。
	// 措辞用「在任账号数」而不是「已开启账号数」：判据是「开着且未到失败阈值的在任者数 > 每组开启账号数」，
	// 而 union 语义下同一个账号可能被别的分组一并开启，只报本组开启数会漏掉这种共用。
	// 与 ReasonNotElected 分开：被收敛掉的账号未必不是本组最优，只是本组已经开够了账号，
	// 再留着就会突破「每组开启账号数」这个硬上限。
	SupplierGroupSchedulingElectionReasonOverCapacity = "分组在任账号数超过上限，收敛关闭"
	// 下面两条是「失败不再立刻关」后的新出口：
	// 未达阈值时保持原状等下一轮，或分组没有备选账号时保留调度（绝不把分组关成空组）。
	// 未达阈值的理由带上进度（第几次/共几次），否则运维只看到"没关"却不知道还要等几轮。
	SupplierGroupSchedulingElectionReasonFailedPendingFmt = "连续失败 %d/%d 次，未达阈值，暂不关闭"
	// 已经关着的账号也会落到这一条（本任务只关不重开），所以措辞用"保持现状"而不是"保留调度"。
	SupplierGroupSchedulingElectionReasonKeepLastOne = "分组内无备选账号，保持现状待人工确认"

	// DefaultSupplierGroupSchedulingElectionCountScoreCap 是"连续成功次数"对综合分的贡献上限的默认值。
	// 健康守护的连续成功计数只增不减（失败才归零），不封顶就会让现任赢家永久固化：
	// 50 次与 51 次没有实质区别，不该因此压过一个用时明显更短的账号。
	// 之所以做成可配而不是写死：不同分组对"资历"的重视程度不一样，有的就是想让长期稳定的
	// 账号更强势。调大它等于把胜负更多交回次数，代价是现任更难被明显更快的账号换掉。
	DefaultSupplierGroupSchedulingElectionCountScoreCap = 10
	// MaxSupplierGroupSchedulingElectionCountScoreCap 是封顶值的上限。
	// 再往上调只是把"资历"推向无限话语权、让择优退化成先到先得，故设上限防误配。
	MaxSupplierGroupSchedulingElectionCountScoreCap = 100

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
	// DefaultSupplierGroupSchedulingElectionPriorityWeight 是「账号优先级」的默认话语权。
	// 语义与用时一致：优先级分先归一到 [0,1] 再加权，权重比值即相对重要性。
	// 默认 0.5（均衡）：全部健康时，优先级最高者可得满分，正好抵一半的话语权，
	// 但不会凌驾于健康/速度之上。管理员可按分组偏好调大——用户诉求是「都健康时
	// 优先级越高越有优势」，调到 1.0 以上即让优先级在健康账号中压过用时分。
	DefaultSupplierGroupSchedulingElectionPriorityWeight = 0.5

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
	// TopNByGroup 是「每组开启账号数」的分组级覆盖：group_id → 该组的 TopN，命中即替换全局 TopN，
	// 未命中的分组仍用全局 TopN。语义与 RequiredModelsByGroup 一致（分组级 override map）：
	// 空 map 表示所有分组都用全局值（默认，无需数据迁移）。归一化里丢弃非正 groupID 与 <=0 的值
	// （等于该组不覆盖、回落全局），超上限的钳到上限，与全局 TopN 同口径。
	TopNByGroup map[int64]int `json:"group_scheduling_election_top_n_by_group"`
	// DisabledGroupIDs 存"被关闭择优"的分组 ID，空列表表示全部分组都参与。
	// 存关闭项而不是开启项，是为了让"新增分组默认参与"无需数据迁移。
	DisabledGroupIDs []int64 `json:"group_scheduling_election_disabled_group_ids"`
	// CountWeight / LatencyWeight 是"连续成功次数"与"测试用时"在综合分里的相对话语权。
	// 两项都先各自归一到 [0,1] 再加权，所以它们的比值就是两者的相对重要性：
	// 例 1.0 : 0.5 表示用时的影响上限是次数的一半（相当于"用时最多抵 Cap/2 次连续成功"）。
	// 0 或缺失都回落到默认值；LatencyWeight 显式配 0 即退化为"只看次数"的旧行为。
	CountWeight   float64 `json:"group_scheduling_election_count_weight"`
	LatencyWeight float64 `json:"group_scheduling_election_latency_weight"`
	// PriorityWeight 是「账号优先级」在综合分里的话语权，默认 0.5。
	// 优先级分在同一组内 min-max 归一化（数值越小优先级越高，映射到 1.0；最大者映射 0）；
	// 与用时同理，优先级全都相同或极差为 0 时给中性分 0.5，让这项不影响相对顺序。
	// 它与 Count/Latency 一样遵循「0 或缺失回落默认值」；
	// PriorityEnabledForGroup 关闭该分组时本项权重不参与计分（映射为 0）。
	PriorityWeight float64 `json:"group_scheduling_election_priority_weight"`
	// PriorityEnabledGlobal 是「账号优先级参与择优」的全局默认开关（默认 false=不参与，与升级前一致）。
	// 打开后所有分组默认把优先级计入综合分；个别分组可用下面两个名单反向覆盖。
	PriorityEnabledGlobal bool `json:"group_scheduling_election_priority_enabled_global"`
	// PriorityEnabledGroupIDs 是「强制开启优先级计分」的网络（force-on）：无论全局开关取何值，
	// 在此名单里的分组都计优先级。全局关闭时它等价于旧的 opt-in 名单（空=都不计，向后兼容）。
	PriorityEnabledGroupIDs []int64 `json:"group_scheduling_election_priority_enabled_group_ids"`
	// PriorityDisabledGroupIDs 是「强制不开启优先级计分」的网络（force-off，空=无排除，默认）。
	// 优先级最高：在此名单内的分组无论全局开关如何都不计优先级，即便同时出现在强制开启名单里也以此为准。
	PriorityDisabledGroupIDs []int64 `json:"group_scheduling_election_priority_disabled_group_ids"`
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
	// CountScoreCap 是"连续成功次数"对综合分的贡献上限，默认 10。
	// 次数分 = min(连续成功次数, CountScoreCap) / CountScoreCap，所以超过这个次数的账号得分完全相同
	// （连续成功 142 次与 190 次在默认值下没有任何差别）。0 或缺失都按默认值处理。
	CountScoreCap int `json:"group_scheduling_election_count_score_cap"`
	// KeepHealthyIncumbentGlobal 是「在任者健康锁定」的全局默认开关，默认 false（与升级前完全一致：默认不锁定）。
	// 打开后所有分组默认锁定，无需逐组勾选；个别分组可用下面两个名单反向覆盖它。
	KeepHealthyIncumbentGlobal bool `json:"group_scheduling_election_keep_healthy_incumbent_global"`
	// KeepHealthyIncumbentGroupIDs 是「强制锁定」名单（force-on）：无论全局开关取何值，列表里的分组都锁定。
	// 锁定含义：只要它「当前开启调度的账号」里没有连续失败到阈值的（单次抖动不算），就保留这些在任账号、
	// 不做任何择优与换人，直接跳过。全局关闭时它等价于旧的 opt-in 名单（空列表=都不锁定，向后兼容）。
	// 破锁口径与失败闸门（FailureThreshold）一致：开着的账号一次/几次没到阈值的失败仍算「在任健康」、保持锁定，
	// 交给失败闸门原地留着等翻盘；只有它连续失败真到阈值、失败闸门也要关它时，才破锁走正常择优补位，绝不锁死一个坏分组。
	KeepHealthyIncumbentGroupIDs []int64 `json:"group_scheduling_election_keep_healthy_incumbent_group_ids"`
	// KeepHealthyIncumbentExcludedGroupIDs 是「强制不锁定」名单（force-off）：无论全局开关取何值，列表里的分组都不锁定、正常择优。
	// 主要用途是在全局锁定打开时，把个别分组从「默认锁定」里排除出去。与 KeepHealthyIncumbentGroupIDs 互斥——
	// 前端保证同一分组不会同时进两个名单，后端按「强制不锁定优先」兜底（见 keepHealthyForGroup）。
	KeepHealthyIncumbentExcludedGroupIDs []int64 `json:"group_scheduling_election_keep_healthy_incumbent_excluded_group_ids"`
	// RequiredModelsByGroup 是「分组必须能服务的模型」清单：group_id → 模型名列表（空=该组无强制要求）。
	// 择优选出赢家后对每个必需模型做覆盖兜底：赢家里若没有账号支持它，就在组内**健康**账号中补选综合分最高的
	// 支持者 union-enable（哪怕它本不是最优）——保证换人不会把某个必需模型换没了。必需模型是硬底线，
	// 连「在任者健康锁定」的分组也会补齐。若支持该模型的账号当前全部测试失败/根本没有，则不硬留失败账号
	// （让正常赢家生效、允许切到能用的账号），只记一条告警待人工恢复。判定「是否支持」复用运行时
	// account.IsModelSupported，与网关逐请求过滤同源。
	RequiredModelsByGroup map[int64][]string `json:"group_scheduling_election_required_models"`
	// DryRun 打开后本任务进入「演练」：照常算赢家、照常产出切换日志，但绝不拨 accounts.schedulable，
	// 明细与日志里的每条都标成「建议」。用途是在改权重/阈值后先空跑几轮看它会切谁，
	// 而不是直接拿线上调度做实验。
	DryRun bool `json:"group_scheduling_election_dry_run"`
	// DryRunGroupIDs 是「只演练不生效」的分组 ID（opt-in），与 DisabledGroupIDs 相反极性。
	// 与 DryRun 是并集关系：总开关打开则全部分组演练，否则只有列表内的分组演练。
	// 之所以还要分组粒度：同一套权重在不同分组上的后果差别很大，常见诉求是「只盯住一两个分组」；
	// 全量演练会让所有分组同时失去择优，反而更像故障。
	DryRunGroupIDs []int64 `json:"group_scheduling_election_dry_run_group_ids"`
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
	// Priority 是账号的优先级（数值越小优先级越高）。不参与择优时（开关关闭）无意义，
	// 默认 0 表示未配置——与网关 filterByMinPriority 同源语义。
	Priority int
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
	// Suggested 为 true 表示这条只是「建议」：目标状态已算出来、也进了切换日志，
	// 但演练模式下没有真的写库。必须与真实切换区分——切换日志的取数条件就是
	// before <> after，不标的话运维会把"它想切"读成"它切了"。
	Suggested bool `json:"suggested,omitempty"`
	// GroupDecisions 按分组记录「为什么是它」：综合分与分量、组内名次、入选线、
	// 是否因覆盖必需模型被补选、是否走了在任者锁定等。
	// 一个账号同属多个分组时各组结论可以不同（在 A 组当选、在 B 组落选），
	// 所以依据必须按分组各存一份，不能只存一条。
	// ⚠️ 只有升级后新产生的运行才有这个字段，旧运行记录里没有，前端必须能降级显示。
	GroupDecisions []SupplierGroupSchedulingElectionDecisionDetail `json:"group_decisions,omitempty"`
}

// SupplierGroupSchedulingElectionDecisionDetail 是「某账号在某分组里为什么是这个裁决」。
// 存在的意义只有一个：让切换日志能自己解释清楚，管理员不必再去翻配置和源码。
type SupplierGroupSchedulingElectionDecisionDetail struct {
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name,omitempty"`
	// Scored 为 false 表示该账号在这个分组里当前不是「测试成功」，压根没进择优，
	// 下面那组评分字段全部无意义（零值），前端必须据此不显示评分。
	Scored bool `json:"scored"`
	// Elected 表示该账号在本组入选（择优前 N，或因覆盖必需模型被补选）。
	// 账号级 reason 是 union 的：在 A 组入选、在 B 组落选时整体仍记「当选」，
	// 只有逐组标出来才能解释「它为什么在某组没入选却依然被打开」。
	Elected bool `json:"elected,omitempty"`
	// 综合分 = CountWeight × CountScore + LatencyWeight × LatencyScore
	//        + PriorityWeight × PriorityScore，各项都给出来，管理员可以照着配置自己复算一遍。
	// ⚠️ 这组字段刻意不带 omitempty：0 是合法取值（新账号的连续成功次数就是 0），
	// 省略后前端只能显示「—」，会把「真的是 0」读成「没有数据」。
	Score          float64 `json:"score"`
	CountScore     float64 `json:"count_score"`
	LatencyScore   float64 `json:"latency_score"`
	PriorityScore  float64 `json:"priority_score"`
	CountWeight    float64 `json:"count_weight"`
	LatencyWeight  float64 `json:"latency_weight"`
	PriorityWeight float64 `json:"priority_weight"`
	// PriorityEnabled 表示该组本轮是否把「账号优先级」计入综合分（分组级开关的最终判定结果）。
	// 必须单独给出来：PriorityScore 为 0 时只看分数分不清「本组压根不计优先级」还是
	// 「计了、但这个账号算出来就是最低的 0 分」；PriorityWeight 两种情况都是配置值，同样分不出来。
	// 与上面那组评分字段同理，刻意不带 omitempty —— false 是合法取值，
	// 省略后前端只能把它读成「没有这个数据」。
	PriorityEnabled bool `json:"priority_enabled"`
	// CountScoreCap 是次数分的封顶值：连续成功次数超过它之后，次数分不再增长。
	CountScoreCap int `json:"count_score_cap"`
	// EffectiveLatencyMs 是参与评分的延迟（含成功率惩罚与在任者迟滞折算），
	// 与日志里显示的「测试用时」不是同一个数。
	EffectiveLatencyMs int64 `json:"effective_latency_ms,omitempty"`
	// LatencyFallback 为 true 表示用时分是中性值（同平台样本不足或极差为 0），不是算出来的。
	LatencyFallback bool `json:"latency_fallback,omitempty"`
	// Rank / RankTotal：综合分在该组「测试成功账号」里的名次（1 起）与参评总数。
	// 同样不带 omitempty —— 未参评时是 0，前端靠 Scored 区分，而不是靠字段缺失。
	Rank      int `json:"rank"`
	RankTotal int `json:"rank_total"`
	// WinnerCutoff 是入选分数线（第 TopN 名的综合分）；落选时用它说明「差多少」。
	WinnerCutoff float64 `json:"winner_cutoff"`
	TopN         int     `json:"top_n"`
	// Locked 表示该组本轮走「在任者健康锁定」，没有做择优 —— 此时名次只是参考，
	// 真正决定保留的是「它本来就开着且测试正常」。
	Locked bool `json:"locked,omitempty"`
	// OverCapacity 表示该账号在本组被「每组开启账号数」这个硬上限收敛掉了：本组当前开着的账号已经够数，
	// 本轮不再新开，超出的在任者在这里让位关闭。
	// 与 Locked 的分工：Locked 说明「本组本轮没做择优」，OverCapacity 说明「这个账号是这次收敛的代价」——
	// 只有后者为 true 时才不能对着一行"被关闭"的记录写「未换人」。
	OverCapacity bool `json:"over_capacity,omitempty"`
	// RequiredModels 非空表示该账号是因为「分组要求这些模型、而赢家里没人支持」被补选开启的。
	// 它的综合分并不是 TopN，不写清楚日志看起来像择优算错了。
	RequiredModels []string `json:"required_models,omitempty"`
	// NoAlternative 表示该组一个测试成功的账号都没有，失败账号因此保持原状待人工确认。
	NoAlternative bool `json:"no_alternative,omitempty"`
	// TestFailed 表示该账号在该组当前是测试失败状态。
	TestFailed bool `json:"test_failed,omitempty"`
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
	PendingCount int `json:"pending_count"`
	KeptCount    int `json:"kept_count"`
	// RequiredModelUncoveredCount / RequiredModelWarnings 记录「分组配置了必需模型、但当前没有健康账号能提供它」
	// 的情况。这不是失败（本轮仍让能用的账号生效），但必须被看见——否则某个模型静默断供、无人知晓。
	RequiredModelUncoveredCount int                                                   `json:"required_model_uncovered_count"`
	RequiredModelWarnings       []SupplierGroupSchedulingElectionRequiredModelWarning `json:"required_model_warnings,omitempty"`
	// DryRun 表示本轮处于演练配置下（总开关打开或配了演练分组）。
	// 放在结果顶层而不是只留在明细里，是因为运行列表默认只显示一行摘要，必须一眼看出这轮没生效。
	DryRun bool `json:"dry_run,omitempty"`
	// SuggestedEnabledCount / SuggestedDisabledCount 是演练模式下「本应开启/关闭但没有真改」的账号数。
	// 它们不计入 EnabledCount / DisabledCount —— 那两项的含义是「真的被拨动了」，
	// 把建议数混进去会让摘要骗人（"开启 3 个"里可能一个都没生效）。
	SuggestedEnabledCount  int                                          `json:"suggested_enabled_count"`
	SuggestedDisabledCount int                                          `json:"suggested_disabled_count"`
	Groups                 []SupplierGroupSchedulingElectionGroupDetail `json:"groups"`
	Items                  []SupplierGroupSchedulingElectionAccountItem `json:"items"`
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
	// Suggested 为 true 表示这条是演练产生的「建议切换」：目标状态算出来了，但没有真的写库。
	// 也就是说这份日志的定义条件（before <> after）在演练模式下描述的是"想改什么"而不是"改了什么"，
	// 不把它标出来，事后回看会把建议当成既成事实。
	Suggested bool `json:"suggested"`
	// GroupDecisions 按分组记录「为什么是它」，与运行明细里的同名字段同源。
	// ⚠️ 切换日志的 SQL 是逐字段投影的（见 supplierGroupSchedulingElectionChangeInnerSQL），
	// 这里加了字段还必须同步在投影里补一列，否则接口永远不会返回它。
	// 旧运行记录里没有这个字段，前端必须能降级。
	GroupDecisions []SupplierGroupSchedulingElectionDecisionDetail `json:"group_decisions,omitempty"`
}

type SupplierGroupSchedulingElectionChangeLogListParams struct {
	GroupID   int64
	AccountID int64
	// RunIDs 锁定到若干次择优运行（任务批次），把几批放在一起对照；空 = 不按批次筛。
	// 用列表而不是单值：页面顶部的批次标签可多选，行内批次按钮也是往同一份选择里加 ——
	// 两者必须共用一条路径，否则「顶部多选」和「行内单选」会互相覆盖。
	RunIDs []int64
	// Search 按账号名模糊匹配（分组管理页从某个分组进入时不带它，任务中心页全局看时用）。
	Search string
	// Platform 精确匹配平台（openai / anthropic …），空 = 不按平台筛。
	// 必须与 Search 分开：Search 同时匹配账号名与平台，拿它当平台筛选会把
	// 「名字里含 openai 的账号」一起捞进来，看着像筛选失灵。
	Platform    string
	Direction   string
	StartedFrom *time.Time
	StartedTo   *time.Time
	// IncludeSkipped 为 true 时，把「本该动却没动」的记录也纳入：在任者健康锁定、
	// 分组只剩它保留、连续失败未达阈值、写库失败。默认只看开关真被拨动（before <> after）的条目。
	IncludeSkipped bool
	Page           int
	PageSize       int
}

// SupplierGroupSchedulingElectionRecentRun 是顶部快捷标签用的一条「最近批次」。
// 带上 changed_at 是为了让前端显示「时间流水号」（20260923-100000）而不是裸的批次号 ——
// 裸号既看不出是哪一批，也读不出先后，跨天回看还得回头去对时间。
type SupplierGroupSchedulingElectionRecentRun struct {
	RunID     int64     `json:"run_id"`
	ChangedAt time.Time `json:"changed_at"`
}

type SupplierGroupSchedulingElectionChangeLogListResult struct {
	Items    []SupplierGroupSchedulingElectionChangeLog `json:"items"`
	Total    int64                                      `json:"total"`
	Page     int                                        `json:"page"`
	PageSize int                                        `json:"page_size"`
	// RecentRuns 是最近若干次「确实拨动过开关」的批次（新到旧），供页面顶部做快捷筛选标签。
	// 刻意**不受页面筛选影响**：它是个来回切换的入口，若跟着筛选一起收窄，
	// 点一下标签其余标签就消失了，反而没法用它换着看。
	//
	// 「最近」按批次内的最早变更时间排，**不是**按 run_id：runs.id 是自增主键，
	// 多个 worker 并发时会交错，按 id 排出来的「最近」在时间上并不成立。
	RecentRuns []SupplierGroupSchedulingElectionRecentRun `json:"recent_runs,omitempty"`
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
	// dryRun 表示该账号所属分组本轮只演练：算出目标状态并记进日志，但不拨 accounts.schedulable。
	// 只要它在**任一**所属分组被判演练就为真——账号的开关是单一字段，
	// 不可能「在 A 组演练、在 B 组真生效」，跟 winner 的 union 语义保持一致。
	dryRun bool
	// hold 非空表示这个账号本轮"故意没关"，取值见 supplierGroupElectionHold* 常量。
	// 非空即 notable：只靠汇总数字的话，运维看到"关闭 0 个"会以为任务没跑，
	// 而实际恰恰是最需要被看见的情况——有账号连续失败但被闸门拦住了。
	hold string
	// convergedOut 表示这个账号本轮是被「每组开启账号数」上限收敛掉的：它在某个分组里丢了名额，
	// 而没有任何分组选它。它只影响关闭原因文案（"收敛关闭"而不是"非分组最优"），不改变裁决本身。
	convergedOut bool
	// groupDecisions 是逐分组累积的裁决依据（见 SupplierGroupSchedulingElectionDecisionDetail），
	// 最终原样写进运行明细，供切换日志展开「为什么是它」。
	groupDecisions []SupplierGroupSchedulingElectionDecisionDetail
	// noAlternativeGroups 按分组记录「该组一个成功账号都没有」（闸门一）。
	// 账号级的 noAlternative 是 union 的，直接拿它填逐组依据会让其它分组也显示成"无备选"。
	noAlternativeGroups map[int64]struct{}
	// requiredModelsByGroup 记录「该账号因覆盖哪些必需模型被补选」：分组 → 模型名。
	// 必需模型补选发生在打分之后，而依据是在那之后才组装的，所以中途先攒在这里。
	requiredModelsByGroup map[int64][]string
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
	keepHealthyIncluded := make(map[int64]struct{}, len(config.KeepHealthyIncumbentGroupIDs))
	for _, groupID := range config.KeepHealthyIncumbentGroupIDs {
		keepHealthyIncluded[groupID] = struct{}{}
	}
	keepHealthyExcluded := make(map[int64]struct{}, len(config.KeepHealthyIncumbentExcludedGroupIDs))
	for _, groupID := range config.KeepHealthyIncumbentExcludedGroupIDs {
		keepHealthyExcluded[groupID] = struct{}{}
	}
	// keepHealthyForGroup 汇总「全局默认 + 分组级覆盖」得到某分组本轮是否走在任者健康锁定：
	// 强制不锁定名单 → 不锁；强制锁定名单 → 锁；两者都不在 → 跟随全局默认。
	// 强制不锁定优先于强制锁定，避免两个名单误配同一分组时行为不确定。
	keepHealthyForGroup := func(groupID int64) bool {
		if _, excluded := keepHealthyExcluded[groupID]; excluded {
			return false
		}
		if _, included := keepHealthyIncluded[groupID]; included {
			return true
		}
		return config.KeepHealthyIncumbentGlobal
	}
	priorityIncluded := make(map[int64]struct{}, len(config.PriorityEnabledGroupIDs))
	for _, groupID := range config.PriorityEnabledGroupIDs {
		priorityIncluded[groupID] = struct{}{}
	}
	priorityExcluded := make(map[int64]struct{}, len(config.PriorityDisabledGroupIDs))
	for _, groupID := range config.PriorityDisabledGroupIDs {
		priorityExcluded[groupID] = struct{}{}
	}
	// priorityEnabledForGroup 汇总「全局默认 + 分组级覆盖」得到某分组本轮是否把账号优先级计入综合分：
	// 强制不开启名单 → 不计；强制开启名单 → 计；两者都不在 → 跟随全局默认。
	// 强制不开启优先于强制开启，避免两个名单误配同一分组时行为不确定（与 keepHealthyForGroup 同规则）。
	priorityEnabledForGroup := func(groupID int64) bool {
		if _, excluded := priorityExcluded[groupID]; excluded {
			return false
		}
		if _, included := priorityIncluded[groupID]; included {
			return true
		}
		return config.PriorityEnabledGlobal
	}
	dryRunGroups := make(map[int64]struct{}, len(config.DryRunGroupIDs))
	for _, groupID := range config.DryRunGroupIDs {
		dryRunGroups[groupID] = struct{}{}
	}
	// topNForGroup 汇总「全局默认 + 分组级覆盖」得到某分组本轮的「每组开启账号数」：
	// 分组在覆盖表里且值有效 → 用它；否则回落全局 TopN。归一化已保证表里的值都在 [1, Max]，
	// 这里再兜一次 >0 判空，避免直接调用方传进未清洗的 0。
	topNForGroup := func(groupID int64) int {
		if n, ok := config.TopNByGroup[groupID]; ok && n > 0 {
			return n
		}
		return config.TopN
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
		if config.DryRun {
			account.dryRun = true
		} else if _, only := dryRunGroups[member.GroupID]; only {
			account.dryRun = true
		}
		account.groupIDs = append(account.groupIDs, member.GroupID)
	}
	sort.Slice(groupOrder, func(i, j int) bool { return groupOrder[i] < groupOrder[j] })

	result := SupplierGroupSchedulingElectionResult{
		// 演练标记取自配置而不是「本轮是否真有建议」：没有建议时（例如一切已是最优）
		// 摘要也必须写明这轮是演练，否则"开启 0、关闭 0"会被读成任务没干活。
		DryRun:       config.DryRun || len(dryRunGroups) > 0,
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
		// groupTopN 是本组本轮生效的「每组开启账号数」：分组级覆盖优先，否则全局 TopN。
		// 收敛上限、正常择优取前 N、入选分数线、逐组依据 top_n 四处都用它，口径必须一致。
		groupTopN := topNForGroup(groupID)
		// recordWinner 统一「标记账号赢家（union，跨组累积）+ 记入本组明细」。
		// 明细里始终追加（同一账号跨多组时每组都应列出），故不因 account.winner 已置而跳过追加。
		// requiredModel 非空表示这次入选是「为覆盖该必需模型」补选的、并不是综合分进了前 N ——
		// 切换日志必须能区分这两者，否则补选看起来像打分算错了。
		recordWinner := func(accountID int64, requiredModel string) {
			if account := accounts[accountID]; account != nil {
				account.winner = true
				if requiredModel != "" {
					if account.requiredModelsByGroup == nil {
						account.requiredModelsByGroup = make(map[int64][]string)
					}
					account.requiredModelsByGroup[groupID] = appendUniqueString(account.requiredModelsByGroup[groupID], requiredModel)
				}
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
					if account.noAlternativeGroups == nil {
						account.noAlternativeGroups = make(map[int64]struct{})
					}
					account.noAlternativeGroups[groupID] = struct{}{}
				}
			}
		}

		// 综合分依赖组内上下文（同平台内谁最快），必须按组算一次——锁定路径、正常择优、
		// 必需模型补选三处都复用它，保证「谁更优」的口径一致。
		electionScores := supplierGroupSchedulingElectionScores(successMembers, config.CountWeight, config.LatencyWeight, config.PriorityWeight, config.LatencyMinSamples, config.SwitchMargin, config.CountScoreCap, priorityEnabledForGroup(groupID))

		// 在任者健康锁定：仅对被 opt-in 的分组生效。只要该组「当前开着的账号」里没有
		// 已经连续失败到阈值的（单次抖动不算），就保留这些在任账号、跳过择优与换人。
		// 破锁口径必须与失败闸门（supplierGroupSchedulingElectionDecide 的闸门二）一致——
		// 都以「连续失败累计到 FailureThreshold」为界：一次/几次没到阈值的失败仍算「在任」、保住席位，
		// 本轮被失败闸门原地留着等翻盘，锁定继续、绝不换人；只有它连续失败真到阈值、失败闸门也要关它时，
		// 锁才破、才走正常择优补位。否则一次网络抖动就会破锁开出替补，等抖动账号翻盘、锁再合上，
		// 就把两个账号永久焊在一起（多活累积）——单在任者分组尤其致命：在任者一失败 scheduledHealthy 就空了，
		// 若只看「有没有健康在任者」，锁根本合不上，故这里以「有没有未到阈值的在任者」为准。
		locked := false
		// convergedDropped 记录本组因「在任账号数超过上限」被收敛掉的账号，供明细标注关闭原因。
		convergedDropped := make(map[int64]struct{})
		if keepHealthyForGroup(groupID) {
			scheduledHealthy := make([]SupplierGroupSchedulingElectionMember, 0) // 开着且测试成功的在任者，锁定时记为赢家
			// scheduledIncumbentCount 是「开着、且没到关闭阈值的在任者」总数（含成功、含失败但未到阈值）。
			// 只统计已测出状态的成员：未测过的账号走「保持原状」分支、本任务关不掉它们，
			// 把它们算进容量只会白白放弃锁定、丢掉防抖动能力，却换不来收敛。
			scheduledIncumbentCount := 0
			scheduledFailedPastThreshold := false
			for _, member := range groupMembers {
				if !member.Schedulable {
					continue
				}
				switch strings.TrimSpace(member.LastTestStatus) {
				case SupplierGroupSchedulingElectionTestStatusSuccess:
					scheduledHealthy = append(scheduledHealthy, member)
					scheduledIncumbentCount++
				case SupplierGroupSchedulingElectionTestStatusFailed:
					if member.FailedCount+1 >= config.FailureThreshold {
						scheduledFailedPastThreshold = true
						continue
					}
					// 失败但没到阈值：算作仍在任、保住席位等翻盘，本轮不换人。不记为赢家——
					// 它「失败待观察」的原因要如实进明细，保留调度靠失败闸门（闸门二）而不是这里。
					scheduledIncumbentCount++
				}
			}
			// 只要还有「没到关闭阈值的在任者」，本组就走锁定：保留在任者、跳过择优与换人。
			//
			// 但「每组开启账号数」是硬上限，而锁定的动作只是「记赢家、从不关人」，所以这里必须自己收敛：
			// 保留的在任者最多 TopN 个，超出的不记赢家 —— 不被其它分组共用的会被关闭（存量自愈），
			// 被共用的仍由那些分组保住（重合导致的多活是被允许的）。
			//
			// 为什么不能像以前那样「超容量就退回正常择优」：正常择优取的是「本组综合分前 N 名」，
			// 而 winner 是账号级、跨组取并集的，本组前 N 名往往不是当前开着的那几个 ——
			// 于是会在已有的多活之上**再开一个**，每轮只增不减、无法自愈。
			// 实测：某分组已经开着两个账号（都是别组的在任者、本组关不掉），本轮又开出本组 top1，变成 3 个；
			// 下一轮容量判断依然超，于是继续加开。
			//
			// 排序：先「已被其它分组选为赢家」的成员（关掉它会连累那些分组），再按综合分降序。
			// 这样多组共用的账号优先留下，只服务本组的那个先让位 —— 收敛的目标正是后者。
			if scheduledIncumbentCount > 0 && !scheduledFailedPastThreshold {
				sort.SliceStable(scheduledHealthy, func(i, j int) bool {
					sharedI := accounts[scheduledHealthy[i].AccountID] != nil && accounts[scheduledHealthy[i].AccountID].winner
					sharedJ := accounts[scheduledHealthy[j].AccountID] != nil && accounts[scheduledHealthy[j].AccountID].winner
					if sharedI != sharedJ {
						return sharedI
					}
					return supplierGroupSchedulingElectionMemberLess(scheduledHealthy[i], scheduledHealthy[j], electionScores)
				})
				limit := groupTopN
				if limit > len(scheduledHealthy) {
					limit = len(scheduledHealthy)
				}
				// 保留集 = TopN 名额 + 「必需模型覆盖所需的在任支持者」。
				// 后者必须**额外**保留、不占 TopN 名额：必需模型是硬底线，而补选跑在收敛之后、只增开 ——
				// 若这里把支持者当成「超出 TopN 的在任者」关掉，下一轮补选就会再开一个，
				// 且补选目标随组内 min-max 归一化的延迟抖动而换人，
				// 表现为同一分组每轮「关一个、开一个」来回横跳、永不收敛。
				kept := make(map[int64]struct{}, len(scheduledHealthy))
				for index := 0; index < limit; index++ {
					kept[scheduledHealthy[index].AccountID] = struct{}{}
				}
				for _, accountID := range supplierGroupElectionRequiredModelKeepers(scheduledHealthy, limit, config.RequiredModelsByGroup[groupID]) {
					kept[accountID] = struct{}{}
				}
				for _, member := range scheduledHealthy {
					if _, keep := kept[member.AccountID]; keep {
						recordWinner(member.AccountID, "")
						continue
					}
					convergedDropped[member.AccountID] = struct{}{}
					if account := accounts[member.AccountID]; account != nil {
						account.convergedOut = true
					}
				}
				locked = true
			}
		}

		// 正常择优：锁定未生效时才做。综合分排序后取前 N。
		if !locked {
			sort.SliceStable(successMembers, func(i, j int) bool {
				return supplierGroupSchedulingElectionMemberLess(successMembers[i], successMembers[j], electionScores)
			})
			winners := groupTopN
			if winners > len(successMembers) {
				winners = len(successMembers)
			}
			for index := 0; index < winners; index++ {
				recordWinner(successMembers[index].AccountID, "")
			}
		}

		// 必需模型覆盖兜底：赢家没覆盖的必需模型，补选一个健康支持者 union-enable；全失败则告警。
		// 必需模型是硬底线，锁定组也执行——它只增开支持者、不动在任赢家，不破坏锁定语义。
		applySupplierGroupRequiredModelCoverage(
			groupID, groupNames[groupID], config.RequiredModelsByGroup[groupID],
			groupMembers, successMembers, electionScores, detail.WinnerIDs, recordWinner, &result,
		)

		// 组装本组逐账号的裁决依据，最终原样写进运行明细，供切换日志展开「原因」。
		// 必须放在必需模型覆盖之后：补选也算入选，且 RequiredModels 正是在那一步记下的。
		winnerSet := make(map[int64]struct{}, len(detail.WinnerIDs))
		for _, id := range detail.WinnerIDs {
			winnerSet[id] = struct{}{}
		}
		// 名次按综合分另排一份副本：锁定组本轮没做择优（successMembers 保持未排序），
		// 但名次仍要算——否则看不出「在任者按分其实进不了前 N，只是被锁定保住了」。
		ranked := make([]SupplierGroupSchedulingElectionMember, len(successMembers))
		copy(ranked, successMembers)
		sort.SliceStable(ranked, func(i, j int) bool {
			return supplierGroupSchedulingElectionMemberLess(ranked[i], ranked[j], electionScores)
		})
		rankByID := make(map[int64]int, len(ranked))
		for index, member := range ranked {
			rankByID[member.AccountID] = index + 1
		}
		// 入选分数线 = 实际取到的最后一名（TopN 超过参评数时就是末位）的综合分。
		cutoffIndex := groupTopN
		if cutoffIndex > len(ranked) {
			cutoffIndex = len(ranked)
		}
		winnerCutoff := 0.0
		if cutoffIndex > 0 {
			winnerCutoff = electionScores[ranked[cutoffIndex-1].AccountID].Score
		}

		for _, member := range groupMembers {
			account := accounts[member.AccountID]
			if account == nil {
				continue
			}
			decision := SupplierGroupSchedulingElectionDecisionDetail{
				GroupID:      groupID,
				GroupName:    groupNames[groupID],
				Locked:       locked,
				TopN:         groupTopN,
				RankTotal:    len(ranked),
				WinnerCutoff: winnerCutoff,
			}
			// 被容量收敛掉的成员必须单独标出来：它和「在任者健康锁定保住的」在明细里长得一样
			// （都带 Locked），但一个被关闭、一个被保留，不标就会把关闭写成「未换人」。
			if _, dropped := convergedDropped[member.AccountID]; dropped {
				decision.OverCapacity = true
			}
			if _, elected := winnerSet[member.AccountID]; elected {
				decision.Elected = true
			}
			if _, noAlternative := account.noAlternativeGroups[groupID]; noAlternative {
				decision.NoAlternative = true
			}
			decision.TestFailed = strings.TrimSpace(member.LastTestStatus) == SupplierGroupSchedulingElectionTestStatusFailed
			// 只有测试成功的账号才进了择优，才有分可摊开；失败/未测的保持零值。
			if score, ok := electionScores[member.AccountID]; ok {
				decision.Scored = true
				decision.Score = score.Score
				decision.CountScore = score.CountScore
				decision.LatencyScore = score.LatencyScore
				decision.PriorityScore = score.PriorityScore
				decision.CountWeight = config.CountWeight
				decision.LatencyWeight = config.LatencyWeight
				decision.PriorityWeight = config.PriorityWeight
				// 分组级开关的判定结果：本组不计优先级时 PriorityScore 会被置 0，
				// 光看分数和权重都还原不出「是没计还是算出来就是 0」，所以在这里显式记一笔。
				decision.PriorityEnabled = priorityEnabledForGroup(groupID)
				decision.CountScoreCap = config.CountScoreCap
				decision.EffectiveLatencyMs = score.EffectiveLatencyMs
				decision.LatencyFallback = score.LatencyFallback
				decision.Rank = rankByID[member.AccountID]
			}
			decision.RequiredModels = account.requiredModelsByGroup[groupID]
			account.groupDecisions = append(account.groupDecisions, decision)
		}

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
			GroupDecisions:    account.groupDecisions,
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
			// 本该重新择优、却因故没动的条目也要落库，这样切换日志的「含未切换」视图
			// 才有据可查：hold（无备选保留 / 失败待观察）、写库失败早已在此收，
			// 唯独「在任者健康锁定保住的在任账号」before==after 且 hold 为空，需按锁定标记单独放行。
			if account.hold != "" || item.ErrorMessage != "" || supplierGroupElectionItemHasLockedIncumbent(item) {
				result.Items = append(result.Items, item)
			}
			continue
		}

		// 演练：走到这里说明本轮"本该拨动"这个账号，但只记建议、不写库。
		// 连续失败计数仍在上面照常落库——它决定的是下一轮达不达阈值，
		// 演练轮次若不计数，多轮演练就永远看不到「达到阈值才会关」的那一天。
		if account.dryRun {
			item.Suggested = true
			switch item.Action {
			case SupplierGroupSchedulingElectionActionEnabled:
				result.SuggestedEnabledCount++
			case SupplierGroupSchedulingElectionActionDisabled:
				result.SuggestedDisabledCount++
			}
			result.Items = append(result.Items, item)
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

// supplierGroupElectionItemHasLockedIncumbent 判断这条明细是否是「在任者健康锁定保住的在任账号」：
// 它在某个所属分组里走了锁定、且在该组入选（= 本来就开着、测试正常，被原地保留）。
// 这类账号 before==after 且 hold 为空，默认不会进切换日志，但它正是「本该重新择优却因锁定跳过」
// 的记录，「含未切换」视图要能看到，所以在落库时按此标记单独放行。
// 只认 Locked && Elected：锁定组里未开启的成员也带 Locked 标记，但它们本就不是被保住的在任者。
func supplierGroupElectionItemHasLockedIncumbent(item SupplierGroupSchedulingElectionAccountItem) bool {
	for _, decision := range item.GroupDecisions {
		if decision.Locked && decision.Elected {
			return true
		}
	}
	return false
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
			// 关闭原因要能自解释：被容量收敛掉的账号未必不是本组最优（它可能是本组 top1，
			// 只是本组已经开够了账号），记成「非分组最优」会让运维去翻配置找那个不存在的评分问题。
			if account.convergedOut {
				return false, SupplierGroupSchedulingElectionActionDisabled, SupplierGroupSchedulingElectionReasonOverCapacity
			}
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

// supplierGroupSchedulingElectionScore 是一个候选的综合分及其分量。
// 拆开返回而不是只给一个 float64：切换日志要把「为什么是它当选」摊开给管理员复核，
// 只给一个综合分是没法验算的 —— 权重、封顶、归一化口径都在配置里，看不到分量就只能猜。
type supplierGroupSchedulingElectionScore struct {
	Score         float64
	CountScore    float64 // 归一化后的次数分，0..1
	LatencyScore  float64 // 归一化后的用时分，0..1
	PriorityScore float64 // 归一化后的优先级分，0..1；该组未开启优先级计分时为 0
	// EffectiveLatencyMs 是真正参与评分的那份延迟：窗口平均（可信时）叠加成功率惩罚，
	// 在任者再乘迟滞折算系数。与日志里显示的「测试用时」不是同一个数，必须分开标。
	EffectiveLatencyMs int64
	// LatencyFallback 为 true 表示用时分没能算出差异（同平台成功样本不足 2 个、或极差为 0），
	// 取了中性值 0.5。不标出来，管理员会把它当成「算出来的 0.5」去反推配置。
	LatencyFallback bool
}

// supplierGroupSchedulingElectionScores 给同一分组内的候选账号算综合分，越大越优。
// 分组内各候选应已同步过「是否参与优先级计分」开关（priorityEnabledForGroup 判定结果）。
func supplierGroupSchedulingElectionScores(members []SupplierGroupSchedulingElectionMember, countWeight, latencyWeight, priorityWeight float64, latencyMinSamples int, switchMargin float64, countScoreCap int, priorityEnabled bool) map[int64]supplierGroupSchedulingElectionScore {
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

	// 封顶值可配（0 或缺失已在归一化里回落默认，这里再兜一次）：
	// 兜底不能省——直接调用本函数的测试/未来调用方可能传 0，那会让次数分除零。
	countCap := float64(countScoreCap)
	if countCap <= 0 {
		countCap = float64(DefaultSupplierGroupSchedulingElectionCountScoreCap)
	}

	// 优先级分在同一组内 min-max 归一化：数值越小优先级越高（与网关 filterByMinPriority 同源），
	// 所以最小者映射 1.0、最大者映射 0。与用时同理：优先级全都相同或极差为 0 时给中性分 0.5，
	// 让这项不影响相对顺序。该组开关关闭（priorityEnabled=false）时一律取 0。
	priorityMin, priorityMax, priorityHasSpread := 0, 0, false
	if priorityEnabled {
		for _, member := range members {
			if !priorityHasSpread {
				priorityMin, priorityMax = member.Priority, member.Priority
				priorityHasSpread = true
			}
			if member.Priority < priorityMin {
				priorityMin = member.Priority
			}
			if member.Priority > priorityMax {
				priorityMax = member.Priority
			}
		}
	}

	scores := make(map[int64]supplierGroupSchedulingElectionScore, len(members))
	for _, member := range members {
		normalizedCount := float64(member.HealthyCount) / countCap
		if normalizedCount > 1 {
			normalizedCount = 1
		}
		latency := effectiveLatency[member.AccountID]
		normalizedLatency := 0.5
		latencyFallback := true
		// 至少两个样本才谈得上"谁更快"；极差为 0 说明大家一样快，同样比不出高下。
		if latency > 0 && latencySamples[member.Platform] >= 2 {
			if span := maxLatency[member.Platform] - minLatency[member.Platform]; span > 0 {
				normalizedLatency = float64(maxLatency[member.Platform]-latency) / float64(span)
				latencyFallback = false
			}
		}
		normalizedPriority := 0.0
		if priorityEnabled {
			if priorityMax > priorityMin {
				normalizedPriority = float64(priorityMax-member.Priority) / float64(priorityMax-priorityMin)
			} else {
				// 斜率存在但大家都相同：给中性分，避免因「都设了同一个值」而一边倒。
				normalizedPriority = 0.5
			}
		}
		scores[member.AccountID] = supplierGroupSchedulingElectionScore{
			Score:              countWeight*normalizedCount + latencyWeight*normalizedLatency + priorityWeight*normalizedPriority,
			CountScore:         normalizedCount,
			LatencyScore:       normalizedLatency,
			PriorityScore:      normalizedPriority,
			EffectiveLatencyMs: latency,
			LatencyFallback:    latencyFallback,
		}
	}
	return scores
}

// supplierGroupSchedulingElectionMemberLess 定义"更优"排序：综合分高者优先。
// 迟滞死区不在这里做（会被组内 min-max 归一化放大而失效，两候选时分差恒为 latencyWeight），
// 而是在 supplierGroupSchedulingElectionScores 里对在任者的延迟按比例折算实现——见那里的注释。
// 综合分并列时优先保留「当前已开启调度」的账号（黏性），再按最近测试时间、账号 ID 兜底，保证确定性。
func supplierGroupSchedulingElectionMemberLess(a, b SupplierGroupSchedulingElectionMember, scores map[int64]supplierGroupSchedulingElectionScore) bool {
	scoreA, scoreB := scores[a.AccountID].Score, scores[b.AccountID].Score
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

// supplierGroupElectionRequiredModelKeepers 挑出「为了让本组必需模型都有在任者覆盖，
// 必须额外保留（不占 TopN 名额）的在任者 ID」。
//
// 为什么收敛需要它：必需模型覆盖是在补选阶段（applySupplierGroupRequiredModelCoverage）才做的，
// 而补选跑在收敛之后、且只增开 —— 收敛若按「综合分前 TopN」把上一轮补选进来的支持者关掉，
// 下一轮补选就会再开一个，目标还会随组内 min-max 归一化的延迟抖动而换人，
// 于是同一分组每轮「关一个、开一个」来回横跳（生产实测 2026-10-01：两个分组一小时对调两次）。
// 把支持者纳入保留集，补选者一旦开启就不会被下一轮收敛淘汰，振荡即停。
//
// 口径与补选同构：每个未被 TopN 名额覆盖的必需模型，保留综合分最高的一个支持者
// （一个账号可同时覆盖多个模型，故实际保留数 ≤ 未覆盖模型数）。
// members 必须已按「更优在前」排好序，前 limit 个即 TopN 名额。
func supplierGroupElectionRequiredModelKeepers(members []SupplierGroupSchedulingElectionMember, limit int, requiredModels []string) []int64 {
	// 在任者全在 TopN 名额内时，没有可额外保留的对象。
	if len(requiredModels) == 0 || limit >= len(members) {
		return nil
	}
	// 小写 → 原始名：去空、按小写去重，与 applySupplierGroupRequiredModelCoverage 同口径。
	uncovered := make(map[string]string, len(requiredModels))
	for _, model := range requiredModels {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		uncovered[strings.ToLower(trimmed)] = trimmed
	}
	// TopN 名额里已经有人支持的模型算已覆盖，不必再额外保留。
	for index := 0; index < limit; index++ {
		for key, model := range uncovered {
			if supplierGroupElectionMemberSupportsModel(members[index], model) {
				delete(uncovered, key)
			}
		}
	}
	if len(uncovered) == 0 {
		return nil
	}
	keepers := make([]int64, 0, len(uncovered))
	for index := limit; index < len(members) && len(uncovered) > 0; index++ {
		covered := false
		for key, model := range uncovered {
			if supplierGroupElectionMemberSupportsModel(members[index], model) {
				delete(uncovered, key)
				covered = true
			}
		}
		if covered {
			keepers = append(keepers, members[index].AccountID)
		}
	}
	return keepers
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
	electionScores map[int64]supplierGroupSchedulingElectionScore,
	initialWinnerIDs []int64,
	// recordWinner 的第二个参数是该次入选所覆盖的必需模型（正常择优传空串）。
	recordWinner func(int64, string),
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
			recordWinner(best.AccountID, model)
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

// appendUniqueString 追加一个去重后的字符串。同一账号可能因多个必需模型被补选，
// 而每个模型只应记一次；用 slice 而不是 set 是为了让日志里的顺序稳定可读。
func appendUniqueString(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func normalizeSupplierGroupSchedulingElectionConfig(config SupplierGroupSchedulingElectionConfig) SupplierGroupSchedulingElectionConfig {
	if config.TopN <= 0 {
		config.TopN = DefaultSupplierGroupSchedulingElectionTopN
	}
	if config.TopN > MaxSupplierGroupSchedulingElectionTopN {
		config.TopN = MaxSupplierGroupSchedulingElectionTopN
	}
	config.TopNByGroup = normalizeSupplierGroupElectionTopNByGroup(config.TopNByGroup)
	config.DisabledGroupIDs = uniquePositiveInt64s(config.DisabledGroupIDs)
	sort.Slice(config.DisabledGroupIDs, func(i, j int) bool {
		return config.DisabledGroupIDs[i] < config.DisabledGroupIDs[j]
	})
	config.KeepHealthyIncumbentGroupIDs = uniquePositiveInt64s(config.KeepHealthyIncumbentGroupIDs)
	sort.Slice(config.KeepHealthyIncumbentGroupIDs, func(i, j int) bool {
		return config.KeepHealthyIncumbentGroupIDs[i] < config.KeepHealthyIncumbentGroupIDs[j]
	})
	config.KeepHealthyIncumbentExcludedGroupIDs = uniquePositiveInt64s(config.KeepHealthyIncumbentExcludedGroupIDs)
	sort.Slice(config.KeepHealthyIncumbentExcludedGroupIDs, func(i, j int) bool {
		return config.KeepHealthyIncumbentExcludedGroupIDs[i] < config.KeepHealthyIncumbentExcludedGroupIDs[j]
	})
	config.PriorityEnabledGroupIDs = uniquePositiveInt64s(config.PriorityEnabledGroupIDs)
	sort.Slice(config.PriorityEnabledGroupIDs, func(i, j int) bool {
		return config.PriorityEnabledGroupIDs[i] < config.PriorityEnabledGroupIDs[j]
	})
	config.PriorityDisabledGroupIDs = uniquePositiveInt64s(config.PriorityDisabledGroupIDs)
	sort.Slice(config.PriorityDisabledGroupIDs, func(i, j int) bool {
		return config.PriorityDisabledGroupIDs[i] < config.PriorityDisabledGroupIDs[j]
	})
	config.DryRunGroupIDs = uniquePositiveInt64s(config.DryRunGroupIDs)
	sort.Slice(config.DryRunGroupIDs, func(i, j int) bool {
		return config.DryRunGroupIDs[i] < config.DryRunGroupIDs[j]
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
	// 优先级权重与 Count/Latency 同理遵循「0 或缺失回落默认值」：
	// 旧 config_json 里没有它，反序列化是 0，回落即拿到默认 0.5 —— 但只有真正
	// 开启了「优先级参与择优」的分组（priorityEnabledForGroup）才会计入综合分，
	// 否则权重再大也是空转，升级后不改变任何现有分组的结果。
	if config.PriorityWeight <= 0 {
		config.PriorityWeight = DefaultSupplierGroupSchedulingElectionPriorityWeight
	}
	if config.PriorityWeight > MaxSupplierGroupSchedulingElectionWeight {
		config.PriorityWeight = MaxSupplierGroupSchedulingElectionWeight
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
	// 次数封顶值同样遵循「0 或缺失回落默认值」：旧任务的 config_json 里没有这个字段，
	// 反序列化后是 0，回落即拿到封顶 10 的既有行为，升级后不改变任何现有分组的结果。
	if config.CountScoreCap <= 0 {
		config.CountScoreCap = DefaultSupplierGroupSchedulingElectionCountScoreCap
	}
	if config.CountScoreCap > MaxSupplierGroupSchedulingElectionCountScoreCap {
		config.CountScoreCap = MaxSupplierGroupSchedulingElectionCountScoreCap
	}
	config.RequiredModelsByGroup = normalizeSupplierGroupElectionRequiredModels(config.RequiredModelsByGroup)
	return config
}

// normalizeSupplierGroupElectionTopNByGroup 清洗「每组开启账号数」的分组级覆盖：
// 丢弃非正 groupID；值 <=0 视为「不覆盖」丢弃（下游回落全局 TopN）；超上限的钳到上限，
// 与全局 TopN 的归一化同口径。返回 nil 而非空 map，让「没配置」与「配了空」在下游一致
// （len==0 都表示所有分组用全局值）。
func normalizeSupplierGroupElectionTopNByGroup(raw map[int64]int) map[int64]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[int64]int, len(raw))
	for groupID, topN := range raw {
		if groupID <= 0 || topN <= 0 {
			continue
		}
		if topN > MaxSupplierGroupSchedulingElectionTopN {
			topN = MaxSupplierGroupSchedulingElectionTopN
		}
		out[groupID] = topN
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
