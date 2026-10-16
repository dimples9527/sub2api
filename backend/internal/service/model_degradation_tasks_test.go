//go:build unit

package service

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildModelDegradationRenderPrompt(t *testing.T) {
	prompt := buildModelDegradationRenderPrompt()

	// 占位符必须被替换干净，否则字面量会被当成题目发给模型。
	require.NotContains(t, prompt, "{scene}")
	require.NotContains(t, prompt, "{nonce}")
	require.Contains(t, prompt, "鹈鹕骑自行车")

	// 场景必须来自场景池。
	foundScene := false
	for _, scene := range modelDegradationRenderScenes {
		if strings.Contains(prompt, "本次场景："+scene) {
			foundScene = true
			break
		}
	}
	require.True(t, foundScene, "题面里的场景必须来自场景池：%s", prompt)

	// 校验码必须是 8 位十六进制大写（与 manxue-ai 的 token_hex(4).upper() 一致）。
	nonceMatch := regexp.MustCompile(`校验码：([0-9A-F]{8})。`).FindStringSubmatch(prompt)
	require.NotNil(t, nonceMatch, "题面必须带 8 位十六进制大写校验码：%s", prompt)
}

func TestModelDegradationRenderPromptDisplay(t *testing.T) {
	// 展示版题面不能漏出 {} 占位符，否则「查看题目」里看着像坏了。
	display := modelDegradationRenderPromptDisplay()
	require.NotContains(t, display, "{scene}")
	require.NotContains(t, display, "{nonce}")
	require.Contains(t, display, "鹈鹕骑自行车")
}

func TestJudgeModelDegradationLogic(t *testing.T) {
	// 判分口径照搬 manxue-ai 的 candy_passes：只认独立出现的期望数字。
	cases := []struct {
		name     string
		response string
		expected string
		wantPass bool
	}{
		{name: "命中整数答案", response: "9", expected: "9", wantPass: true},
		{name: "带解释仍能命中", response: "答案是 9 个球。", expected: "9", wantPass: true},
		{name: "紧贴中文算独立", response: "最少取出21颗", expected: "21", wantPass: true},
		{name: "句读符号不影响", response: "取出 21。", expected: "21", wantPass: true},
		{name: "期望值归一化后命中", response: "答案是 4。", expected: "4.0", wantPass: true},
		{name: "回答为空", response: "   ", expected: "9", wantPass: false},
		{name: "回答中没有数字", response: "无法确定", expected: "9", wantPass: false},
		{name: "答案不对", response: "8", expected: "9", wantPass: false},
		{name: "小数写法不算命中", response: "4.0", expected: "4", wantPass: false},
		{name: "小数续写不算命中", response: "21.5", expected: "21", wantPass: false},
		{name: "紧贴字母不算命中", response: "x21", expected: "21", wantPass: false},
		{name: "紧贴数字不算命中", response: "2121", expected: "21", wantPass: false},
		{name: "负号前缀不算独立", response: "-21", expected: "21", wantPass: false},
		{name: "全角数字续写不算命中", response: "21２", expected: "21", wantPass: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			passed, detail := judgeModelDegradationLogic(tc.response, tc.expected)
			require.Equal(t, tc.wantPass, passed)
			require.NotEmpty(t, detail)
		})
	}
}

func TestNormalizeModelDegradationSource(t *testing.T) {
	// 空白差异属于排版噪声，不应影响「是否同一份源码」的判断。
	spaced := "<svg>\n  <circle/>\n</svg>"
	compact := "<svg><circle/></svg>"
	require.Equal(t, normalizeModelDegradationSource(spaced), normalizeModelDegradationSource(compact))

	// 不同的图形结构必须得到不同指纹，否则查重会把正常作品误判为复用。
	other := "<svg><ellipse/></svg>"
	require.NotEqual(t, normalizeModelDegradationSource(compact), normalizeModelDegradationSource(other))
}

func TestResolveModelDegradationQuestions_Presets(t *testing.T) {
	// 不传题目时退回旧行为：选中哪个类型就跑该类型的第一道预制题（单选）。
	questions, err := resolveModelDegradationQuestions([]string{ModelDegradationTaskLogic}, nil)
	require.NoError(t, err)
	require.Len(t, questions, 1)
	require.Equal(t, ModelDegradationTaskLogic, questions[0].TaskType)
	require.Equal(t, modelDegradationLogicTasks[0].Key, questions[0].Key)
	require.NotEmpty(t, questions[0].Expected)

	// 两个都不传时逻辑题 + 生成题各一道，且生成题排在最后（它最慢）。
	questions, err = resolveModelDegradationQuestions(nil, nil)
	require.NoError(t, err)
	require.Len(t, questions, 2)
	require.Equal(t, ModelDegradationTaskLogic, questions[0].TaskType)
	require.Equal(t, ModelDegradationTaskRender, questions[len(questions)-1].TaskType)
}

