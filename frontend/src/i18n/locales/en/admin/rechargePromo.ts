export default {
  rechargePromo: {
    title: 'Holiday Recharge Promotion',
    description:
      'Configure tiered holiday recharge bonuses: when enabled, "recharge more, get more" within the time window; expires automatically.',
    enabled: 'Enable Holiday Recharge Promotion',
    enabledHint:
      'When enabled, grants tiered bonus credit ("recharge more, get more") within the time window; expires automatically.',
    start: 'Start Time',
    end: 'End Time',
    windowHint:
      'Leave a start or end time blank for no boundary on that side (active indefinitely once enabled, until turned off).',
    tiers: 'Bonus Tiers',
    tiersHint:
      'Each tier is a "threshold + extra bonus percent". The threshold is a floor: the amount must reach it to qualify, and settlement applies the highest tier reached.',
    tiersExample:
      'Example: with "≥100 → +5%, ≥500 → +10%, ≥1000 → +20%", recharging 300 reaches the first tier (+5%), 500 reaches the second (+10%), and 50 is below the lowest tier (no bonus). The bonus is extra credit stacked on top of the base recharge multiplier, not a price discount.',
    thresholdPlaceholder: 'Threshold',
    bonusPlaceholder: 'Bonus',
    addTier: 'Add Tier',
    removeTier: 'Remove',
    invalidRange: 'End time must not be earlier than start time',
    invalidTier: 'Enter a valid threshold and bonus percent (0-500%)',
    duplicateThreshold: 'Duplicate recharge thresholds are not allowed',
    noTiers: 'Configure at least one tier when the promotion is enabled',
    save: 'Save',
    saveSuccess: 'Holiday recharge promotion saved',
    saveFailed: 'Failed to save',
    loadFailed: 'Failed to load holiday recharge promotion config',
  },
}
