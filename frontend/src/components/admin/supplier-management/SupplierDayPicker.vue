<template>
  <div ref="rootRef" class="sp-day-picker">
    <button
      ref="triggerRef"
      class="sp-day-trigger"
      :class="{ 'is-open': open }"
      type="button"
      :disabled="disabled"
      aria-haspopup="dialog"
      :aria-expanded="open"
      :aria-label="`${label} ${modelValue}`"
      data-test="stat-date"
      @click="toggle"
    >
      <Icon name="calendar" size="sm" class="sp-day-trigger-icon" />
      <span class="sp-day-trigger-value">{{ modelValue || '选择日期' }}</span>
      <span v-if="weekdayLabel" class="sp-day-trigger-weekday">{{ weekdayLabel }}</span>
    </button>

    <Teleport to="body">
      <Transition name="sp-day-pop">
        <div
          v-if="open"
          ref="panelRef"
          class="sp-day-panel"
          :style="panelStyle"
          role="dialog"
          :aria-label="label"
          data-test="day-panel"
          @keydown.escape="closePanel(true)"
        >
          <header class="sp-day-panel-head">
            <button class="sp-day-nav" type="button" aria-label="上一月" data-test="prev-month" @click="stepMonth(-1)">
              <Icon name="chevronLeft" size="xs" />
            </button>
            <strong data-test="panel-month">{{ monthLabel }}</strong>
            <button class="sp-day-nav" type="button" aria-label="下一月" :disabled="nextMonthDisabled" data-test="next-month" @click="stepMonth(1)">
              <Icon name="chevronRight" size="xs" />
            </button>
          </header>
          <div class="sp-day-weekdays" aria-hidden="true">
            <span v-for="weekday in WEEKDAY_LABELS" :key="weekday">{{ weekday }}</span>
          </div>
          <div class="sp-day-grid" @keydown="onGridKeydown">
            <button
              v-for="cell in cells"
              :key="cell.value"
              class="sp-day-cell"
              :class="{ 'is-outside': !cell.inMonth, 'is-today': cell.isToday, 'is-selected': cell.value === modelValue }"
              type="button"
              :data-day="cell.value"
              :data-test="`day-${cell.value}`"
              :disabled="cell.disabled"
              :tabindex="cell.value === focusedValue ? 0 : -1"
              :aria-label="cell.label"
              :aria-current="cell.isToday ? 'date' : undefined"
              :aria-pressed="cell.value === modelValue"
              @click="select(cell.value)"
            >{{ cell.day }}</button>
          </div>
          <footer class="sp-day-panel-foot">
            <button class="sp-day-preset" type="button" :disabled="isBlocked(todayValue)" data-test="panel-today" @click="select(todayValue)">今天</button>
            <button class="sp-day-preset" type="button" :disabled="isBlocked(yesterdayValue)" data-test="panel-yesterday" @click="select(yesterdayValue)">昨天</button>
            <span v-if="max" class="sp-day-hint">最晚可选 {{ max }}</span>
          </footer>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  max?: string
  disabled?: boolean
  label?: string
}>(), { max: '', disabled: false, label: '选择日期' })

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
}>()

const PANEL_WIDTH = 288
const WEEKDAY_LABELS = ['一', '二', '三', '四', '五', '六', '日']
const TRIGGER_WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const KEY_OFFSETS: Record<string, number> = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 }

function toValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function parseValue(value: string) {
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? new Date() : date
}

const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const panelPosition = ref({ top: 0, left: 0 })
const viewYear = ref(0)
const viewMonth = ref(0)
const focusedValue = ref('')

const todayValue = computed(() => toValue(new Date()))
const yesterdayValue = computed(() => toValue(new Date(Date.now() - 24 * 60 * 60 * 1000)))
const weekdayLabel = computed(() => (props.modelValue ? TRIGGER_WEEKDAYS[parseValue(props.modelValue).getDay()] : ''))
const monthLabel = computed(() => `${viewYear.value} 年 ${viewMonth.value + 1} 月`)
const panelStyle = computed(() => ({
  top: `${panelPosition.value.top}px`,
  left: `${panelPosition.value.left}px`,
  width: `${PANEL_WIDTH}px`,
}))

function isBlocked(value: string) {
  return Boolean(props.max) && value > props.max
}

const nextMonthDisabled = computed(() => isBlocked(toValue(new Date(viewYear.value, viewMonth.value + 1, 1))))