func TestResolveModelDegradationQuestions_ByKey(t *testing.T) {
	// 预制题按 key 取库里的正文与期望答案，不信调用方传进来的那份。
	questions, err := resolveModelDegradationQuestions(nil, []ModelDegradationQuestion{{
		TaskType: ModelDegradationTaskLogic,
		Key:      "candy-shape-flavor",
		Prompt:   "伪造的正文",
		Expected: "999",
	}})
	require.NoError(t, err)
	require.Len(t, questions, 1)
	require.Equal(t, "21", questions[0].Expected)
	require.Contains(t, questions[0].Prompt, "糖果")
}

func TestResolveModelDegradationQuestions_Custom(t *testing.T) {
	questions, err := resolveModelDegradationQuestions(nil, []ModelDegradationQuestion{{
		TaskType: ModelDegradationTaskLogic,
		Prompt:   "1+1 等于几？只回答一个数字。",
		Expected: "2",
	}})
	require.NoError(t, err)
	require.Len(t, questions, 1)
	require.Equal(t, modelDegradationCustomKey, questions[0].Key)
	require.Equal(t, "2", questions[0].Expected)

	// 生成题排在逻辑题之后，即使提交时顺序相反。
	questions, err = resolveModelDegradationQuestions(nil, []ModelDegradationQuestion{
		{TaskType: ModelDegradationTaskRender, Key: modelDegradationRenderKey},
		{TaskType: ModelDegradationTaskLogic, Key: "candy-shape-flavor"},
	})
	require.NoError(t, err)
	require.Len(t, questions, 2)
	require.Equal(t, ModelDegradationTaskLogic, questions[0].TaskType)
	require.Equal(t, ModelDegradationTaskRender, questions[1].TaskType)

	// 同一道预制题提交两次只跑一次。
	questions, err = resolveModelDegradationQuestions(nil, []ModelDegradationQuestion{
		{TaskType: ModelDegradationTaskLogic, Key: "candy-shape-flavor"},
		{TaskType: ModelDegradationTaskLogic, Key: "candy-shape-flavor"},
	})
	require.NoError(t, err)
	require.Len(t, questions, 1)
}

func TestResolveModelDegradationQuestions_Rejects(t *testing.T) {
	cases := []struct {
		name      string
		questions []ModelDegradationQuestion
	}{
		{name: "未知逻辑题", questions: []ModelDegradationQuestion{{TaskType: ModelDegradationTaskLogic, Key: "nope"}}},
		{name: "未知生成题", questions: []ModelDegradationQuestion{{TaskType: ModelDegradationTaskRender, Key: "nope"}}},
		{name: "检测项无效", questions: []ModelDegradationQuestion{{TaskType: "video", Key: "candy-shape-flavor"}}},
		{name: "自定义题没正文", questions: []ModelDegradationQuestion{{TaskType: ModelDegradationTaskRender}}},
		{name: "自定义逻辑题没期望答案", questions: []ModelDegradationQuestion{{TaskType: ModelDegradationTaskLogic, Prompt: "1+1"}}},
		{name: "期望答案不是数字", questions: []ModelDegradationQuestion{{TaskType: ModelDegradationTaskLogic, Prompt: "1+1", Expected: "二"}}},
		{name: "同类两道自定义题", questions: []ModelDegradationQuestion{
			{TaskType: ModelDegradationTaskLogic, Prompt: "1+1", Expected: "2"},
			{TaskType: ModelDegradationTaskLogic, Prompt: "2+2", Expected: "4"},
		}},
		// 上限必须挡住：单轮整体超时是按「各题上限之和」推导的，放开数量会让它失效。
		{name: "逻辑题超上限", questions: []ModelDegradationQuestion{
			{TaskType: ModelDegradationTaskLogic, Key: "candy-shape-flavor"},
			{TaskType: ModelDegradationTaskLogic, Prompt: "1+1", Expected: "2"},
		}},
		{name: "生成题超上限", questions: []ModelDegradationQuestion{
			{TaskType: ModelDegradationTaskRender, Key: modelDegradationRenderKey},
			{TaskType: ModelDegradationTaskRender, Prompt: "画一棵树"},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveModelDegradationQuestions(nil, tc.questions)
			require.Error(t, err)
		})
	}
}
