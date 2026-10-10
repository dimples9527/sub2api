import { listSupplierAccounts, type SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import { list as listSupplierProviders } from '@/api/admin/supplierProviders'

/**
 * 「配置健康守护账号」与「配置参与择优的分组」两个弹窗共用的候选数据。
 *
 * 这两个弹窗原来只挂在自动化页的任务编辑弹窗下面，数据也由那个页面顺手取。
 * 抽成共享组件、上游账号页也要原地打开它们之后，「哪些账号算可用」这件事必须只有一处定义 ——
 * 否则两个页面各写一份分页循环，日后放宽/收紧口径时必然只改一边，
 * 出现「同一个账号在自动化页能勾、在账号页却不见了」这种查不出原因的差异。
 */

/**
 * 可用账号候选：启用 + 已匹配到本地账号。
 *
 * 只收这两种，是因为不可用账号既进不了择优的在任集合，也没法当守护对象 ——
 * 让它们出现在列表里只会让人以为「选了就会生效」。
 */
export async function fetchEligibleSupplierAccounts(): Promise<SupplierProviderAccount[]> {
  const items: SupplierProviderAccount[] = []
  let page = 1
  let result = await listSupplierAccounts({
    active: true,
    match_status: 'matched',
    page,
    page_size: 200,
  })
  items.push(...(result.items || []))

  while (items.length < result.total && result.items.length > 0) {
    page += 1
    result = await listSupplierAccounts({
      active: true,
      match_status: 'matched',
      page,
      page_size: 200,
    })
    items.push(...(result.items || []))
  }
  return items
}

/**
 * 已关闭（停用）的供应商 ID。账号列表接口不返回这个状态，只能另取一份供应商列表。
 *
 * 拿不到时返回空集合而不是抛错：它只影响标灰与那个快捷过滤，属附加信息，
 * 不该把账号列表一起拖垮、也不该弹一个「加载失败」把用户引到错误的方向。
 */
export async function fetchDisabledSupplierProviderIds(): Promise<Set<number>> {
  const providers = await listSupplierProviders({ page: 1, page_size: 200 }).catch(() => null)
  return new Set(
    (providers?.items || [])
      .filter(provider => provider.enabled === false)
      .map(provider => Number(provider.id))
      .filter(id => Number.isSafeInteger(id) && id > 0)
  )
}
