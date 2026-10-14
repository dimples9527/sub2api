/**
 * 模型降智检测 API。
 *
 * 凭据（Base URL / API Key）由用户在每次检测时提交，服务端只用一次、不保存，
 * 因此这里的请求体里带 api_key 是设计如此，不会在响应里回显。
 */

import { apiClient } from './client'

export type ModelDegradationTaskType = 'logic' | 'render'
export type ModelDegradationRunStatus = 'running' | 'completed' | 'failed' | 'cancelled'
export type ModelDegradationVerdict = 'normal' | 'degraded' | 'unknown'
export type ModelDegradationItemVerdict = 'ok' | 'degraded' | 'unknown' | 'generated'

/**
 * 检测可选的上游协议。
 *
 * - chat_completions：OpenAI 兼容的 /chat/completions（Bearer 认证）
 * - responses：OpenAI Responses API 的 /responses（Bearer 认证）
 * - messages：Anthropic Messages API 的 /messages（x-api-key + anthropic-version）
 *
 * 后端 modelDegradationProtocols 是同一份清单，改动时要一起改。
 */
export const MODEL_AUDIT_PROTOCOLS = ['chat_completions', 'responses', 'messages'] as const

/** 检测使用的上游协议。 */
export type ModelDegradationProtocol = (typeof MODEL_AUDIT_PROTOCOLS)[number]

/**
 * 检测可选的思考等级档位（不含「不指定」）。
 *
 * 与源站（manxue.ai）的下拉一致，也是本仓 REASONING_EFFORT_LEVELS 的子集；
 * 后端 modelDegradationReasoningEfforts 是同一份清单，改动时要一起改。
 */
export const MODEL_AUDIT_REASONING_EFFORTS = ['low', 'medium', 'high', 'xhigh', 'max'] as const

/**
 * 检测可选的思考等级。
 *
 * 空串表示不指定 —— 请求体里不带该字段，由上游使用自身默认值。
 */
export type ModelDegradationReasoningEffort = '' | (typeof MODEL_AUDIT_REASONING_EFFORTS)[number]

export interface ModelDegradationItem {
  task_type: ModelDegradationTaskType
  task_key: string
  prompt: string
  expected: string
  response: string
  passed: boolean
  verdict: ModelDegradationItemVerdict | ''
  detail: string
  source_hash: string
  duration_ms: number
}

export interface ModelDegradationRun {
  id: number
  base_url: string
  model_id: string
  /** 本次检测使用的上游协议。 */
  protocol: ModelDegradationProtocol
  /** 本次检测使用的思考等级；空串表示未指定。 */
  reasoning_effort: ModelDegradationReasoningEffort
  status: ModelDegradationRunStatus
  verdict: ModelDegradationVerdict
  logic_passed: number
  logic_total: number
  render_verdict: ModelDegradationItemVerdict | ''
  error_message: string
  duration_ms: number
  created_at: string
  completed_at?: string
  items?: ModelDegradationItem[]
}

export interface StartModelDegradationPayload {
  base_url: string
  api_key: string
  model_id: string
  /** 上游协议；决定端点、认证头与响应体解析方式。 */
  protocol: ModelDegradationProtocol
  /** 思考等级；不传表示不指定，由上游使用默认值。 */
  reasoning_effort?: ModelDegradationReasoningEffort
  tasks?: ModelDegradationTaskType[]
  /** 本次要跑的具体题目；不传则按 tasks 跑该类型下的全部预制题。 */
  questions?: ModelDegradationQuestionPayload[]
}

/** 题库里的一道预制题。 */
export interface ModelDegradationTaskOption {
  task_type: ModelDegradationTaskType
  /** 预制题的标识；提交时原样回传即可。 */
  key: string
  /** 短标题，铺下拉选项用。 */
  title: string
  /** 题目全文。 */
  prompt: string
  /** 逻辑题的期望答案；生成题为空。 */
  expected: string
}

/** 内置题库，按检测项分组。 */
export interface ModelDegradationTaskBank {
  logic: ModelDegradationTaskOption[]
  render: ModelDegradationTaskOption[]
}

/**
 * 一道要提交的题。
 *
 * 带 key 表示选的是预制题（服务端按 key 取正文与期望答案）；
 * key 为空则视为自定义题，正文取 prompt。
 */
export interface ModelDegradationQuestionPayload {
  task_type: ModelDegradationTaskType
  key?: string
  prompt?: string
  /** 自定义逻辑题的期望答案（数字）；预制题与生成题都不填。 */
  expected?: string
}

export interface ListUpstreamModelsPayload {
  base_url: string
  api_key: string
  /** 协议决定 /models 用哪套认证头。 */
  protocol: ModelDegradationProtocol
}

/** 发起一次检测，立即返回任务记录（检测在服务端异步执行）。 */
export async function startModelDegradation(payload: StartModelDegradationPayload): Promise<ModelDegradationRun> {
  const { data } = await apiClient.post<ModelDegradationRun>('/model-degradation/runs', payload)
  return data
}

/**
 * 读取内置题库。
 *
 * 题库是编译期常量、与用户凭据无关，所以不需要先填 Base URL / API Key。
 * 放在服务端而不是前端硬编码，是为了让题目正文与判分用的期望答案只有一份。
 */
export async function listModelDegradationTasks(): Promise<ModelDegradationTaskBank> {
  const { data } = await apiClient.get<ModelDegradationTaskBank>('/model-degradation/tasks')
  return { logic: data.logic ?? [], render: data.render ?? [] }
}

/**
 * 用给定凭据读取上游的模型列表。
 *
 * 由服务端代理请求：浏览器直连第三方上游会被 CORS 挡掉，而且服务端还能
 * 复用与检测相同的地址校验（强制 HTTPS、拒绝私网）。
 */
export async function listUpstreamModels(
  payload: ListUpstreamModelsPayload
): Promise<string[]> {
  const { data } = await apiClient.post<{ models: string[] }>('/model-degradation/models', payload)
  return data.models ?? []
}

/** 读取单次检测详情（含各题明细）。 */
export async function getModelDegradationRun(id: number): Promise<ModelDegradationRun> {
  const { data } = await apiClient.get<ModelDegradationRun>(`/model-degradation/runs/${id}`)
  return data
}

/** 读取当前用户的检测历史。 */
export async function listModelDegradationRuns(limit = 20): Promise<ModelDegradationRun[]> {
  const { data } = await apiClient.get<{ items: ModelDegradationRun[] }>('/model-degradation/runs', {
    params: { limit }
  })
  return data.items ?? []
}

/**
 * 中止一轮仍在运行中的检测。
 *
 * 服务端会取消正在发出的上游请求，并把这一轮标记为已中止；
 * 已经跑完的题仍会保留在明细里。对已完结的记录会拿到 409。
 */
export async function cancelModelDegradationRun(id: number): Promise<void> {
  await apiClient.post(`/model-degradation/runs/${id}/cancel`)
}

export const modelDegradationAPI = {
  startModelDegradation,
  getModelDegradationRun,
  listModelDegradationRuns,
  listModelDegradationTasks,
  listUpstreamModels,
  cancelModelDegradationRun
}

export default modelDegradationAPI
