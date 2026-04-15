<template>
  <div class="space-y-5">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2.5 block text-[13px] font-semibold text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-3 gap-3">
        <button
          v-for="option in filteredOptions"
          :key="`${option.pay_amount}-${option.credit_amount}`"
          type="button"
          :class="[
            'rounded-xl border-2 px-4 py-3.5 text-center font-medium transition-colors',
            modelValue === option.pay_amount
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(option.pay_amount)"
        >
          <div class="text-[17px] font-semibold tracking-tight">
            {{ option.credit_amount }}$
          </div>
          <div
            :class="[
              'mt-1.5 text-[11px] font-semibold',
              modelValue === option.pay_amount
                ? 'text-primary-700 dark:text-primary-200'
                : 'text-sky-600 dark:text-sky-300',
            ]"
          >
            实付金额 {{ option.pay_amount }} CNY
          </div>
          <div v-if="option.original_pay_amount && option.original_pay_amount > option.pay_amount" class="mt-1.5 space-y-1 text-xs">
            <div class="text-[11px] text-gray-400 line-through dark:text-dark-500">
              原价 {{ option.original_pay_amount }} CNY
            </div>
            <div v-if="option.one_time" class="text-[11px] text-amber-600 dark:text-amber-300">
              1x
            </div>
          </div>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
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
import { ref, computed, watch } from 'vue'
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

// 0 = no limit
const filteredOptions = computed(() =>
  props.options.filter((option) =>
    (props.min <= 0 || option.pay_amount >= props.min) && (props.max <= 0 || option.pay_amount <= props.max)
  )
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

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
