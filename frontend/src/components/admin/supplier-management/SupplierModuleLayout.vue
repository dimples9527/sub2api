<template>
  <AppLayout>
    <div class="supplier-management-page">
      <slot />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import './supplier-management.css'
</script>
