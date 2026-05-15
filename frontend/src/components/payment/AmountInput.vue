<template>
  <div class="space-y-5">
    <div>
      <label class="mb-2.5 block text-[13px] font-semibold text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-2 gap-2.5 xl:grid-cols-3">
        <button
          v-for="option in filteredOptions"
          :key="`${option.pay_amount}-${option.credit_amount}`"
          type="button"
          :class="[
            'group relative min-h-[132px] overflow-hidden rounded-[20px] border px-3.5 py-3.5 text-left transition-all duration-300',
            isSelected(option.pay_amount)
              ? 'border-teal-400/80 bg-[linear-gradient(145deg,rgba(240,253,250,0.98),rgba(204,251,241,0.82)_35%,rgba(226,232,240,0.88)_64%,rgba(254,243,199,0.44))] text-slate-900 shadow-[0_18px_38px_-28px_rgba(13,148,136,0.52)] ring-1 ring-teal-300/70 dark:border-cyan-400/70 dark:bg-[linear-gradient(150deg,rgba(11,58,86,0.96),rgba(15,118,110,0.84)_42%,rgba(15,23,42,0.98)_78%,rgba(120,53,15,0.55))] dark:text-white dark:ring-cyan-400/40'
              : 'border-slate-200/80 bg-[linear-gradient(145deg,rgba(255,255,255,0.98),rgba(240,249,255,0.92)_36%,rgba(226,232,240,0.84)_66%,rgba(255,251,235,0.55))] text-slate-800 shadow-[0_14px_30px_-26px_rgba(15,23,42,0.34)] hover:-translate-y-0.5 hover:border-teal-300/60 hover:shadow-[0_18px_36px_-26px_rgba(20,184,166,0.3)] dark:border-slate-700/80 dark:bg-[linear-gradient(155deg,rgba(15,23,42,0.96),rgba(13,37,49,0.96)_40%,rgba(30,41,59,0.98)_74%,rgba(82,38,15,0.34))] dark:text-slate-100 dark:hover:border-teal-400/50',
          ]"
          @click="selectAmount(option.pay_amount)"
        >
          <span
            class="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
            :class="isSelected(option.pay_amount)
              ? 'bg-[linear-gradient(118deg,transparent_18%,rgba(255,255,255,0.28)_48%,transparent_72%)] dark:bg-[linear-gradient(118deg,transparent_18%,rgba(255,255,255,0.12)_48%,transparent_72%)]'
              : 'bg-[linear-gradient(118deg,transparent_18%,rgba(255,255,255,0.42)_48%,transparent_72%)] dark:bg-[linear-gradient(118deg,transparent_18%,rgba(255,255,255,0.08)_48%,transparent_72%)]'"
          />
          <span class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/70 to-transparent dark:via-white/20" />

          <span class="relative flex h-full flex-col">
            <div class="flex items-start justify-between gap-2.5">
              <div
                :class="[
                  'inline-flex items-center rounded-full border px-2.5 py-1 text-[10px] font-semibold backdrop-blur-sm',
                  isSelected(option.pay_amount)
                    ? 'border-teal-500/20 bg-white/72 text-teal-700 dark:border-cyan-300/20 dark:bg-white/10 dark:text-cyan-100'
                    : 'border-white/70 bg-white/80 text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300',
                ]"
              >
                <span class="opacity-70">到账</span>
                <span
                  :class="[
                    'ml-1 text-[12px] font-bold tracking-tight',
                    isSelected(option.pay_amount)
                      ? 'text-teal-700 dark:text-cyan-100'
                      : 'text-sky-700 dark:text-sky-200',
                  ]"
                >
                  {{ option.credit_amount }}$
                </span>
              </div>

              <div class="flex flex-col items-end gap-2">
                <span
                  v-if="option.one_time"
                  :class="[
                    'rounded-full px-2 py-0.5 text-[9px] font-semibold',
                    isSelected(option.pay_amount)
                      ? 'bg-amber-500/15 text-amber-700 dark:bg-amber-300/15 dark:text-amber-200'
                      : 'bg-amber-100/90 text-amber-700 dark:bg-amber-400/12 dark:text-amber-200',
                  ]"
                >
                  限购一次
                </span>
                <span
                  v-if="isSelected(option.pay_amount)"
                  class="rounded-full border border-teal-500/20 bg-white/75 px-2 py-0.5 text-[9px] font-semibold text-teal-700 dark:border-cyan-300/20 dark:bg-white/10 dark:text-cyan-100"
                >
                  已选择
                </span>
              </div>
            </div>

            <div class="mt-4">
              <div
                :class="[
                  'text-[10px] font-medium tracking-[0.12em]',
                  isSelected(option.pay_amount)
                    ? 'text-teal-700/80 dark:text-cyan-100/70'
                    : 'text-slate-500 dark:text-slate-400',
                ]"
              >
                实付金额
              </div>
              <div class="mt-1.5 flex items-end gap-1.5">
                <span
                  :class="[
                    'text-[24px] font-bold leading-none tracking-tight',
                    isSelected(option.pay_amount)
                      ? 'text-slate-900 dark:text-white'
                      : 'text-slate-900 dark:text-slate-50',
                  ]"
                >
                  {{ option.pay_amount }}
                </span>
                <span
                  :class="[
                    'pb-0.5 text-[10px] font-semibold tracking-[0.14em]',
                    isSelected(option.pay_amount)
                      ? 'text-amber-700/85 dark:text-amber-200/80'
                      : 'text-slate-500 dark:text-slate-400',
                  ]"
                >
                  CNY
                </span>
              </div>
              <div
                :class="[
                  'mt-1.5 text-[11px] font-medium',
                  isSelected(option.pay_amount)
                    ? 'text-emerald-800 dark:text-teal-100'
                    : 'text-sky-700 dark:text-sky-200',
                ]"
              >
                到账金额 <span class="font-semibold">{{ option.credit_amount }}$</span>
              </div>
            </div>

            <div class="mt-auto pt-4">
              <div
                :class="[
                  'flex items-center justify-between rounded-xl border px-2.5 py-1.5 text-[10px]',
                  isSelected(option.pay_amount)
                    ? 'border-white/60 bg-white/68 text-slate-600 dark:border-white/10 dark:bg-white/10 dark:text-slate-300'
                    : 'border-white/80 bg-white/72 text-slate-500 dark:border-white/10 dark:bg-white/5 dark:text-slate-400',
                ]"
              >
                <span>支付后即时生效</span>
                <span
                  :class="[
                    'font-semibold',
                    isSelected(option.pay_amount)
                      ? 'text-amber-700 dark:text-amber-200'
                      : 'text-slate-600 dark:text-slate-300',
                  ]"
                >
                  {{ isSelected(option.pay_amount) ? '当前选择' : '快捷充值' }}
                </span>
              </div>

              <div class="mt-1.5 flex min-h-[16px] items-center justify-between gap-3 text-[10px]">
                <span
                  v-if="option.original_pay_amount && option.original_pay_amount > option.pay_amount"
                  :class="[
                    'line-through',
                    isSelected(option.pay_amount)
                      ? 'text-slate-500 dark:text-slate-400'
                      : 'text-slate-400 dark:text-slate-500',
                  ]"
                >
                  原价 {{ option.original_pay_amount }} CNY
                </span>
                <span v-else class="text-slate-400 dark:text-slate-500">推荐档位</span>

                <span
                  :class="[
                    'font-medium',
                    isSelected(option.pay_amount)
                      ? 'text-cyan-700 dark:text-cyan-100'
                      : 'text-sky-600 dark:text-sky-300',
                  ]"
                >
                  {{ option.credit_amount }}$ 到账
                </span>
              </div>
            </div>
          </span>
        </button>
      </div>
    </div>

    <div>
      <label class="mb-2.5 block text-[13px] font-semibold text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-400 dark:text-dark-500">
          $
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full py-3.5 pl-8 pr-4 text-sm"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RechargeAmountOption } from '@/types/payment'

const props = withDefaults(defineProps<{
  options?: RechargeAmountOption[]
  modelValue: number | null
  min?: number
  max?: number
}>(), {
  options: () => [],
  min: 0,
  max: 0,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

const filteredOptions = computed(() =>
  props.options.filter((option) =>
    (props.min <= 0 || option.pay_amount >= props.min) && (props.max <= 0 || option.pay_amount <= props.max),
  ),
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function isSelected(payAmount: number) {
  return props.modelValue === payAmount
}

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
