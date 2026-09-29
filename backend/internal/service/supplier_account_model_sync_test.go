package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type upstreamModelBatchRepoStub struct {
	AccountRepository
	accounts      []*Account
	credUpdates   map[int64]map[string]any
	extraUpdates  map[int64]map[string]any
	getByIDsCalls int
}

func (r *upstreamModelBatchRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	r.getByIDsCalls++
	out := make([]*Account, 0, len(ids))
	for _, id := range ids {
		for _, account := range r.accounts {
			if account.ID == id {
				out = append(out, account)
				break
			}
		}
	}
	return out, nil
}

func (r *upstreamModelBatchRepoStub) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	if r.credUpdates == nil {
		r.credUpdates = map[int64]map[string]any{}
	}
	r.credUpdates[id] = credentials
	return nil
}

func (r *upstreamModelBatchRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if r.extraUpdates == nil {
		r.extraUpdates = map[int64]map[string]any{}
	}
	r.extraUpdates[id] = updates
	return nil
}

func upstreamModelBatchTestAccount(id int64, modelMapping map[string]any) *Account {
	credentials := map[string]any{
		"api_key":  "openai-key",
		"base_url": "https://openai.example.com/v1",
	}
	if modelMapping != nil {
		credentials["model_mapping"] = modelMapping
	}
	return &Account{
		ID:          id,
		Name:        "batch-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: credentials,
	}
}

// upstreamModelBatchCompleteModelsBody 带齐「完整能力」字段（reasoning + input_modalities
// + context_window）。缺任一字段就会被判定为不完整，进而去 models.dev 拉真实数据——
// 单测里那会走网络，所以这里的响应必须自带完整能力。
const upstreamModelBatchCompleteModelsBody = `{"data":[` +
	`{"id":"gpt-5","reasoning":false,"input_modalities":["text"],"context_window":128000},` +
	`{"id":"gpt-4o","reasoning":false,"input_modalities":["text"],"context_window":128000}]}`

// upstreamModelBatchUpstream 按调用顺序返回预先排好的响应。
// 并发度必须传 1 使用：并发下 stub 自身的切片读写会 data race。
func upstreamModelBatchUpstream(bodies ...string) *httpUpstreamRecorder {
	responses := make([]*http.Response, 0, len(bodies))
	for _, body := range bodies {
		responses = append(responses, &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		})
	}
	// 兜底：队列耗尽时 Stub 会返回 nil 响应，调用方解引用会 panic，遮住真正的断言失败。
	fallback := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
	}
	return &httpUpstreamRecorder{responses: responses, resp: fallback}
}

func upstreamModelBatchTestService(repo *upstreamModelBatchRepoStub, upstream *httpUpstreamRecorder) *AccountTestService {
	return &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          upstreamModelSyncTestConfig(),
	}
}

func TestSyncUpstreamModelCatalogBatchMergePreviewDoesNotWriteWhitelist(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{
		upstreamModelBatchTestAccount(1, nil),
		upstreamModelBatchTestAccount(2, map[string]any{"gpt-4o": "gpt-4o"}),
	}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
		upstreamModelBatchCompleteModelsBody,
		upstreamModelBatchCompleteModelsBody,
	))

	// 预览：Apply=false 时只算「应用后会长什么样」，一个账号的 model_mapping 都不能动。
	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1, 2},
		Mode:        UpstreamModelBatchSyncModeMerge,
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 2, result.Success)
	require.False(t, result.Applied)
	require.Len(t, repo.credUpdates, 0)

	require.Equal(t, 2, result.Results[0].Added)
	require.Equal(t, 0, result.Results[0].Removed)
	require.Equal(t, 2, result.Results[0].FinalCount)
	require.Equal(t, 1, result.Results[1].Added)
	require.Equal(t, 0, result.Results[1].Removed)
	require.Equal(t, 2, result.Results[1].FinalCount)
}

func TestSyncUpstreamModelCatalogBatchMergeApplyKeepsCustomMapping(t *testing.T) {
	t.Parallel()

	// 账号里已有一条手写别名映射（legacy-alias → gpt-5），merge 必须原样留着。
	repo := &upstreamModelBatchRepoStub{accounts: []*Account{
		upstreamModelBatchTestAccount(1, map[string]any{"gpt-4o": "gpt-4o", "legacy-alias": "gpt-5"}),
	}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
		upstreamModelBatchCompleteModelsBody,
	))

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1},
		Mode:        UpstreamModelBatchSyncModeMerge,
		Apply:       true,
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.Equal(t, 1, result.Results[0].Added)
	require.Equal(t, 0, result.Results[0].Removed)
	require.Equal(t, 3, result.Results[0].FinalCount)
	// 手写映射条目数要如实报出来：它们两种模式下都不会被动。
	require.Equal(t, 1, result.Results[0].CustomMappingKept)

	mapping, ok := repo.credUpdates[1]["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-4o", mapping["gpt-4o"])
	require.Equal(t, "gpt-5", mapping["gpt-5"])
	require.Equal(t, "gpt-5", mapping["legacy-alias"])
}

