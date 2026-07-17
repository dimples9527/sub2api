<template>
  <AppLayout>
    <div class="help-page-shell">
      <iframe
        src="/help.html"
        class="help-page-frame"
        title="Help Docs"
      ></iframe>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
</script>

<style scoped>
.help-page-shell {
  height: calc(100vh - 2rem);
  min-height: 640px;
  overflow: hidden;
  border-radius: 12px;
  background: var(--color-bg-card, #fff);
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.06);
}

.help-page-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: #f9fafb;
}

@media (max-width: 768px) {
  .help-page-shell {
    height: calc(100vh - 1rem);
    min-height: 520px;
    border-radius: 0;
  }
}
</style>
