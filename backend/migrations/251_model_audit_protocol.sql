-- 上游模型降智检测：记录本次检测使用的上游协议。
--
-- 为什么要单独存这一列：同一个 Base URL 下，/chat/completions、/responses 与
-- /messages 的认证方式、请求体、响应体形状都不一样，能测什么模型也取决于协议。
-- 历史记录里没有这一项，就无法解释「同一个模型为什么换了个协议之后判定变了」，
-- 也无法复现当时打的到底是哪个端点。
--
-- 取值：'chat_completions'（OpenAI 兼容）| 'responses'（OpenAI Responses）| 'messages'（Anthropic Messages）。
-- 默认值是 chat_completions：本列落地之前的所有记录走的都是它，默认值即它们当时的真实协议，
-- 因此不需要额外回填。
ALTER TABLE model_audit_runs
    ADD COLUMN IF NOT EXISTS protocol VARCHAR(32) NOT NULL DEFAULT 'chat_completions';
