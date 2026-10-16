package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// 检测项类型：logic = 有唯一答案的逻辑题，render = 生成作品的开放题。
const (
	ModelDegradationTaskLogic  = "logic"
	ModelDegradationTaskRender = "render"
)

// 单次检测的整体判定。
const (
	ModelDegradationVerdictNormal   = "normal"
	ModelDegradationVerdictDegraded = "degraded"
	ModelDegradationVerdictUnknown  = "unknown"
)

// 单题判定结果：逻辑题（答对 / 答错 / 未响应）与生成题（结构完整 / 退化 / 未响应）共用。
const (
	ModelDegradationItemOK        = "ok"
	ModelDegradationItemDegraded  = "degraded"
	ModelDegradationItemUnknown   = "unknown"
	ModelDegradationItemGenerated = "generated"
)

// modelDegradationLogicTask 一道有唯一数字答案的逻辑题。
//
// 选题标准：答案唯一、可被正则提取、难度落在「正常模型稳定答对、被降智或换成
// 小模型的实例容易答错」的区间。答案过于显然（如 3+5）没有区分度，故不收录。
type modelDegradationLogicTask struct {
	Key string
	// Title 短标题。题目全文太长，塞进下拉选项里没法扫，所以另给一个短标题，
	// 全文由前端单独铺在「查看题目」区域。
	Title    string
	Prompt   string
	Expected string
}

// modelDegradationNoToolsPrefix 所有逻辑题统一的开场白。
//
// 摆在题首而不是题尾：模型偶尔会用「我没有联网 / 无法调用工具」当借口拒答，
// 先把这句话说在前面，拒答本身就成为一个可判定的降智信号。
const modelDegradationNoToolsPrefix = "不使用任何外部工具回答以下问题："

// modelDegradationLogicTasks 内置逻辑题库。当前只有一道题（每轮单选一道）。
//
// 题面照搬 manxue-ai 的 CANDY_PROMPT（仅保留本仓的「不使用外部工具」开场白），
// 判分口径也照搬其 candy_passes：只认独立出现的期望数字（见 judgeModelDegradationLogic），
// 所以不要求「只回答数字」，带解释的长回答同样可判。
var modelDegradationLogicTasks = []modelDegradationLogicTask{
	{
		Key:   "candy-shape-flavor",
		Title: "糖果摸取：保证不同形状的苹果味和桃子味",
		Prompt: modelDegradationNoToolsPrefix +
			"在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。" +
			"现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，" +
			"那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？" +
			"（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）\n\n" +
			"| 形状 | 苹果味 | 桃子味 | 西瓜味 |\n" +
			"| 圆形 | 7 | 9 | 8 |\n" +
			"| 五角星形 | 7 | 6 | 4 |",
		Expected: "21",
	},
}

// 生成题（唯一的预制题）的标识与短标题。
const (
	modelDegradationRenderKey   = "pelican-bicycle"
	modelDegradationRenderTitle = "画一只骑自行车的鹈鹕"
)

// modelDegradationRenderScenes 生成题的场景池，与 manxue-ai 的 SCENES 一致。
// 每轮随机选一个注入题面：同一道题反复提交时题面不重样，上游缓存或回放的旧作品
// 会当场露馅（配合每轮的随机校验码一起看，见 newModelDegradationRenderNonce）。
var modelDegradationRenderScenes = []string{
	"海边木栈道", "秋日林道", "春日草坡", "雨后湿地",
	"黄昏公路", "湖畔风车", "热带海岛", "雪山谷地",
}

