-- availability_7d 原为 NOT NULL DEFAULT 0，使「上游未上报可用率」与「真实 0%」不可区分，
-- 监控绑定页会把没有采样的监控项显示成红色 0.00%。放开 NOT NULL 后由同步侧写入 NULL 表示无数据。
-- 历史行仍是 0，只有重新同步后才能区分。
ALTER TABLE supplier_provider_monitor_targets ALTER COLUMN availability_7d DROP NOT NULL;
ALTER TABLE supplier_provider_monitor_targets ALTER COLUMN availability_7d DROP DEFAULT;

ALTER TABLE supplier_provider_monitor_samples ALTER COLUMN availability_7d DROP NOT NULL;
ALTER TABLE supplier_provider_monitor_samples ALTER COLUMN availability_7d DROP DEFAULT;
