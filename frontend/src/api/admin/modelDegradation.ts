/**
 * 降智检测（模型审计）管理端接口。
 *
 * 只处理「检测历史可见性」这类只给管理端使用的配置：
 * 管理端可以隐藏指定平台分组 / 第三方上游主机，普通用户读取检测历史时
 * 不再看到对应记录；管理员账号不受该配置影响，仍能看到全量数据。
 */

import { apiClient } from '../client'

/** 按主机归桶的第三方上游记录数（供管理端挑选要隐藏的上游）。 */
export interface ModelDegradationHistoryHostStat {
  host: string
  run_count: number
}

/** 检测历史可见性配置。 */
export interface ModelDegradationHistoryVisibility {
  hidden_group_ids: number[]
  hidden_hosts: string[]
  hosts: ModelDegradationHistoryHostStat[]
}

export interface ModelDegradationHistoryVisibilityUpdateResult {
  hidden: boolean
}

export async function getModelDegradationHistoryVisibility(): Promise<ModelDegradationHistoryVisibility> {
  const { data } = await apiClient.get<ModelDegradationHistoryVisibility>(
    '/admin/model-degradation/history-visibility',
  )
  return data
}

export async function setModelDegradationHistoryGroupHidden(
  groupId: number,
  hidden: boolean,
): Promise<ModelDegradationHistoryVisibilityUpdateResult> {
  const { data } = await apiClient.put<ModelDegradationHistoryVisibilityUpdateResult>(
    `/admin/model-degradation/history-visibility/groups/${groupId}`,
    { hidden },
  )
  return data
}

/**
 * host 允许传完整地址（如 https://api.example.com:8443/v1），
 * 服务端会归一化成小写主机名（去 scheme / 路径 / 端口）后再落库。
 */
export async function setModelDegradationHistoryHostHidden(
  host: string,
  hidden: boolean,
): Promise<ModelDegradationHistoryVisibilityUpdateResult> {
  const { data } = await apiClient.put<ModelDegradationHistoryVisibilityUpdateResult>(
    '/admin/model-degradation/history-visibility/hosts',
    { host, hidden },
  )
  return data
}

export const modelDegradationAPI = {
  getModelDegradationHistoryVisibility,
  setModelDegradationHistoryGroupHidden,
  setModelDegradationHistoryHostHidden,
}

export default modelDegradationAPI
