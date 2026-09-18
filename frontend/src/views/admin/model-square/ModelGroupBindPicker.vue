<template>
  <div class="model-group-bind">
    <div class="model-group-bind-head">
      <!--
        批量模式下不渲染这个二级标题：弹窗标题已经写着「批量绑定分组」，再顶一行「分组绑定」
        就是重复（AGENTS.md 明确禁止弹窗内容区重复二级标题）。平台徽标与计数保留 ——
        它们不是标题，而是下面每个选项的图例。
      -->
      <h4 v-if="mode === 'model'" class="model-group-bind-title">分组绑定</h4>
      <div class="model-group-bind-head-meta">
        <!--
          当前平台徽标兼作颜色图例：下面的分组名与徽标与它共用同一套平台取色，
          管理员一眼能对上「这个颜色 = 本平台的分组」，另一个颜色就是 composite。
        -->
        <span
          v-if="platformLabel"
          class="model-group-bind-platform"
          :class="platformBadgeClass(platform)"
        >
          {{ platformLabel }}
        </span>
        <span class="model-group-bind-count">{{ selectionSummary }}</span>
      </div>
    </div>
    <!--
      必须说清「归属由这里唯一决定」：此前分组是渠道反推出来的，管理员在配置页看不到也改不动。
      只写「选择分组」会让留空看起来像「跟随自动」，而留空实际意味着模型不出现在任何分组下。
    -->
    <p v-if="mode === 'model'" class="model-group-bind-note">
      模型在模型广场里归属的分组完全由这里决定，留空则不出现在任何分组下。
      只列出与当前平台一致的分组；composite 分组可以接收任意平台。
    </p>
    <!--
      批量模式下「留空」不等于解绑 —— 这里是「本次要加哪些」，不是「这个模型归哪些」。
      不写清追加语义，管理员会以为没勾的分组会被删掉，从而不敢只勾一个。
    -->
    <p v-else class="model-group-bind-note">
      勾选本次要追加的分组。模型原有的绑定会保留，没有勾选的分组不受影响。
    </p>

    <div v-if="contextState === 'loading'" class="model-group-bind-placeholder">分组数据加载中…</div>
    <!--
      「拉取失败」与「确实没有可绑定的分组」必须分开说：
      前者是一次抖动，后者是配置缺失，处理动作完全不同（重试 vs 去建分组）。
    -->
    <div v-else-if="contextState === 'unavailable'" class="model-group-bind-placeholder is-warn">
      分组数据加载失败，暂时无法绑定。请点页面上的「刷新」重试。
    </div>
    <div v-else-if="options.length === 0" class="model-group-bind-placeholder">
      当前平台下没有可绑定的分组，请先在「分组管理」里创建。
    </div>
    <div v-else class="model-group-bind-options">
      <label
        v-for="group in options"
        :key="String(group.id)"
        class="model-group-bind-option"
        :title="groupTitle(group)"
      >
        <input
          type="checkbox"
          class="model-group-bind-checkbox"
          :value="group.id"
          :checked="isSelected(group.id)"
          @change="toggle(group.id, ($event.target as HTMLInputElement).checked)"
        />
        <!--
          分组名与徽标都按平台着色：候选里同时存在「本平台分组」和 composite 分组，
          光看名字分不出来，而绑错平台的分组会让模型在展示页整组消失。
          名字也着色是因为徽标只占一小块，视线扫一列名字时颜色比文字标签更快。
        -->
        <span
          class="model-group-bind-name"
          :class="groupPlatformTextClass(group, platformOverrides)"
        >
          {{ group.name }}
        </span>
        <span
          class="model-group-bind-platform"
          :class="groupPlatformBadgeClass(group, platformOverrides)"
        >
          {{ groupPlatformLabel(group, platformOverrides) }}
        </span>
        <span v-if="group.rate_multiplier != null" class="model-group-bind-rate">
          {{ group.rate_multiplier }}x
        </span>
      </label>
    </div>

    <!--
      已绑定、但当前平台选不到的分组（模型换过平台，或分组被平台覆盖改到了别处）。
      必须单独列出来并允许取消：它们会计入「已选 N」，但不在候选列表里就既看不见也取消不掉 ——
      管理员看到「已选 2、只勾了 1 个」，却没有任何入口把多出来的那个去掉。
    -->
    <div v-if="orphanBoundGroups.length" class="model-group-bind-orphans">
      <p class="model-group-bind-orphans-note">
        以下分组已绑定，但当前平台绑不了（模型换过平台，或分组平台被覆盖改到了别处）。
        展示页按这些分组筛不到本模型，建议取消绑定。
      </p>
      <div
        v-for="group in orphanBoundGroups"
        :key="String(group.id)"
        class="model-group-bind-orphan"
      >
        <span class="model-group-bind-orphan-name">{{ group.name }}</span>
        <span class="model-group-bind-platform" :class="group.badgeClass">
          {{ group.platformLabel }}
        </span>
        <button
          type="button"
          class="model-group-bind-orphan-remove"
          :aria-label="`取消绑定 ${group.name}`"
          @click="toggle(group.id, false)"
        >
          取消绑定
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { listBindableGroups, normalizeModelGroupIDs, type ModelSquareUserGroup } from '@/api/admin/modelSquare'
import { platformBadgeClass } from '@/utils/platformColors'
import { resolvePlatformDisplayLabel } from '@/utils/customPlatformLabels'
import { groupPlatformBadgeClass, groupPlatformLabel, groupPlatformTextClass } from './groupPlatformStyle'

