<template>
  <BaseDialog :show="show" title="分组择优诊断快照" width="full" @close="emit('close')">
    <div class="sp-election-diagnostics-dialog">
      <p class="sp-election-diagnostics-hint">
        排查「择优调度异常」所需的上下文都在这一段文本里：任务配置、各分组当前开启数、
        开启中的账号及其所属分组、最近几轮的逐账号逐分组决策依据。
        点「复制全文」带走即可，不需要再自己写 SQL 查库。
        怀疑某个分组有问题时，先在文本里搜分组名。
      </p>
      <div class="sp-election-diagnostics-actions">
        <label class="sp-election-diagnostics-limit">
          <span>回溯轮数</span>
          <div class="sp-election-diagnostics-limit-select">
            <Select v-model="runLimit" :options="runLimitOptions" :searchable="false" @update:model-value="load" />
          </div>
        </label>
        <button class="sp-button small ghost" type="button" :disabled="loading" @click="load">
          {{ loading ? '生成中…' : '重新生成' }}
        </button>
        <button class="sp-button small primary" type="button" :disabled="loading || !text" @click="copyText">
          复制全文
        </button>
      </div>
      <pre v-if="text" class="sp-election-diagnostics-text">{{ text }}</pre>
      <p v-else class="sp-election-diagnostics-empty">
        {{ loading ? '正在生成诊断快照…' : '暂无内容，点「重新生成」重试。' }}
      </p>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { getGroupElectionDiagnostics } from '@/api/admin/supplierAutomation'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useClipboard } from '@/composables/useClipboard'

interface Props {
  show: boolean
}
const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const text = ref('')
const runLimit = ref(3)

const runLimitOptions: SelectOption[] = [
  { value: 3, label: '最近 3 轮' },
  { value: 5, label: '最近 5 轮' },
  { value: 10, label: '最近 10 轮' },
]

// 每次打开都重新生成：快照反映的是「此刻」的状态，缓存住反而会让人拿着过期数据去排查。
watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void load()
    }
  }
)

async function load() {
  loading.value = true
  try {
    const result = await getGroupElectionDiagnostics({ run_limit: Number(runLimit.value) })
    text.value = result.text || ''
  } catch (error) {
    text.value = ''
    appStore.showError(extractApiErrorMessage(error, '生成诊断快照失败'))
  } finally {
    loading.value = false
  }
}

async function copyText() {
  const copied = await copyToClipboard(text.value, '诊断快照已复制')
  if (!copied) {
    appStore.showError('复制失败，请手动选中文本复制')
  }
}
</script>

<style scoped>
/* BaseDialog 会 Teleport 到 body，拿不到页面根节点上的 --sp-*，
   所以这份快照用到的变量必须在弹窗根元素上重新声明一份。 */
.sp-election-diagnostics-dialog {
  /* 与「调度切换日志」同色：两者都属于择优调度，橙色是这一族的既有色。 */
  --sp-election-diagnostics-accent: #ea580c;
  --sp-election-diagnostics-fg: #0f172a;
  --sp-election-diagnostics-muted: #64748b;
  --sp-election-diagnostics-line: #e5e7eb;
  --sp-election-diagnostics-soft: #f8fafc;
  display: grid;
  gap: 12px;
}

/* 快照是宽文本，按内容放宽；只命中装着本弹窗的那一个 modal-content。 */
:global(.modal-content:has(.sp-election-diagnostics-dialog)) {
  max-width: 95vw;
}

/* 暗色覆盖一律写普通的 `.dark xxx`：scoped 里的 `:global(.dark) xxx`
   会被编译成裸 `.dark`，后代选择器与 data-v 一起丢掉，暗色静默失效。 */
.dark .sp-election-diagnostics-dialog {
  --sp-election-diagnostics-accent: #fb923c;
  --sp-election-diagnostics-fg: #e2e8f0;
  --sp-election-diagnostics-muted: #94a3b8;
  --sp-election-diagnostics-line: #374151;
  --sp-election-diagnostics-soft: #111827;
}

.sp-election-diagnostics-hint {
  margin: 0;
  color: var(--sp-election-diagnostics-muted);
  font-size: 12px;
  line-height: 1.6;
}

.sp-election-diagnostics-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.sp-election-diagnostics-limit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sp-election-diagnostics-muted);
  font-size: 12px;
}

.sp-election-diagnostics-limit-select {
  min-width: 132px;
}

/* 等宽 + 可滚动：快照是按列对齐的文本，换成比例字体就会错位。
   高度限制在 70vh，避免把弹窗撑出视口、让外层也跟着滚动。 */
.sp-election-diagnostics-text {
  margin: 0;
  max-height: 70vh;
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--sp-election-diagnostics-line);
  border-radius: 8px;
  background: var(--sp-election-diagnostics-soft);
  color: var(--sp-election-diagnostics-fg);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre;
  tab-size: 2;
}

.sp-election-diagnostics-empty {
  margin: 0;
  padding: 24px 0;
  color: var(--sp-election-diagnostics-muted);
  font-size: 13px;
  text-align: center;
}
</style>