const cells = computed(() => {
  const firstWeekday = (new Date(viewYear.value, viewMonth.value, 1).getDay() + 6) % 7
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(viewYear.value, viewMonth.value, index + 1 - firstWeekday)
    const value = toValue(date)
    return {
      value,
      day: date.getDate(),
      label: `${date.getFullYear()} 年 ${date.getMonth() + 1} 月 ${date.getDate()} 日`,
      inMonth: date.getMonth() === viewMonth.value && date.getFullYear() === viewYear.value,
      isToday: value === todayValue.value,
      disabled: isBlocked(value),
    }
  })
})

function syncViewFromModel() {
  const date = parseValue(props.modelValue || todayValue.value)
  viewYear.value = date.getFullYear()
  viewMonth.value = date.getMonth()
  focusedValue.value = props.modelValue || todayValue.value
}

// 面板 teleport 到 body 后不再跟随触发器，所以位置按触发器的实时 rect 算，空间不够时向上翻转。
function updatePosition() {
  const trigger = triggerRef.value
  if (!trigger) return
  const rect = trigger.getBoundingClientRect()
  const panelHeight = panelRef.value?.offsetHeight ?? 0
  const gap = 8
  const below = rect.bottom + gap
  const flip = below + panelHeight > window.innerHeight - gap && rect.top - panelHeight - gap > gap
  panelPosition.value = {
    top: flip ? rect.top - panelHeight - gap : below,
    left: Math.max(12, Math.min(rect.left, window.innerWidth - PANEL_WIDTH - 12)),
  }
}

function focusDay(value: string) {
  panelRef.value?.querySelector<HTMLButtonElement>(`[data-day="${value}"]`)?.focus()
}

function onDocumentMouseDown(event: MouseEvent) {
  const target = event.target as Node
  if (rootRef.value?.contains(target) || panelRef.value?.contains(target)) return
  closePanel()
}

async function openPanel() {
  if (props.disabled) return
  syncViewFromModel()
  open.value = true
  document.addEventListener('mousedown', onDocumentMouseDown, true)
  window.addEventListener('resize', updatePosition)
  window.addEventListener('scroll', updatePosition, true)
  await nextTick()
  updatePosition()
  focusDay(focusedValue.value)
}

function closePanel(restoreFocus = false) {
  if (!open.value) return
  open.value = false
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  window.removeEventListener('resize', updatePosition)
  window.removeEventListener('scroll', updatePosition, true)
  if (restoreFocus) triggerRef.value?.focus()
}

function toggle() {
  if (open.value) closePanel()
  else void openPanel()
}

function stepMonth(offset: number) {
  const base = new Date(viewYear.value, viewMonth.value + offset, 1)
  viewYear.value = base.getFullYear()
  viewMonth.value = base.getMonth()
}

function select(value: string) {
  if (isBlocked(value)) return
  emit('update:modelValue', value)
  emit('change', value)
  closePanel(true)
}

function onGridKeydown(event: KeyboardEvent) {
  const offset = KEY_OFFSETS[event.key]
  if (offset === undefined) return
  event.preventDefault()
  const base = parseValue(focusedValue.value || props.modelValue || todayValue.value)
  const next = new Date(base.getFullYear(), base.getMonth(), base.getDate() + offset)
  const value = toValue(next)
  if (isBlocked(value)) return
  focusedValue.value = value
  viewYear.value = next.getFullYear()
  viewMonth.value = next.getMonth()
  void nextTick(() => focusDay(value))
}

watch(() => props.modelValue, () => {
  if (open.value) syncViewFromModel()
})

onBeforeUnmount(() => closePanel())
</script>