interface Props {
  modelValue: number[]
  groups: ModelSquareUserGroup[]
  platform: string
  platformOverrides?: Map<string, string>
  /**
   * 分组上下文状态。loading / unavailable 时不能渲染成「没有可绑定的分组」——
   * 那会把一次接口抖动说成配置缺失，管理员会去建重复的分组。
   */
  contextState?: 'loading' | 'unavailable' | 'ready'
  /**
   * 用途。`model` = 编辑单个模型的分组归属；`batch` = 为一批模型挑选要追加的分组。
   *
   * 两者的 modelValue 语义不同（「这个模型归哪些」vs「本次要加哪些」），
   * 文案必须跟着变 —— 批量模式下说「留空则不出现在任何分组下」会把管理员吓住。
   */
  mode?: 'model' | 'batch'
}

const props = withDefaults(defineProps<Props>(), {
  platformOverrides: () => new Map<string, string>(),
  contextState: 'ready',
  mode: 'model',
})

const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()

const options = computed(() => listBindableGroups(props.groups, props.platform, props.platformOverrides))
const selectedIds = computed(() => new Set(props.modelValue.map(id => Number(id))))
const selectedCount = computed(() => props.modelValue.length)
const selectionSummary = computed(() =>
  props.mode === 'batch' ? `已选 ${selectedCount.value} 个分组` : `已选 ${selectedCount.value}`,
)

/*
  平台为空时不给徽标：resolvePlatformDisplayLabel('') 会回一个兜底文案「API」，
  那看起来像是一个真实平台名，会让管理员以为自己选错了平台。
*/
const platformLabel = computed(() => (props.platform ? resolvePlatformDisplayLabel(props.platform) : ''))

interface OrphanBoundGroup {
  id: number
  name: string
  badgeClass: string
  platformLabel: string
}

/*
  判据是「不在 options 里」而不是「平台不兼容」：换平台、分组被覆盖改到别处、
  分组被删掉都会让一条绑定失效，用「不在候选列表里」一个条件就能全部覆盖，
  而且与 listBindableGroups 的口径天然一致 —— 后者将来多过滤一类分组，这里自动跟上。
*/
const orphanBoundGroups = computed<OrphanBoundGroup[]>(() => {
  // 批量模式下列表的勾选本来就来自 options，不可能出现孤儿；
  // 上下文没加载出来时也判不出「选不到」是数据没到还是真的不兼容，宁可先不显示。
  if (props.mode !== 'model' || props.contextState !== 'ready') return []

  const bindable = new Set(options.value.map(group => String(group.id)))
  const groupById = new Map(props.groups.map(group => [String(group.id), group]))
  const result: OrphanBoundGroup[] = []

  for (const raw of props.modelValue) {
    const id = Number(raw)
    if (!Number.isInteger(id) || id <= 0) continue
    if (bindable.has(String(id))) continue

    const group = groupById.get(String(id))
    result.push(group
      ? {
          id,
          name: group.name,
          badgeClass: groupPlatformBadgeClass(group, props.platformOverrides),
          platformLabel: groupPlatformLabel(group, props.platformOverrides),
        }
      // 分组本身也被删掉了：名字查不出来，但这条绑定仍然占着位置，必须能取消。
      : { id, name: `分组 #${id}`, badgeClass: platformBadgeClass(''), platformLabel: '已不存在' })
  }

  return result
})

