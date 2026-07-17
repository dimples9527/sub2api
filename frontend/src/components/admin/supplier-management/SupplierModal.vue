<template>
  <Teleport to="body">
    <Transition name="sp-fade">
      <div v-if="show" class="supplier-management-page sp-overlay" @click.self="$emit('close')">
        <section class="sp-modal" :class="modalClass" role="dialog" aria-modal="true" :aria-label="title">
          <header class="sp-modal-head">
            <div><div class="sp-eyebrow">供应商管理</div><h3>{{ title }}</h3></div>
            <button class="sp-close" type="button" aria-label="关闭" title="关闭" @click="$emit('close')">×</button>
          </header>
          <div class="sp-modal-body"><slot /></div>
          <footer class="sp-modal-foot">
            <button class="sp-button ghost" type="button" @click="$emit('close')">取消</button>
            <button class="sp-button primary" type="button" @click="$emit('confirm')">{{ confirmText }}</button>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'

const props = withDefaults(defineProps<{ show: boolean; title: string; confirmText?: string; modalClass?: string }>(), {
  confirmText: '仅演示，不执行修改',
  modalClass: '',
})
const emit = defineEmits<{ close: []; confirm: [] }>()
const onKeydown = (event: KeyboardEvent) => {
  if (props.show && event.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