<style scoped>
.sp-day-picker { position: relative; min-width: 0; }
.sp-day-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 38px;
  padding: 0 10px;
  border: 1px solid var(--sp-line, #e5e7eb);
  border-radius: 8px;
  background: var(--sp-panel, #ffffff);
  color: var(--sp-text, #111827);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: border-color .18s ease, box-shadow .18s ease;
}
.sp-day-trigger:hover:not(:disabled) { border-color: color-mix(in srgb, var(--sp-blue, #2563eb) 45%, var(--sp-line, #e5e7eb)); }
.sp-day-trigger:focus-visible,
.sp-day-trigger.is-open {
  outline: none;
  border-color: var(--sp-blue, #2563eb);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-blue, #2563eb) 16%, transparent);
}
.sp-day-trigger:disabled { opacity: .6; cursor: not-allowed; }
.sp-day-trigger-icon { flex: none; color: var(--sp-blue, #2563eb); }
.sp-day-trigger-value { flex: 1; min-width: 0; font-variant-numeric: tabular-nums; font-weight: 600; letter-spacing: .01em; }
.sp-day-trigger-weekday {
  flex: none;
  padding: 1px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-blue, #2563eb) 10%, transparent);
  color: var(--sp-blue, #2563eb);
  font-size: 11px;
  font-weight: 600;
}

/* 面板 teleport 到 body，拿不到 .supplier-management-page 的变量和字体继承，这里自带一份 */
.sp-day-panel {
  --sp-panel: #ffffff;
  --sp-line: #e5e7eb;
  --sp-soft: #f1f5f9;
  --sp-text: #111827;
  --sp-muted: #64748b;
  --sp-dim: #94a3b8;
  --sp-blue: #2563eb;
  --sp-amber: #d97706;
  position: fixed;
  z-index: 45;
  padding: 12px;
  border: 1px solid var(--sp-line);
  border-radius: 14px;
  background: var(--sp-panel);
  box-shadow: 0 18px 40px -18px rgba(15, 23, 42, .35), 0 2px 6px rgba(15, 23, 42, .06);
  color: var(--sp-text);
  font-family: inherit;
}
.dark .sp-day-panel {
  --sp-panel: #1f2937;
  --sp-line: #374151;
  --sp-soft: #374151;
  --sp-text: #f9fafb;
  --sp-muted: #9ca3af;
  --sp-dim: #6b7280;
  --sp-blue: #60a5fa;
  --sp-amber: #fbbf24;
  box-shadow: 0 18px 40px -18px rgba(0, 0, 0, .6), 0 2px 6px rgba(0, 0, 0, .3);
}
.sp-day-panel button { font-family: inherit; }
.sp-day-panel-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 10px; }
.sp-day-panel-head strong { font-size: 13px; font-variant-numeric: tabular-nums; letter-spacing: .02em; }
.sp-day-nav {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  background: var(--sp-panel);
  color: var(--sp-muted);
  cursor: pointer;
  transition: border-color .16s ease, color .16s ease, background .16s ease;
}
.sp-day-nav:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-blue) 40%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-blue) 8%, var(--sp-panel));
  color: var(--sp-blue);
}
.sp-day-nav:disabled { opacity: .4; cursor: not-allowed; }
.sp-day-weekdays,
.sp-day-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 2px; }
.sp-day-weekdays { margin-bottom: 4px; }
.sp-day-weekdays span { padding: 4px 0; color: var(--sp-dim); font-size: 11px; font-weight: 600; text-align: center; }
.sp-day-weekdays span:nth-child(n + 6) { color: var(--sp-amber); }
.sp-day-cell {
  position: relative;
  height: 32px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--sp-text);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background .14s ease, color .14s ease, border-color .14s ease;
}
.sp-day-cell:hover:not(:disabled) { background: color-mix(in srgb, var(--sp-blue) 10%, transparent); color: var(--sp-blue); }
.sp-day-cell:focus-visible { outline: none; border-color: var(--sp-blue); }
.sp-day-cell.is-outside { color: var(--sp-dim); }
.sp-day-cell:disabled { opacity: .32; cursor: not-allowed; }
/* 今天用底部小点标记，和「已选中」的实心态分开，两者叠加时点子转成白色 */
.sp-day-cell.is-today::after {
  position: absolute;
  bottom: 4px;
  left: 50%;
  width: 3px;
  height: 3px;
  margin-left: -1.5px;
  border-radius: 50%;
  background: var(--sp-amber);
  content: '';
}
.sp-day-cell.is-selected { border-color: var(--sp-blue); background: var(--sp-blue); color: #ffffff; font-weight: 700; }
.dark .sp-day-cell.is-selected { color: #0b1220; }
.sp-day-cell.is-selected.is-today::after { background: rgba(255, 255, 255, .85); }
.dark .sp-day-cell.is-selected.is-today::after { background: rgba(11, 18, 32, .7); }
.sp-day-panel-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--sp-soft);
}
.sp-day-preset {
  padding: 4px 10px;
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color .16s ease, color .16s ease;
}
.sp-day-preset:hover:not(:disabled) { border-color: color-mix(in srgb, var(--sp-blue) 40%, var(--sp-line)); color: var(--sp-blue); }
.sp-day-preset:disabled { opacity: .45; cursor: not-allowed; }
.sp-day-hint { margin-left: auto; color: var(--sp-dim); font-size: 11px; font-variant-numeric: tabular-nums; }
.sp-day-pop-enter-active,
.sp-day-pop-leave-active { transition: opacity .16s ease, transform .16s ease; }
.sp-day-pop-enter-from,
.sp-day-pop-leave-to { opacity: 0; transform: translateY(-6px) scale(.98); }
@media (prefers-reduced-motion: reduce) {
  .sp-day-pop-enter-active,
  .sp-day-pop-leave-active { transition: none; }
}
</style>
