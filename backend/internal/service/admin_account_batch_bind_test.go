package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// batchBindAccountRepo 只实现批量绑定分组真正走到的仓储方法，其余交给内嵌接口占位。
// 嵌入接口而不是手写全部方法：AccountRepository 很大，写全反而会掩盖「测试到底覆盖了哪条路径」。
type batchBindAccountRepo struct {
	AccountRepository
	accounts     map[int64]*Account
	bindCalls    map[int64][]int64
	bindErrByID  map[int64]error
	bulkUpdateID []int64
}

func (r *batchBindAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	out := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := r.accounts[id]; ok {
			out = append(out, account)
		}
	}
	return out, nil
}

func (r *batchBindAccountRepo) BulkUpdate(_ context.Context, ids []int64, _ AccountBulkUpdate) (int64, error) {
	r.bulkUpdateID = append(r.bulkUpdateID, ids...)
	return int64(len(ids)), nil
}

func (r *batchBindAccountRepo) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	if err := r.bindErrByID[accountID]; err != nil {
		return err
	}
	if r.bindCalls == nil {
		r.bindCalls = make(map[int64][]int64)
	}
	r.bindCalls[accountID] = append([]int64(nil), groupIDs...)
	return nil
}

type batchBindGroupRepo struct {
	GroupRepository
	missing map[int64]struct{}
}

func (r *batchBindGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	if _, missing := r.missing[id]; missing {
		return nil, ErrGroupNotFound
	}
	return &Group{ID: id, Name: fmt.Sprintf("分组 %d", id)}, nil
}

func newBatchBindService(accounts ...*Account) (*adminServiceImpl, *batchBindAccountRepo) {
	repo := &batchBindAccountRepo{accounts: make(map[int64]*Account)}
	for _, account := range accounts {
		repo.accounts[account.ID] = account
	}
	svc := &adminServiceImpl{accountRepo: repo, groupRepo: &batchBindGroupRepo{}}
	return svc, repo
}

func openAIBatchAccount(id int64, groupIDs ...int64) *Account {
	return &Account{
		ID:       id,
		Name:     fmt.Sprintf("account-%d", id),
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		GroupIDs: groupIDs,
	}
}

func TestBatchBindAccountGroupsAppendsToExistingGroups(t *testing.T) {
	svc, repo := newBatchBindService(openAIBatchAccount(11, 10))

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11},
		GroupIDs:   []int64{20, 30},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Bound)
	require.Equal(t, 0, result.Failed)
	// 追加语义：原有分组 10 必须留在写入结果里，不能被替换掉。
	require.Equal(t, []int64{10, 20, 30}, repo.bindCalls[11])
	require.Equal(t, []int64{10, 20, 30}, result.Results[0].GroupIDs)
	require.Equal(t, []int64{20, 30}, result.Results[0].Added)
	require.Equal(t, "bound", result.Results[0].Status)
}

func TestBatchBindAccountGroupsSkipsAccountsAlreadyBound(t *testing.T) {
	svc, repo := newBatchBindService(openAIBatchAccount(11, 10, 20))

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11},
		GroupIDs:   []int64{20},
	})

	require.NoError(t, err)
	require.Equal(t, 0, result.Bound)
	require.Equal(t, 1, result.Unchanged)
	// 无变更时不能写库：否则会白刷一次绑定并投递一次调度事件。
	require.Empty(t, repo.bindCalls)
	require.Empty(t, repo.bulkUpdateID)
	require.Equal(t, "unchanged", result.Results[0].Status)
	require.Empty(t, result.Results[0].Added)
}

func TestBatchBindAccountGroupsIsolatesPerAccountFailure(t *testing.T) {
	svc, repo := newBatchBindService(
		openAIBatchAccount(11, 10),
		openAIBatchAccount(12),
		openAIBatchAccount(13),
	)
	repo.bindErrByID = map[int64]error{12: errors.New("分组校验失败")}

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11, 12, 13},
		GroupIDs:   []int64{20},
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.Bound)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Results, 3)

	failed := result.Results[1]
	require.Equal(t, int64(12), failed.AccountID)
	require.Equal(t, "failed", failed.Status)
	require.Equal(t, "分组校验失败", failed.Error)
	// 一个账号被拦下不能连带整批失败：后面那个账号仍然要写成功。
	require.Equal(t, []int64{10, 20}, repo.bindCalls[11])
	require.Equal(t, []int64{20}, repo.bindCalls[13])
}

func TestBatchBindAccountGroupsReportsMissingAccount(t *testing.T) {
	svc, _ := newBatchBindService(openAIBatchAccount(11))

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11, 99},
		GroupIDs:   []int64{20},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Bound)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, "账号不存在", result.Results[1].Error)
}

func TestBatchBindAccountGroupsRejectsEmptyAndOversizedInput(t *testing.T) {
	svc, _ := newBatchBindService(openAIBatchAccount(11))

	_, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		GroupIDs: []int64{20},
	})
	require.ErrorContains(t, err, "至少需要选择一个账号")

	_, err = svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11},
	})
	require.ErrorContains(t, err, "至少需要选择一个分组")

	oversized := make([]int64, BatchBindAccountGroupsMaxAccounts+1)
	for i := range oversized {
		oversized[i] = int64(i + 1)
	}
	_, err = svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: oversized,
		GroupIDs:   []int64{20},
	})
	require.ErrorContains(t, err, "一次最多绑定")
}

func TestBatchBindAccountGroupsDeduplicatesInputs(t *testing.T) {
	svc, repo := newBatchBindService(openAIBatchAccount(11, 10))

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11, 11, 0},
		GroupIDs:   []int64{20, 20, 30},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	require.Equal(t, []int64{10, 20, 30}, repo.bindCalls[11])
}

func TestBatchBindAccountGroupsReportsGroupNotFound(t *testing.T) {
	svc, repo := newBatchBindService(openAIBatchAccount(11))
	svc.groupRepo = &batchBindGroupRepo{missing: map[int64]struct{}{20: {}}}

	result, err := svc.BatchBindAccountGroups(context.Background(), &BatchBindAccountGroupsInput{
		AccountIDs: []int64{11},
		GroupIDs:   []int64{20},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, "failed", result.Results[0].Status)
	require.Empty(t, repo.bindCalls)
}
