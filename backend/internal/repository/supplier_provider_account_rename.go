package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 上游账号身份（upstream_account_key）的稳定性维护。
//
// supplier_provider_accounts 的唯一键是 (provider_id, upstream_account_key)，同步靠它找行、
// 绑定（local_account_id）也挂在这行上。所以 key 一旦跟着「名字」走，上游改名就会让
// ReplaceAccounts 把旧行判成「上游已删除」、再插一条新行，绑定留在旧行上、新行永远匹配不到
// 本地账号；表现出来就是列表里一条「已删除」+ 一条未匹配的新行，绑定的本地账号同时脱离
// 健康守护与择优调度的管辖（候选查询只取 active=TRUE 且能匹配到本地账号的行）。
//
// 两层防护：
//  1. migrateSupplierAccountKeys（根治）：newapi 解析过去拿账号名当 key，现在改取上游 token id。
//     这里把存量行原地换成 id，映射来自「同一条上游记录里的 name 与 id」，是确定的。
//  2. planSupplierAccountRenames（兜底）：上游确实给不出稳定 id 时，key 只能继续是名字。
//     这种情况用「同分组内一对一 + 倍率一致 + 名字相似 + 旧行有绑定」识别改名，
//     让旧行原地接管新 key。⚠️ 这是推断不是事实，判据必须严格，否则会把绑定迁到别的账号上。

type supplierPreviousAccount struct {
	id             int64
	key            string
	name           string
	groupKey       string
	rateMultiplier float64
	localAccountID sql.NullInt64
	active         bool
}

type supplierAccountRename struct {
	previousID int64
	item       service.SupplierProviderRemoteAccount
}

// supplierRemoteAccountKey 与 ReplaceAccounts 的 upsert 共用同一套 key 推导，
// 避免「识别改名时算出来的 key」和「真正写库的 key」不一致。
func supplierRemoteAccountKey(item service.SupplierProviderRemoteAccount) string {
	key := strings.TrimSpace(item.Key)
	if key != "" {
		return key
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func loadSupplierPreviousAccounts(ctx context.Context, tx *sql.Tx, providerID int64) ([]supplierPreviousAccount, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, upstream_account_key, name, COALESCE(group_key, ''), COALESCE(rate_multiplier, 0),
       local_account_id, active
FROM supplier_provider_accounts
WHERE provider_id = $1
FOR UPDATE`, providerID)
	if err != nil {
		return nil, fmt.Errorf("query previous supplier accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	previous := make([]supplierPreviousAccount, 0)
	for rows.Next() {
		var row supplierPreviousAccount
		if err := rows.Scan(&row.id, &row.key, &row.name, &row.groupKey, &row.rateMultiplier, &row.localAccountID, &row.active); err != nil {
			return nil, fmt.Errorf("scan previous supplier account: %w", err)
		}
		previous = append(previous, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate previous supplier accounts: %w", err)
	}
	return previous, nil
}

// migrateSupplierAccountKeys 把「历史上拿账号名当 key」的行原地换成上游稳定 id。
//
// newapi 类型的供应商过去用 name 当 upstream_account_key（见 parseSupplierNewAPIAccounts），
// 上游一改名 key 就变，行会被判成「已删除 + 新增」，绑定的本地账号随即断链。
// 解析层改成取上游 token id 之后，这里负责把存量行迁过去：映射来自「同一条上游记录里的
// name 与 id」，是确定的，不需要按名字跨轮猜测。
//
// 必须在 upsert 之前执行：先 upsert 会插一条 key=id 的新行，老行当场被判成已删除，迁移就晚了。
// 跳过条件：本轮同名记录多于一条（无法确定迁哪条）、目标 id 行已存在（已经在用 id）、
// 或快照里没有 key 等于该名字的老行。
func migrateSupplierAccountKeys(ctx context.Context, tx *sql.Tx, providerID int64, previous []supplierPreviousAccount, existingKeys map[string]struct{}, items []service.SupplierProviderRemoteAccount) error {
	nameCounts := make(map[string]int, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			nameCounts[name]++
		}
	}

	// previous 里 key 唯一，先按 key 建索引，避免逐条查库。
	previousIndexByKey := make(map[string]int, len(previous))
	for index, row := range previous {
		previousIndexByKey[row.key] = index
	}

	for _, item := range items {
		key := supplierRemoteAccountKey(item)
		name := strings.TrimSpace(item.Name)
		// key == name 说明上游没给稳定 id，维持原样。
		if key == "" || name == "" || key == name {
			continue
		}
		// 本轮同名记录多于一条时无法确定该迁哪一条，留到人工处理。
		if nameCounts[name] != 1 {
			continue
		}
		if _, exists := existingKeys[key]; exists {
			continue
		}
		index, ok := previousIndexByKey[name]
		if !ok {
			continue
		}
		row := previous[index]
		if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_accounts
SET upstream_account_key = $2, updated_at = NOW()
WHERE id = $1`, row.id, key); err != nil {
			return fmt.Errorf("migrate supplier account key: %w", err)
		}
		// 快照同步改掉，否则后面的改名识别会把「key 迁移」误看成「旧行消失 + 新行出现」。
		delete(existingKeys, row.key)
		previous[index].key = key
		existingKeys[key] = struct{}{}
	}
	return nil
}

