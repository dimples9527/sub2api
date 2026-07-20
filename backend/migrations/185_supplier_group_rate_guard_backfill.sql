WITH sole_active AS (
    SELECT local_group_id,
           MIN(id) FILTER (WHERE active = TRUE) AS group_id
    FROM supplier_provider_groups
    WHERE local_group_id IS NOT NULL
    GROUP BY local_group_id
    HAVING COUNT(*) FILTER (WHERE active = TRUE) = 1
       AND COUNT(*) FILTER (WHERE rate_guard_selected = TRUE AND active = FALSE) = 0
)
UPDATE supplier_provider_groups g
SET rate_guard_selected = TRUE,
    rate_guard_selection_mode = 'auto',
    updated_at = NOW()
FROM sole_active s
WHERE g.id = s.group_id
  AND (
      g.rate_guard_selected = FALSE
      OR g.rate_guard_selection_mode <> 'auto'
  );
