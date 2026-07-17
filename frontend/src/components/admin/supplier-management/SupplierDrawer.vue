<template>
  <Teleport to="body">
    <Transition name="sp-fade">
      <div v-if="show" class="supplier-management-page sp-overlay" @click.self="$emit('close')">
        <aside class="sp-drawer" role="dialog" aria-modal="true" :aria-label="title">
          <header class="sp-drawer-head">
            <div><div class="sp-eyebrow">{{ eyebrow }}</div><h3>{{ title }}</h3></div>
            <button class="sp-close" type="button" aria-label="关闭" title="关闭" @click="$emit('close')">×</button>
          </header>
          <div class="sp-drawer-body"><slot /></div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'

const props = withDefaults(defineProps<{ show: boolean; title: string; eyebrow?: string }>(), { eyebrow: 'DETAIL' })
const emit = defineEmits<{ close: [] }>()
const onKeydown = (event: KeyboardEvent) => {
  if (props.show && event.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
