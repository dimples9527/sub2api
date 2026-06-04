export interface ModelSquareRateWarning {
  group_id: number
  group_name: string
  local_rate_multiplier: number
  upstream_rate_multiplier: number
}

function formatRate(value: number): string {
  if (!Number.isFinite(value)) return '-'
  return Number(value.toFixed(6)).toString()
}

export function formatModelSquareRateSyncWarning(
  warnings: ModelSquareRateWarning[] | null | undefined,
  maxVisible: number = 3
): string {
  if (!warnings?.length) return ''

  const visibleCount = Math.max(1, maxVisible)
  const visibleWarnings = warnings.slice(0, visibleCount)
  const details = visibleWarnings
    .map(
      (warning) =>
        `${warning.group_name || `#${warning.group_id}`} (upstream ${formatRate(
          warning.upstream_rate_multiplier
        )}x, local ${formatRate(warning.local_rate_multiplier)}x)`
    )
    .join(', ')
  const suffix = warnings.length > visibleWarnings.length ? ` and ${warnings.length} groups total` : ''

  return `Upstream group rates are greater than or equal to local rates: ${details}${suffix}`
}

export function visibleModelSquareRateWarnings(
  warnings: ModelSquareRateWarning[] | null | undefined,
  maxVisible: number = 3
): ModelSquareRateWarning[] {
  if (!warnings?.length) return []
  return warnings.slice(0, Math.max(1, maxVisible))
}

export function remainingModelSquareRateWarningCount(
  warnings: ModelSquareRateWarning[] | null | undefined,
  maxVisible: number = 3
): number {
  if (!warnings?.length) return 0
  return Math.max(0, warnings.length - Math.max(1, maxVisible))
}