// modelDegradationRenderPromptTemplate 生成题的题面模板（照搬 manxue-ai 的 PROMPT，
// 仅保留本仓的保护句，见下）。
//
// 与逻辑题不同，这题没有唯一答案，也不再做自动判定：只生成、展示作品，好坏由用户
// 自己看。因此题目要固定到足以横向比较：同一 prompt 反复提交，源码应当每次不同。
//
// {scene} / {nonce} 是每轮随机填充的占位符，由 buildModelDegradationRenderPrompt
// 拼装；题库列表展示时用 modelDegradationRenderPromptDisplay 换成说明文字。
//
// 首尾两句不是客套：实测有上游（编码代理类模型）把「创建…」当成建文件任务，
// 回一句「需要先查看当前工作区结构，然后创建一个可直接打开的文件」就收工 ——
// 我们是非流式单轮调用，不会再有下一轮，于是整轮被判成「生成题没有产出」。
// 所以必须显式要求：本轮直接给出源码，不许建文件、不许调工具、不许只描述计划。
const modelDegradationRenderPromptTemplate = modelDegradationNoToolsPrefix +
	"创建一幅独立的 SVG 鹈鹕骑自行车 2D 循环动画。" +
	"主角是一只可爱的鹈鹕，有橙色大嘴、踩在踏板上的蹼足，以及微微张开保持平衡的翅膀。" +
	"自行车轮子持续转动，脚与踏板动作协调，动画流畅，背景明亮，配色鲜活。" +
	"本次场景：{scene}。请为这个场景独立构图。" +
	"在画面右下角用可见的 SVG text 元素显示本次校验码：{nonce}。" +
	"只返回一个完整的 SVG，可使用内联 CSS 或 SMIL 动画；" +
	"不要 HTML、JavaScript、外部图片、外部字体或其他外部资源。" +
	"请直接在本次回复里输出完整的 SVG 源码，不要创建文件、不要调用工具、不要询问确认，" +
	"也不要只描述计划或解释做法。"

// buildModelDegradationRenderPrompt 拼出本轮生成题的题面：随机场景 + 随机校验码。
//
// 不强求加密随机：math/rand/v2 足够分散，这两个值只用于防回放与人工核对。
func buildModelDegradationRenderPrompt() string {
	scene := modelDegradationRenderScenes[rand.IntN(len(modelDegradationRenderScenes))] //nolint:gosec
	return strings.NewReplacer(
		"{scene}", scene,
		"{nonce}", newModelDegradationRenderNonce(),
	).Replace(modelDegradationRenderPromptTemplate)
}

// newModelDegradationRenderNonce 生成本轮校验码：8 位十六进制大写，与 manxue-ai 的
// secrets.token_hex(4).upper() 一致。模型需把它画在画面右下角的 SVG text 里，
// 人工核对时一眼就能看出作品是不是回放的历史输出。
func newModelDegradationRenderNonce() string {
	return fmt.Sprintf("%08X", rand.Uint32()) //nolint:gosec
}

// modelDegradationRenderPromptDisplay 题库列表里的展示题面：把两个动态占位符
// 换成说明文字。实际下发给模型的题面每轮由 buildModelDegradationRenderPrompt 拼装，
// 并原样存进检测明细（前端运行详情能看到当轮的真实题面）。
func modelDegradationRenderPromptDisplay() string {
	return strings.NewReplacer(
		"{scene}", "（每轮从场景池随机选择）",
		"{nonce}", "（每轮随机生成）",
	).Replace(modelDegradationRenderPromptTemplate)
}

// modelDegradationCustomKey 自定义题在库里的标识。多道自定义题共用它 ——
// 它只用来区分「预制 / 自定义」，题目全文另有 prompt 字段。
const modelDegradationCustomKey = "custom"

// 每轮检测各类型最多能跑几道题。
//
// 单选：逻辑题与生成题在一轮里都只跑一道。这两个上限不是为了好看：
// 单轮的整体超时是按「各题上限之和」推导出来的（见 modelDegradationRunTotalTimeout），
// 放开数量会让它失效。题库里两类目前各只有一道预制题，前端单选呈现。
const (
	modelDegradationMaxLogicQuestions  = 1
	modelDegradationMaxRenderQuestions = 1
)

// modelDegradationMaxPromptRunes 自定义题的题目长度上限（按字符数）。
// 题目会原样发给上游按 token 计费，给个上限免得一次请求把额度烧掉。
const modelDegradationMaxPromptRunes = 2000

var (
	// 回答中的数字（含负号与小数）。
	modelDegradationNumberRegex = regexp.MustCompile(`-?\d+(?:\.\d+)?`)
)

