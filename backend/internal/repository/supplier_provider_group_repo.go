package repository

import "context"

func clearSupplierProviderGroupLocalMapping(ctx context.Context, exec sqlExecutor, localGroupID int64) error {
	_, err := exec.ExecContext(ctx, `
UPDATE supplier_provider_groups
SET local_group_id = NULL,
    auto_match_status = 'unmatched',
    auto_match_ignored = TRUE,
    matched_upstream_name = NULL,
    name_change_pending = FALSE,
    rate_guard_selected = FALSE,
    rate_guard_ignored = FALSE,
    rate_guard_selection_mode = '',
    updated_at = NOW()
WHERE local_group_id = $1
`, localGroupID)
	return err
}