function isSelected(id: number | string): boolean {
  return selectedIds.value.has(Number(id))
}

function toggle(id: number | string, checked: boolean): void {
  const numeric = Number(id)
  if (!Number.isInteger(numeric) || numeric <= 0) return

  const next = checked
    ? [...props.modelValue, numeric]
    : props.modelValue.filter(item => Number(item) !== numeric)

  // 排序去重交给共享实现：与后端保存口径、配置页脏检测口径必须完全一致。
  emit('update:modelValue', normalizeModelGroupIDs(next))
}

function groupTitle(group: ModelSquareUserGroup): string {
  /*
    平台取「有效平台」而不是 group.platform：徽标颜色按有效平台着色，
    tooltip 若还写原始平台，覆盖过的分组会出现「橙色徽标 + 写着 anthropic」这种自相矛盾的提示。
  */
  const parts = [group.name, groupPlatformLabel(group, props.platformOverrides)]
  if (group.rate_multiplier != null) parts.push(`${group.rate_multiplier}x`)
  return parts.join(' · ')
}
</script>

<style scoped>
.model-group-bind-head {
  @apply flex items-center justify-between gap-3;
}

.model-group-bind-title {
  @apply text-sm font-medium text-gray-900 dark:text-gray-100;
}

.model-group-bind-count {
  @apply shrink-0 text-xs text-gray-400 dark:text-dark-500;
}

/* ml-auto：批量模式下没有左侧标题，元信息要自己靠右，否则会贴到左边看起来像漏了内容。 */
.model-group-bind-head-meta {
  @apply ml-auto flex shrink-0 items-center gap-2;
}

/*
  平台徽标：分组徽标与「当前平台」徽标共用这一个类，取色统一由 groupPlatformStyle 给出。
  这里只给形状和边框宽度 —— 背景/文字/边框色必须留给平台色类，写死颜色会盖掉它。
*/
.model-group-bind-platform {
  @apply inline-flex shrink-0 items-center rounded border px-1.5 py-0.5 text-[10px] font-medium;
}

.model-group-bind-note {
  @apply mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400;
}

.model-group-bind-placeholder {
  @apply mt-2 rounded-lg border border-dashed border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-dark-600 dark:text-dark-400;
}

.model-group-bind-placeholder.is-warn {
  @apply border-amber-300 text-amber-700 dark:border-amber-500/40 dark:text-amber-300;
}

.model-group-bind-options {
  @apply mt-2 grid max-h-40 grid-cols-1 gap-1 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-2 sm:grid-cols-2 dark:border-dark-600 dark:bg-dark-800;
}

.model-group-bind-option {
  @apply flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 transition-colors hover:bg-white dark:hover:bg-dark-700;
}

.model-group-bind-checkbox {
  @apply h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500;
}

/*
  名字只给尺寸/截断，文字色由 groupPlatformTextClass 按平台给出。
  基础色写在这里会被平台色盖掉，等于白写。
*/
.model-group-bind-name {
  @apply min-w-0 flex-1 truncate text-xs;
}

.model-group-bind-rate {
  @apply shrink-0 text-xs text-gray-400 dark:text-dark-500;
}

/*
  孤儿绑定区：与候选列表在视觉上分开。用琥珀而不是灰 ——
  这一块是「已经失效、需要你处理」的状态，和「未绑定分组」属于同一类提醒。
*/
.model-group-bind-orphans {
  @apply mt-2 rounded-lg border border-amber-300 bg-amber-50 p-2 dark:border-amber-500/40 dark:bg-amber-500/10;
}

.model-group-bind-orphans-note {
  @apply mb-1 text-xs leading-relaxed text-amber-800 dark:text-amber-200;
}

.model-group-bind-orphan {
  @apply flex items-center gap-2 rounded px-1 py-1;
}

.model-group-bind-orphan-name {
  @apply min-w-0 flex-1 truncate text-xs text-amber-900 dark:text-amber-100;
}

.model-group-bind-orphan-remove {
  @apply shrink-0 rounded border border-amber-300 px-1.5 py-0.5 text-[10px] font-medium text-amber-800 transition-colors hover:bg-amber-100 dark:border-amber-500/40 dark:text-amber-200 dark:hover:bg-amber-500/20;
}
</style>