// judgeModelDegradationLogic 判断逻辑题回答是否命中期望答案。
//
// 判分口径照搬 manxue-ai 的 candy_passes：只认「独立出现」的期望数字 ——
// 前面不能紧贴数字、字母、下划线、点、正负号，后面不能紧贴数字、字母、下划线，
// 也不能是小数续写（期望 21 时「21.5」「21.0」都不算命中）。
// 「答案是 9 个球」「最少取出 21 颗」这类带解释的回答照常通过，
// 而「2121」「x21」「-21」这类噪声不会误判成答对。
//
// Python 的 lookaround 在 Go regexp（RE2）里表达不了，边界判定手工做。
func judgeModelDegradationLogic(response, expected string) (bool, string) {
	if strings.TrimSpace(response) == "" {
		return false, "上游没有返回内容"
	}

	// 期望答案先按数值归一化（"4.0" → "4"、"1e3" → "1000"）：自定义题的期望值
	// 是用户手填的，可能带小数尾巴或科学计数法，归一化后再做独立匹配，
	// 避免「题里填 4.0、模型答 4」这种同值不同写法被判错。
	if modelDegradationHasStandaloneNumber(response, normalizeModelDegradationExpected(expected)) {
		return true, "命中答案 " + expected
	}

	matches := modelDegradationNumberRegex.FindAllString(response, -1)
	if len(matches) == 0 {
		return false, "回答中没有可识别的数字（期望 " + expected + "）"
	}
	return false, "未出现期望答案 " + expected + "，回答中的数字：" + strings.Join(matches, ", ")
}

// modelDegradationHasStandaloneNumber 判断 text 里是否出现独立的 expected 字面。
//
// 边界规则与 manxue-ai 的 candy_passes 一致：
//   - 左侧：不能是数字、字母、下划线、点、正号、负号；
//   - 右侧：不能是数字、字母、下划线，也不能是「点 + 数字」的小数续写。
//
// 「数字」按 Unicode Nd 类别判断（与 Python 的 \d 口径一致），全角数字同样算数字。
func modelDegradationHasStandaloneNumber(text, expected string) bool {
	if expected == "" {
		return false
	}
	for from := 0; from+len(expected) <= len(text); {
		idx := strings.Index(text[from:], expected)
		if idx < 0 {
			return false
		}
		start := from + idx
		if modelDegradationNumberBoundaryOK(text, start, start+len(expected)) {
			return true
		}
		// 出现处被边界否决时从下一个字节继续找：expected 是 ASCII 数字串，
		// 后续命中只会落在完整字符边界上，逐字节推进不会切坏结果。
		from = start + 1
	}
	return false
}

// modelDegradationNumberBoundaryOK 判断 text[start:end) 处的字面两侧是否算「独立」。
func modelDegradationNumberBoundaryOK(text string, start, end int) bool {
	if r, _ := utf8.DecodeLastRuneInString(text[:start]); modelDegradationBlocksLeftBoundary(r) {
		return false
	}
	r, size := utf8.DecodeRuneInString(text[end:])
	if size == 0 {
		return true // 命中一直顶到文本末尾
	}
	if modelDegradationBlocksRightBoundary(r) {
		return false
	}
	if r == '.' {
		next, nextSize := utf8.DecodeRuneInString(text[end+size:])
		return !(nextSize > 0 && unicode.IsDigit(next))
	}
	return true
}

// modelDegradationBlocksLeftBoundary 左侧紧贴这些字符时不构成独立数字。
func modelDegradationBlocksLeftBoundary(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		r == '_' || r == '.' || r == '+' || r == '-'
}

// modelDegradationBlocksRightBoundary 右侧紧贴这些字符时不构成独立数字。
func modelDegradationBlocksRightBoundary(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_'
}

