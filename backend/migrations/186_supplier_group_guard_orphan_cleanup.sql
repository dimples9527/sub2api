UPDATE supplier_provider_groups g
SET local_group_id = NULL,
    auto_match_status = 'unmatched',
    auto_match_ignored = TRUE,
    matched_upstream_name = NULL,
    name_change_pending = FALSE,
    rate_guard_selected = FALSE,
    rate_guard_selection_mode = '',
    updated_at = NOW()
WHERE g.local_group_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM groups lg
      WHERE lg.id = g.local_group_id
        AND lg.deleted_at IS NULL
  );
