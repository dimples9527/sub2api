<template>
  <div class="space-y-4">
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <button
          v-for="option in filteredOptions"
          :key="option.pay_amount"
          type="button"
          :data-testid="`amount-option-${option.pay_amount}`"
          :class="[
            'rounded-lg border p-3 text-left transition-all',
            modelValue === option.pay_amount
              ? 'border-primary-500 bg-primary-50 shadow-sm ring-1 ring-primary-500 dark:border-primary-400 dark:bg-primary-900/30 dark:ring-primary-400'
              : 'border-gray-200 bg-white hover:border-primary-200 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-700 dark:hover:bg-dark-700',
          ]"
          @click="selectAmount(option.pay_amount)"
        >
          <span class="block text-xs text-gray-400 dark:text-gray-500">
            {{ t('payment.rechargePayAmount') }}
          </span>
          <span class="mt-1 block text-lg font-semibold text-gray-900 dark:text-white">
            {{ formatAmount(option.pay_amount) }}
          </span>
          <span class="mt-2 block border-t border-gray-100 pt-2 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
            {{ t('payment.rechargeCreditAmount') }}
            <strong class="ml-1 font-semibold text-primary-600 dark:text-primary-400">
              {{ formatAmount(option.credit_amount) }}
            </strong>
          </span>
        </button>
      </div>
    </div>

    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-500">
          $
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full py-3 pl-8 pr-4"
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
  amounts?: number[]
  options?: RechargeAmountOption[]
  modelValue: number | null
  min?: number
  max?: number
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

const amountOptions = computed<RechargeAmountOption[]>(() => {
  if (props.options?.length) return props.options
  return props.amounts.map((amount) => ({
    pay_amount: amount,
    credit_amount: amount,
  }))
})

const filteredOptions = computed(() =>
  amountOptions.value.filter((option) =>
    (props.min <= 0 || option.pay_amount >= props.min) &&
    (props.max <= 0 || option.pay_amount <= props.max),
  )
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `>= ${props.min}`
  if (props.max > 0) return `<= ${props.max}`
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

function formatAmount(value: number): string {
  return `$${Number(value).toLocaleString(undefined, {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 2,
    maximumFractionDigits: 2,
  })}`
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