// replace 只覆盖「白名单」那部分条目：本地独有（上游已下架）的白名单模型会被移除，
// 而手写别名映射 / 通配符规则一条都不动。
func TestSyncUpstreamModelCatalogBatchReplaceKeepsCustomMapping(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{
		upstreamModelBatchTestAccount(1, map[string]any{
			"gpt-4o":       "gpt-4o",            // 白名单：上游也有 → 保留
			"old-model":    "old-model",         // 白名单：上游没有 → 覆盖时移除
			"legacy-alias": "gpt-5",             // 手写映射 → 永远保留
			"claude-*":     "claude-3-5-sonnet", // 通配符手写规则 → 永远保留
		}),
	}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
		upstreamModelBatchCompleteModelsBody,
	))

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1},
		Mode:        UpstreamModelBatchSyncModeReplace,
		Apply:       true,
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Results[0].UpstreamTotal)
	// 上游两个模型里只有 gpt-5 是白名单里原本没有的。
	require.Equal(t, 1, result.Results[0].Added)
	// 被移除的只有 old-model：legacy-alias / claude-* 属于手写映射，不计入移除。
	require.Equal(t, 1, result.Results[0].Removed)
	require.Equal(t, 2, result.Results[0].CustomMappingKept)
	// gpt-5、gpt-4o（白名单）+ legacy-alias、claude-*（手写映射）
	require.Equal(t, 4, result.Results[0].FinalCount)

	mapping, ok := repo.credUpdates[1]["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Len(t, mapping, 4)
	require.Equal(t, "gpt-5", mapping["gpt-5"])
	require.Equal(t, "gpt-4o", mapping["gpt-4o"])
	require.Equal(t, "gpt-5", mapping["legacy-alias"])
	require.Equal(t, "claude-3-5-sonnet", mapping["claude-*"])
	require.NotContains(t, mapping, "old-model")
}

// 手写映射与上游模型同名时必须保留手写的那条，否则「模型映射不动」就成了空话。
func TestSyncUpstreamModelCatalogBatchDoesNotOverwriteCustomMappingKey(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{UpstreamModelBatchSyncModeMerge, UpstreamModelBatchSyncModeReplace} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			repo := &upstreamModelBatchRepoStub{accounts: []*Account{
				upstreamModelBatchTestAccount(1, map[string]any{"gpt-5": "my-custom-target"}),
			}}
			svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
				upstreamModelBatchCompleteModelsBody,
			))

			result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
				AccountIDs:  []int64{1},
				Mode:        mode,
				Apply:       true,
				Concurrency: 1,
			})
			require.NoError(t, err)
			// 上游的 gpt-5 被手写映射占着，只有 gpt-4o 是新增的白名单条目。
			require.Equal(t, 1, result.Results[0].Added)
			require.Equal(t, 1, result.Results[0].CustomMappingKept)

			mapping, ok := repo.credUpdates[1]["model_mapping"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "my-custom-target", mapping["gpt-5"])
			require.Equal(t, "gpt-4o", mapping["gpt-4o"])
		})
	}
}

// 预览与落库共用同一套 plan：replace 预览算出的 removed 必须等于落库会删掉的数量，
// 且预览一个字节都不能写。
func TestSyncUpstreamModelCatalogBatchReplacePreviewMatchesApply(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{
		upstreamModelBatchTestAccount(1, map[string]any{
			"gpt-4o":       "gpt-4o",
			"old-model":    "old-model",
			"legacy-alias": "gpt-5",
		}),
	}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
		upstreamModelBatchCompleteModelsBody,
	))

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1},
		Mode:        UpstreamModelBatchSyncModeReplace,
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.False(t, result.Applied)
	require.Equal(t, 1, result.Results[0].Added)
	require.Equal(t, 1, result.Results[0].Removed)
	require.Equal(t, 3, result.Results[0].FinalCount)
	require.Equal(t, 1, result.Results[0].CustomMappingKept)
	require.Empty(t, repo.credUpdates)
}

func TestSyncUpstreamModelCatalogBatchRejectsInvalidMode(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{upstreamModelBatchTestAccount(1, nil)}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(`{"data":[]}`))

	_, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs: []int64{1},
		Mode:       "overwrite",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "merge or replace")
}

func TestSyncUpstreamModelCatalogBatchDeduplicatesAccountIDs(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{upstreamModelBatchTestAccount(1, nil)}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(
		upstreamModelBatchCompleteModelsBody,
	))

	// 同一个账号被重复提交时只同步一次：否则两个 worker 会并发写同一个 credentials。
	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1, 1, 1},
		Mode:        UpstreamModelBatchSyncModeMerge,
		Apply:       true,
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Results, 1)
}

func TestSyncUpstreamModelCatalogBatchUpstreamFailureDoesNotExposeBody(t *testing.T) {
	t.Parallel()

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"sk-secret-key-is-invalid"}}`)),
	}}
	repo := &upstreamModelBatchRepoStub{accounts: []*Account{upstreamModelBatchTestAccount(1, nil)}}
	svc := upstreamModelBatchTestService(repo, upstream)

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{1},
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, UpstreamModelBatchSyncStatusFailed, result.Results[0].Status)
	// 上游响应体不得外泄（与单账号接口同一条安全约束）。
	require.NotContains(t, result.Results[0].ErrorMessage, "sk-secret-key-is-invalid")
	require.Len(t, repo.credUpdates, 0)
}

func TestSyncUpstreamModelCatalogBatchReportsUnknownAccount(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(`{"data":[]}`))

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{
		AccountIDs:  []int64{99},
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, UpstreamModelBatchSyncStatusNotFound, result.Results[0].Status)
}

func TestSyncUpstreamModelCatalogBatchEmptyAccountIDs(t *testing.T) {
	t.Parallel()

	repo := &upstreamModelBatchRepoStub{accounts: []*Account{}}
	svc := upstreamModelBatchTestService(repo, upstreamModelBatchUpstream(`{"data":[]}`))

	result, err := svc.SyncUpstreamModelCatalogBatch(context.Background(), UpstreamModelBatchSyncInput{})
	require.NoError(t, err)
	require.Equal(t, 0, result.Total)
	require.Equal(t, UpstreamModelBatchSyncModeMerge, result.Mode)
	require.Empty(t, result.Results)
	// 没有目标账号时不应该去查库。
	require.Equal(t, 0, repo.getByIDsCalls)
}
