import { platformBadgeClass, platformTextClass } from '@/utils/platformColors'
import { resolvePlatformDisplayLabel } from '@/utils/customPlatformLabels'
import { effectiveGroupPlatform, type ModelSquareUserGroup } from '@/api/admin/modelSquare'

/*
  分组平台配色。配置页里凡是要「显示一个分组」的地方都走这里。

  为什么要单独抽一个模块，而不是在两个组件里各写一份：弹窗里的绑定控件和表格的绑定列
  是同一列数据的两种呈现，取色一旦分叉，同一个分组在两处会是两个颜色，管理员会以为
  它们是两个不同的分组。

  为什么取「有效平台」而不是 group.platform：平台覆盖会把分组挂到另一个平台下，
  按原始平台着色会和 listBindableGroups 的过滤结果自相矛盾 —— 分组能出现在候选列表里，
  却标着另一个平台的颜色。详见 effectiveGroupPlatform 的注释。
*/
export function groupDisplayPlatform(
  group: ModelSquareUserGroup,
  platformOverrides?: Map<string, string>
): string {
  return effectiveGroupPlatform(group, platformOverrides || new Map<string, string>())
}

/** 分组徽标配色。与全站 platformColors 同一套取色，不在这里另立一份色板。 */
export function groupPlatformBadgeClass(
  group: ModelSquareUserGroup,
  platformOverrides?: Map<string, string>
): string {
  return platformBadgeClass(groupDisplayPlatform(group, platformOverrides))
}

/*
  分组名的文字配色。与徽标同出 platformColors：色板将来调整时两边一起变，
  不会出现「名字还是旧绿、徽标已经换了新绿」这种只有并排看才发现的不一致。
  名字与徽标都要着色，是因为徽标只占一小块 —— 视线扫一列名字时，颜色比文字标签更快。
*/
export function groupPlatformTextClass(
  group: ModelSquareUserGroup,
  platformOverrides?: Map<string, string>
): string {
  return platformTextClass(groupDisplayPlatform(group, platformOverrides))
}

/*
  分组平台名。tooltip 里必须带上它 —— 颜色不能是唯一的信息载体：
  色觉障碍下同色系（如 gemini 蓝 / zhipu 靛）根本分不出来，只看颜色等于没给信息。
*/
export function groupPlatformLabel(
  group: ModelSquareUserGroup,
  platformOverrides?: Map<string, string>
): string {
  return resolvePlatformDisplayLabel(groupDisplayPlatform(group, platformOverrides))
}
