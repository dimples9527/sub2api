-- 上游模型降智检测：记录本次检测所用平台 API Key 绑定的分组（能识别时）。
--
-- 为什么要单独存这一列：检测的 Base URL 允许填平台自身地址、API Key 允许填平台
-- 自己签发的 Key —— 这种「测自己链路」的场景下，分组决定了平台会把请求转发到哪些
-- 渠道，是解释「同一个模型为什么在不同记录里表现不同」的关键维度。
--
-- 识别方式：拿请求里填的明文 Key 去平台的密钥表匹配（key 列明文存储），
-- 只匹配发起者本人名下的 Key。匹配不上（填的是第三方上游 Key）时两列都留空 ——
-- 那是常态，不是异常。
-- group_name 存发起时的名字快照：分组会被改名或删除，历史记录要保留当时的语义。
ALTER TABLE model_audit_runs
    ADD COLUMN IF NOT EXISTS group_id BIGINT,
    ADD COLUMN IF NOT EXISTS group_name TEXT NOT NULL DEFAULT '';
