<template>
  <BaseDialog :show="show" :title="t('admin.upstreamProviders.balanceSamplerSettings')" width="wide" @close="$emit('close')">
    <div class="balance-sampler-dialog space-y-5">
      <div class="balance-sampler-controls">
        <label><input data-test="balance-sampler-enabled" type="checkbox" :checked="enabled" @change="emitEnabled" /> {{ t('admin.upstreamProviders.balanceSamplerAutoRun') }}</label>
        <label>{{ t('admin.upstreamProviders.balanceSamplerIntervalSeconds') }}<input data-test="balance-sampler-interval" type="number" min="60" step="60" class="input" :value="intervalSeconds" @input="emitInterval" /></label>
      </div>
      <div class="provider-panel"><strong>{{ t('admin.upstreamProviders.amountScale') }}</strong>
        <div class="provider-list"><label v-for="provider in providers" :key="provider.slug" class="provider-row"><span><strong>{{ provider.name }}</strong><small>{{ provider.slug }}</small></span><input :data-test="`balance-sampler-scale-${provider.slug}`" type="number" min="0.000001" step="any" class="input" :value="providerAmountScales[provider.slug]" :placeholder="String(defaultScales[provider.slug] || 1)" @input="emitScale(provider.slug, $event)" /></label><div v-if="!providers.length">{{ t('common.noData') }}</div></div>
      </div>
    </div>
    <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" :disabled="saving" data-test="balance-sampler-cancel" @click="$emit('close')">{{ t('common.cancel') }}</button><button class="btn btn-primary" :disabled="saving" data-test="balance-sampler-save" @click="$emit('save')">{{ saving ? t('common.saving') : t('common.save') }}</button></div></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
defineProps<{ show: boolean; enabled: boolean; intervalSeconds: number; providerAmountScales: Record<string, number>; providers: Array<{ slug: string; name: string }>; defaultScales: Record<string, number>; saving: boolean }>()
const emit = defineEmits<{ close: []; save: []; 'update:enabled': [boolean]; 'update:intervalSeconds': [number]; 'update:providerScale': [string, number] }>()
const { t } = useI18n()
function emitEnabled(e: Event) { emit('update:enabled', (e.target as HTMLInputElement).checked) }
function emitInterval(e: Event) { emit('update:intervalSeconds', Number((e.target as HTMLInputElement).value)) }
function emitScale(slug: string, e: Event) { emit('update:providerScale', slug, Number((e.target as HTMLInputElement).value)) }
</script>
<style scoped>
.balance-sampler-controls{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.balance-sampler-controls label{display:flex;align-items:center;gap:10px}.provider-panel{border:1px solid #e5e7eb;border-radius:12px;padding:16px}.provider-list{display:grid;gap:10px;margin-top:12px}.provider-row{display:grid;grid-template-columns:minmax(0,1fr) 180px;gap:12px;align-items:center}.provider-row span{display:grid}.provider-row small{color:#64748b}@media(max-width:768px){.balance-sampler-controls,.provider-row{grid-template-columns:1fr}.balance-sampler-controls label{flex-direction:column;align-items:flex-start}.provider-row .input{width:100%}}
</style>