// normalizeModelDegradationExpected 把期望答案归一化成最短十进制写法。
// 解析失败时原样返回：调用方已校验过期望答案必须是数字，这里只是兜底。
func normalizeModelDegradationExpected(expected string) string {
	value, err := strconv.ParseFloat(strings.TrimSpace(expected), 64)
	if err != nil {
		return strings.TrimSpace(expected)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// normalizeModelDegradationSource 归一化源码后取 SHA256 指纹。
//
// 归一化只去掉空白字符：空白差异属于排版噪声，不影响「是否同一份源码」的判断；
// 而大小写、属性顺序等都可能被模型有意打散，过度归一化会造成误判为同一份。
func normalizeModelDegradationSource(source string) string {
	var b strings.Builder
	b.Grow(len(source))
	for _, r := range source {
		switch r {
		case ' ', '\t', '\n', '\r':
			continue
		}
		b.WriteRune(r)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// ModelDegradationLogicTaskOptions 列出逻辑题库，供前端铺下拉与「查看题目」。
func ModelDegradationLogicTaskOptions() []ModelDegradationTaskOption {
	options := make([]ModelDegradationTaskOption, 0, len(modelDegradationLogicTasks))
	for _, task := range modelDegradationLogicTasks {
		options = append(options, ModelDegradationTaskOption{
			TaskType: ModelDegradationTaskLogic,
			Key:      task.Key,
			Title:    task.Title,
			Prompt:   task.Prompt,
			Expected: task.Expected,
		})
	}
	return options
}

// ModelDegradationRenderTaskOptions 列出生成题库。当前只有一道预制题。
// 题面里的场景与校验码是每轮随机的，这里给的是换成说明文字的展示版。
func ModelDegradationRenderTaskOptions() []ModelDegradationTaskOption {
	return []ModelDegradationTaskOption{{
		TaskType: ModelDegradationTaskRender,
		Key:      modelDegradationRenderKey,
		Title:    modelDegradationRenderTitle,
		Prompt:   modelDegradationRenderPromptDisplay(),
	}}
}

// findModelDegradationLogicTask 按 key 查一道预制逻辑题。
func findModelDegradationLogicTask(key string) (modelDegradationLogicTask, bool) {
	for _, task := range modelDegradationLogicTasks {
		if task.Key == key {
			return task, true
		}
	}
	return modelDegradationLogicTask{}, false
}

// resolveModelDegradationQuestions 把「跑哪些检测项」解析成实际要跑的题目列表。
//
// questions 非空时以它为准，tasks 只用于兜底；为空时退回旧行为 ——
// 选中哪个类型就跑该类型的第一道预制题，这样老调用方不用改。
//
// 生成题排在最后：它最慢，放后面能让逻辑题的结果尽早出现在轮询里。
func resolveModelDegradationQuestions(tasks []string, questions []ModelDegradationQuestion) ([]ModelDegradationQuestion, error) {
	if len(questions) == 0 {
		return presetModelDegradationQuestions(tasks), nil
	}

	logic := make([]ModelDegradationQuestion, 0, modelDegradationMaxLogicQuestions)
	render := make([]ModelDegradationQuestion, 0, modelDegradationMaxRenderQuestions)
	seenPreset := make(map[string]bool, len(questions))
	seenCustom := make(map[string]bool, 2)

	for _, raw := range questions {
		question, err := resolveModelDegradationQuestion(raw)
		if err != nil {
			return nil, err
		}

		// 预制题按 key 去重（同一个 key 提交两次只跑一次）；自定义题按类型限一道。
		if question.Key == modelDegradationCustomKey {
			if seenCustom[question.TaskType] {
				return nil, fmt.Errorf("同一类型只能有一道自定义题")
			}
			seenCustom[question.TaskType] = true
		} else {
			dedup := question.TaskType + ":" + question.Key
			if seenPreset[dedup] {
				continue
			}
			seenPreset[dedup] = true
		}

		if question.TaskType == ModelDegradationTaskLogic {
			logic = append(logic, question)
			if len(logic) > modelDegradationMaxLogicQuestions {
				return nil, fmt.Errorf("逻辑题最多 %d 道", modelDegradationMaxLogicQuestions)
			}
			continue
		}
		render = append(render, question)
		if len(render) > modelDegradationMaxRenderQuestions {
			return nil, fmt.Errorf("生成题最多 %d 道", modelDegradationMaxRenderQuestions)
		}
	}

	if len(logic) == 0 && len(render) == 0 {
		return nil, fmt.Errorf("请至少选择一道题")
	}
	return append(logic, render...), nil
}

// presetModelDegradationQuestions 按检测项展开成该类型下的一道预制题。
//
// 不传 questions 的老调用方走这条路：每类只取第一道（与前端单选一致），
// 而不是把题库全跑一遍 —— 全跑会突破单题数量上限，也会把单轮拖到超时。
func presetModelDegradationQuestions(tasks []string) []ModelDegradationQuestion {
	selected := normalizeModelDegradationTasks(tasks)
	resolved := make([]ModelDegradationQuestion, 0, 2)

	if selected[ModelDegradationTaskLogic] && len(modelDegradationLogicTasks) > 0 {
		task := modelDegradationLogicTasks[0]
		resolved = append(resolved, ModelDegradationQuestion{
			TaskType: ModelDegradationTaskLogic,
			Key:      task.Key,
			Prompt:   task.Prompt,
			Expected: task.Expected,
		})
	}
	if selected[ModelDegradationTaskRender] {
		resolved = append(resolved, ModelDegradationQuestion{
			TaskType: ModelDegradationTaskRender,
			Key:      modelDegradationRenderKey,
			Prompt:   buildModelDegradationRenderPrompt(),
		})
	}
	return resolved
}

// resolveModelDegradationQuestion 解析单道题：带 key 取预制题，否则按自定义题校验。
func resolveModelDegradationQuestion(raw ModelDegradationQuestion) (ModelDegradationQuestion, error) {
	taskType := strings.TrimSpace(raw.TaskType)
	key := strings.TrimSpace(raw.Key)

	if key != "" {
		switch taskType {
		case ModelDegradationTaskLogic:
			task, ok := findModelDegradationLogicTask(key)
			if !ok {
				return ModelDegradationQuestion{}, fmt.Errorf("逻辑题不存在：%s", key)
			}
			return ModelDegradationQuestion{
				TaskType: ModelDegradationTaskLogic,
				Key:      task.Key,
				Prompt:   task.Prompt,
				Expected: task.Expected,
			}, nil
		case ModelDegradationTaskRender:
			if key != modelDegradationRenderKey {
				return ModelDegradationQuestion{}, fmt.Errorf("生成题不存在：%s", key)
			}
			return ModelDegradationQuestion{
				TaskType: ModelDegradationTaskRender,
				Key:      modelDegradationRenderKey,
				Prompt:   buildModelDegradationRenderPrompt(),
			}, nil
		default:
			return ModelDegradationQuestion{}, fmt.Errorf("检测项无效：%s", raw.TaskType)
		}
	}

	prompt := strings.TrimSpace(raw.Prompt)
	if prompt == "" {
		return ModelDegradationQuestion{}, fmt.Errorf("自定义题必须填写题目内容")
	}
	if len([]rune(prompt)) > modelDegradationMaxPromptRunes {
		return ModelDegradationQuestion{}, fmt.Errorf("自定义题最多 %d 个字", modelDegradationMaxPromptRunes)
	}

	switch taskType {
	case ModelDegradationTaskLogic:
		// 逻辑题靠「回答里出现期望数字」判分，没有期望答案就没法判对错，
		// 强行跑只会让整轮被算成退化，所以这里直接要求填。
		expected := strings.TrimSpace(raw.Expected)
		if expected == "" {
			return ModelDegradationQuestion{}, fmt.Errorf("自定义逻辑题必须填写期望答案")
		}
		if _, err := strconv.ParseFloat(expected, 64); err != nil {
			return ModelDegradationQuestion{}, fmt.Errorf("期望答案必须是数字：%s", expected)
		}
		return ModelDegradationQuestion{
			TaskType: ModelDegradationTaskLogic,
			Key:      modelDegradationCustomKey,
			Prompt:   prompt,
			Expected: expected,
		}, nil
	case ModelDegradationTaskRender:
		return ModelDegradationQuestion{
			TaskType: ModelDegradationTaskRender,
			Key:      modelDegradationCustomKey,
			Prompt:   prompt,
		}, nil
	default:
		return ModelDegradationQuestion{}, fmt.Errorf("检测项无效：%s", raw.TaskType)
	}
}
