//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJudgeModelDegradationLogic(t *testing.T) {
	cases := []struct {
		name     string
		response string
		expected string
		wantPass bool
	}{
		{name: "命中整数答案", response: "9", expected: "9", wantPass: true},
		{name: "带解释仍能命中", response: "答案是 9 个球。", expected: "9", wantPass: true},
		{name: "等价小数写法", response: "4.0", expected: "4", wantPass: true},
		{name: "回答为空", response: "   ", expected: "9", wantPass: false},
		{name: "回答中没有数字", response: "无法确定", expected: "9", wantPass: false},
		{name: "答案不对", response: "8", expected: "9", wantPass: false},
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