// planSupplierAccountRenames 在 upsert 之前算出「哪些旧行其实是改了名」。
// existingKeys 是同步前该供应商名下的全部 key（含已下线行）：只在它之外的 key 才算「本轮新增」，
// 否则一条历史上删掉又出现的记录会被误当成改名。
func planSupplierAccountRenames(previous []supplierPreviousAccount, items []service.SupplierProviderRemoteAccount, existingKeys map[string]struct{}) []supplierAccountRename {
	currentItems := make(map[string]service.SupplierProviderRemoteAccount, len(items))
	for _, item := range items {
		key := supplierRemoteAccountKey(item)
		if key == "" {
			continue
		}
		currentItems[key] = item
	}

	appearedByGroup := make(map[string][]string)
	for key, item := range currentItems {
		if _, exists := existingKeys[key]; exists {
			continue
		}
		groupKey := normalizeSupplierAccountGroupKey(item.GroupKey)
		// 上游没给分组 key 时无法确认「同一分组内一删一增」，直接放弃配对。
		if groupKey == "" {
			continue
		}
		appearedByGroup[groupKey] = append(appearedByGroup[groupKey], key)
	}

	disappearedByGroup := make(map[string][]supplierPreviousAccount)
	for _, row := range previous {
		if !row.active {
			continue
		}
		if _, exists := currentItems[row.key]; exists {
			continue
		}
		// 没有绑定本地账号的行，改名迁移没有意义，按「删除 + 新增」处理即可。
		if !row.localAccountID.Valid {
			continue
		}
		groupKey := normalizeSupplierAccountGroupKey(row.groupKey)
		if groupKey == "" {
			continue
		}
		disappearedByGroup[groupKey] = append(disappearedByGroup[groupKey], row)
	}

	renames := make([]supplierAccountRename, 0, len(disappearedByGroup))
	for groupKey, disappeared := range disappearedByGroup {
		appeared := appearedByGroup[groupKey]
		// 同组内必须一对一，否则无法确定谁改成了谁，宁可不迁移。
		if len(disappeared) != 1 || len(appeared) != 1 {
			continue
		}
		row := disappeared[0]
		item := currentItems[appeared[0]]
		if !supplierAccountRatesEqual(row.rateMultiplier, item.RateMultiplier) {
			continue
		}
		if !supplierAccountNamesLookRenamed(row.name, item.Name) {
			continue
		}
		renames = append(renames, supplierAccountRename{previousID: row.id, item: item})
	}
	// map 遍历顺序不稳定，排序让执行顺序与结果可预期。
	sort.Slice(renames, func(i, j int) bool { return renames[i].previousID < renames[j].previousID })
	return renames
}

func supplierAccountRatesEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// supplierAccountNamesLookRenamed 判断两个上游账号名是否指向同一个账号。
// 归一化后相同、或互为前缀、或首 2 字符与末 2 字符都相同，都算改名；
// 与 supplierProviderSuspectedLocalAccountSimilaritySQL 的判定保持一致。
func supplierAccountNamesLookRenamed(previousName, currentName string) bool {
	previous := normalizeSupplierAccountName(previousName)
	current := normalizeSupplierAccountName(currentName)
	if previous == "" || current == "" {
		return false
	}
	if previous == current {
		return true
	}
	if strings.HasPrefix(previous, current) || strings.HasPrefix(current, previous) {
		return true
	}
	previousRunes := []rune(previous)
	currentRunes := []rune(current)
	if len(previousRunes) < 2 || len(currentRunes) < 2 {
		return false
	}
	return previousRunes[0] == currentRunes[0] &&
		previousRunes[1] == currentRunes[1] &&
		previousRunes[len(previousRunes)-1] == currentRunes[len(currentRunes)-1] &&
		previousRunes[len(previousRunes)-2] == currentRunes[len(currentRunes)-2]
}

// normalizeSupplierAccountGroupKey 把分组 key 归一化成配对键：上游偶尔会改大小写或多补空格，
// 同一分组不该因为这种差异配不上。空 key 表示「不知道属于哪个分组」，不参与配对。
func normalizeSupplierAccountGroupKey(groupKey string) string {
	return strings.ToLower(strings.TrimSpace(groupKey))
}

// normalizeSupplierAccountName 与 SQL 侧 supplierProviderLocalAccountNormalizeSQL 等价：
// 小写后只保留字母与数字。
func normalizeSupplierAccountName(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
