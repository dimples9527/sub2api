<template>
  <div class="space-y-4">
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <!-- 价签是 -top-3 的绝对定位、会向上溢出，所以只有需要价签时才留这段顶部间距；
           没有优惠提示时留空会让它与下方「自定义金额」区块的 label 间距对不上。 -->
      <!-- 窄屏两列：375px 手机上三列时每列内容区只剩约 68px（视口 − main 的 p-4 − 卡片 p-6
           − 两处 gap − 边框与内边距），而「到账 $1000.00」这类文案需要约 78px，
           会折成两行把卡片撑高、同一行的高度还参差。两列后内容区约 120px，到账行不再折行。 -->
      <div class="grid grid-cols-2 gap-x-4 gap-y-4 sm:grid-cols-3" :class="{ 'pt-2': hasBonusTiers }">
        <button
          v-for="(amt, index) in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'relative rounded-lg border-2 px-2 py-3 text-center font-medium transition-colors sm:px-3',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : quoteFor(amt).percent > 0
                ? 'border-red-200 bg-red-50 text-gray-700 hover:border-red-300 dark:border-red-500/40 dark:bg-red-500/10 dark:text-gray-200 dark:hover:border-red-400/60'
                : TIER_CARD_CLASS[tierOf(index)],
          ]"
          :data-testid="`quick-amount-${amt}`"
          @click="selectAmount(amt)"
        >
          <!-- 促销价签（单行）：仅命中档位的金额显示；红底白字、内圈点线、右侧圆孔、整体旋转 -->
          <span
            v-if="quoteFor(amt).percent > 0"
            class="pointer-events-none absolute -right-2 -top-3 z-10 rotate-12"
            data-testid="quick-amount-bonus-badge"
          >
            <span
              class="relative flex items-center gap-1 whitespace-nowrap rounded bg-red-600 py-0.5 pl-1.5 pr-1 text-[11px] font-extrabold leading-tight tracking-tight text-white shadow-md ring-2 ring-white before:pointer-events-none before:absolute before:inset-[2px] before:rounded-sm before:border before:border-dotted before:border-white/70 dark:bg-red-500 dark:ring-dark-800"
            >
              <span>{{ badgeText(amt) }}</span>
              <span class="h-1 w-1 shrink-0 rounded-full bg-white"></span>
            </span>
          </span>
          <!-- 眉题：默认是档位标签（颜色跟随图标：选中=主色 / 命中优惠=红 / 否则档位色）；
               优惠力度最大的那一档改显示「推荐」并换成主色。
               刻意不做「挂角角标」：促销价签已占掉右上角，而卡片间距只有 16px，
               相邻卡片「左卡的右上价签」与「右卡的左上角标」会实打实地叠在一起。 -->
          <span
            class="mb-0.5 block text-[11px] font-medium leading-tight"
            :class="recommendedAmount === amt ? 'text-primary-600 dark:text-primary-400' : iconClass(amt, index)"
            :data-testid="recommendedAmount === amt ? 'quick-amount-recommended' : undefined"
          >{{ t(recommendedAmount === amt ? 'payment.amountTier.recommended' : TIER_LABEL_KEY[tierOf(index)]) }}</span>
          <!-- 图标 + 金额：图标按档位着色，与金额同行居中。
               金额的两个 span 必须写在同一行 —— 模板里的换行会被压成一个空格，让符号与数字之间多出空隙。
               tabular-nums 只给数字：加到外层会让窄字符的符号也占满等宽格子，符号与数字之间被撑开。 -->
          <span class="flex items-center justify-center gap-1">
            <Icon :name="TIER_ICON_NAME[tierOf(index)]" size="sm" :class="['shrink-0', iconClass(amt, index)]" />
            <span class="block"><span v-if="currency" class="text-sm opacity-60">{{ currencySymbol(currency) }}</span><span class="text-base font-semibold tabular-nums">{{ amt }}</span></span>
          </span>
          <!-- 到账额度：始终显示。倍率、优惠阶梯、假日活动都会抬高到账额，
               没有优惠时也该让用户确认「付多少、得多少」。 -->
          <span
            :class="[
              'mt-0.5 block text-[11px] font-normal leading-tight',
              quoteFor(amt).percent > 0 ? 'text-red-600 dark:text-red-300' : 'text-gray-400 dark:text-gray-500',
            ]"
            data-testid="quick-amount-credited"
          >{{ secondLine(amt) }}</span>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
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
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import type { RechargeBonusTier } from '@/types/payment'
import { formatRechargeBonusNumber, quoteRechargeBonus, type RechargeBonusMode } from '@/utils/rechargeBonus'
import { currencySymbol, formatPaymentAmount } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  /** 充值优惠阶梯（按 min_amount 升序）；为空时不显示价签与第二行 */
  bonusTiers?: RechargeBonusTier[]
  /** 阶梯模式：bonus 赠金 / discount 折扣 */
  bonusMode?: RechargeBonusMode
  /** 充值倍率（1 支付币种 = multiplier USD），用于计算到账金额 */
  multiplier?: number
  /** 支付币种（折扣模式第二行实付金额的币种与精度） */
  currency?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  bonusTiers: () => [],
  bonusMode: 'bonus',
  multiplier: 1,
  currency: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

