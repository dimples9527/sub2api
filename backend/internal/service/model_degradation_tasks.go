package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 检测项类型：logic = 有唯一答案的逻辑题，render = 生成 HTML 作品的开放题。
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
// 题目要求只回答数字，避免模型输出长解释后再夹带答案，导致判分被无关数字干扰。
var modelDegradationLogicTasks = []modelDegradationLogicTask{
	{
		Key:   "candy-shape-flavor",
		Title: "糖果摸取：保证不同形状的苹果味和桃子味",
		Prompt: modelDegradationNoToolsPrefix + "在一个黑色的袋子里放有三种口味的糖果（苹果味、桃子味、西瓜味），" +
			"每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。" +
			"不同口味的糖和不同形状的数量统计如下表（行是形状，列是口味）：" +
			"圆形 —— 苹果味 7、桃子味 9、西瓜味 8；五角星形 —— 苹果味 7、桃子味 6、西瓜味 4。" +
			"参赛者需要在活动前决定摸出的糖果数目。最少取出多少个糖果，" +
			"才能保证手中同时拥有不同形状的苹果味和桃子味的糖？" +
			"（手中有圆形苹果味配五角星桃子味，或者有圆形桃子味配五角星苹果味，都满足要求）" +
			"只回答一个数字，不要解释。",
		Expected: "21",
	},
}

// 生成题（唯一的预制题）的标识与短标题。
const (
	modelDegradationRenderKey   = "pelican-bicycle"
	modelDegradationRenderTitle = "画一只骑自行车的鹈鹕"
)

// modelDegradationRenderPrompt 生成题的题目。
//
// 与逻辑题不同，这题没有唯一答案，判分看的是「输出是否退化」而非画得好不好，
// 因此题目要固定到足以横向比较：同一 prompt 反复提交，源码应当每次不同。
//
// 尾句不是客套：实测有上游（编码代理类模型）把「创建一个HTML」当成建文件任务，
// 回一句「需要先查看当前工作区结构，然后创建一个可直接打开的 HTML 文件」就收工 ——
// 我们是非流式单轮调用，不会再有下一轮，于是整题判成「回答中没有 SVG / Canvas 代码」。
// 所以必须显式要求：本轮直接给出源码，不许建文件、不许调工具、不许只描述计划。
const modelDegradationRenderPrompt = modelDegradationNoToolsPrefix +
	"创建一个HTML，内容是SVG绘制一个鹈鹕骑自行车的2D动画，你不需要任何测试。" +
	"请直接在本次回复里输出完整的 HTML 源码（以 <!DOCTYPE html> 开头），" +
	"不要创建文件、不要调用工具、不要询问确认，也不要只描述计划或解释做法。"

// modelDegradationCustomKey 自定义题在库里的标识。多道自定义题共用它 ——
// 它只用来区分「预制 / 自定义」，题目全文另有 prompt 字段。
const modelDegradationCustomKey = "custom"

// 每轮检测各类型最多能跑几道题。
//
// 单选：逻辑题与生成题在一轮里都只跑一道。这两个上限不是为了好看：
// 单轮的整体超时是按「各题上限之和」推导出来的（见 modelDegradationTotalTimeout），
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
// 判分口径与 channel_monitor_challenge.go 的 validateChallenge 一致：从回答里
// 提取所有数字，只要出现期望值即算通过。用数值比较而非字符串比较，是为了让
// 「4」「4.0」「答案是 4 小时」这类等价写法都能通过。
func judgeModelDegradationLogic(response, expected string) (bool, string) {
	if strings.TrimSpace(response) == "" {
		return false, "上游没有返回内容"
	}

	matches := modelDegradationNumberRegex.FindAllString(response, -1)
	for _, m := range matches {
		if modelDegradationNumbersEqual(m, expected) {
			return true, "命中答案 " + expected
		}
	}

	if len(matches) == 0 {
		return false, "回答中没有可识别的数字（期望 " + expected + "）"
	}
	return false, "未出现期望答案 " + expected + "，回答中的数字：" + strings.Join(matches, ", ")
}

// modelDegradationNumbersEqual 按数值比较两个数字字符串。
func modelDegradationNumbersEqual(a, b string) bool {
	left, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return false
	}
	right, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return false
	}
	return left == right
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
func ModelDegradationRenderTaskOptions() []ModelDegradationTaskOption {
	return []ModelDegradationTaskOption{{
		TaskType: ModelDegradationTaskRender,
		Key:      modelDegradationRenderKey,
		Title:    modelDegradationRenderTitle,
		Prompt:   modelDegradationRenderPrompt,
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
			Prompt:   modelDegradationRenderPrompt,
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
				Prompt:   modelDegradationRenderPrompt,
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
