-- 上游模型降智检测：记录本次检测使用的推理程度（reasoning_effort）。
--
-- 为什么要单独存这一列：同一个模型在 low 与 high 下的表现差异很大，逻辑题答错
-- 可能只是推理深度不够，而不是上游降智。历史记录里没有这一项，就无法解释
-- 「同一个模型为什么上次判定正常、这次判定退化」。
--
-- 取值与源站（manxue.ai）一致：''（不指定）| low | medium | high | xhigh | max，
-- 同时也是本仓既有的 REASONING_EFFORT_LEVELS 词表的子集。
-- 空串表示请求体里不带该字段，由上游使用自身默认值 —— 这比强制填一个默认档更安全：
-- 本功能的 Base URL 是用户任意填的上游，未必接受 reasoning_effort。
ALTER TABLE model_audit_runs
    ADD COLUMN IF NOT EXISTS reasoning_effort VARCHAR(16) NOT NULL DEFAULT '';