// 有优惠阶梯时按钮顶部要留出价签溢出的空间（模板里的 pt-2 依赖它）。
// 到账额度那行已改成始终显示，不再由这个判断控制。
const hasBonusTiers = computed(() => props.bonusTiers.length > 0)

function currencyDigits(): number {
  if (!props.currency) return 2
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: props.currency }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function quoteFor(amt: number) {
  return quoteRechargeBonus(props.bonusTiers, amt, {
    multiplier: props.multiplier,
    mode: props.bonusMode,
    currencyDigits: currencyDigits(),
  })
}

// 快捷金额按「列表位置」分三档着色（列表是升序的，所以靠后的金额更大）。
// 刻意不按金额数值写阈值：amounts 由后台配置、支付币种也不固定，
// 同一个 100 在 CNY 下是小额、在 USD 下是大额，写死阈值会得出错误的分档。
type AmountTier = 'low' | 'mid' | 'high'

// 文字色必须写在每个档位字符串里，不能靠「基础 class + 档位 class」叠加：
// Tailwind 里同权重的 text-* 谁生效取决于 CSS 生成顺序，不取决于 class 属性的先后。
const TIER_CARD_CLASS: Record<AmountTier, string> = {
  low: 'border-sky-200 bg-sky-50 text-gray-700 hover:border-sky-300 hover:bg-sky-100 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-gray-200 dark:hover:border-sky-400/50 dark:hover:bg-sky-500/20',
  mid: 'border-violet-200 bg-violet-50 text-gray-700 hover:border-violet-300 hover:bg-violet-100 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-gray-200 dark:hover:border-violet-400/50 dark:hover:bg-violet-500/20',
  high: 'border-amber-200 bg-amber-50 text-gray-700 hover:border-amber-300 hover:bg-amber-100 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-gray-200 dark:hover:border-amber-400/50 dark:hover:bg-amber-500/20',
}

const TIER_ICON_CLASS: Record<AmountTier, string> = {
  low: 'text-sky-600 dark:text-sky-400',
  mid: 'text-violet-600 dark:text-violet-400',
  high: 'text-amber-600 dark:text-amber-400',
}

const TIER_ICON_NAME: Record<AmountTier, 'dollar' | 'trendingUp' | 'fire'> = {
  low: 'dollar',
  mid: 'trendingUp',
  high: 'fire',
}

// 档位标签的 i18n key 单独映射，避免在模板里拼 key 字符串
const TIER_LABEL_KEY: Record<AmountTier, string> = {
  low: 'payment.amountTier.low',
  mid: 'payment.amountTier.mid',
  high: 'payment.amountTier.high',
}

function tierOf(index: number): AmountTier {
  const total = filteredAmounts.value.length
  if (total <= 0) return 'low'
  const third = total / 3
  if (index < third) return 'low'
  if (index < third * 2) return 'mid'
  return 'high'
}

// 「最划算」= 优惠力度最大的一档；力度并列时取金额更大的那个。
// 没有任何档位命中优惠时返回 null —— 无优惠时没有客观依据判断哪一档更划算，不硬编一个出来。
const recommendedAmount = computed<number | null>(() => {
  let bestAmount: number | null = null
  let bestPercent = 0
  for (const amt of filteredAmounts.value) {
    const percent = quoteFor(amt).percent
    if (percent <= 0) continue
    if (percent > bestPercent || (percent === bestPercent && bestAmount !== null && amt > bestAmount)) {
      bestAmount = amt
      bestPercent = percent
    }
  }
  return bestAmount
})

// 选中态与优惠命中态要盖过档位色：选中态返回空串，图标直接继承卡片的主色文字色；
// 命中档位则整卡转红，图标跟着变红，与红色价签同一套信号。
function iconClass(amt: number, index: number): string {
  if (props.modelValue === amt) return ''
  if (quoteFor(amt).percent > 0) return 'text-red-600 dark:text-red-400'
  return TIER_ICON_CLASS[tierOf(index)]
}

// 价签文案：赠金「+20%」，折扣「20% OFF」
function badgeText(amt: number): string {
  const percent = formatRechargeBonusNumber(quoteFor(amt).percent)
  return props.bonusMode === 'discount' ? `${percent}% OFF` : `+${percent}%`
}

function secondLine(amt: number): string {
  const quote = quoteFor(amt)
  if (props.bonusMode === 'discount') {
    return t('payment.rechargeBonus.payShort', { amount: formatPaymentAmount(quote.payBase, props.currency) })
  }
  return t('payment.rechargeBonus.creditedShort', { amount: '$' + quote.credited.toFixed(2) })
}

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
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
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
