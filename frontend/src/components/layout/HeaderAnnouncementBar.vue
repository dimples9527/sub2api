<template>
  <div
    v-if="message"
    class="header-announcement hidden xl:flex"
    role="status"
    aria-live="polite"
    :aria-label="message"
  >
    <div class="header-announcement__identity" aria-hidden="true">
      <span class="header-announcement__dot"></span>
      <span class="header-announcement__label">公告</span>
    </div>
    <div class="header-announcement__viewport">
      <div class="header-announcement__track">
        <span class="header-announcement__message">{{ message }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  message: string
}>()
</script>

<style scoped>
.header-announcement {
  position: relative;
  align-items: center;
  gap: 0.65rem;
  width: min(40vw, 30rem);
  min-width: 16rem;
  height: 2.35rem;
  padding: 0 0.85rem 0 0.8rem;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 1rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 252, 0.84)),
    linear-gradient(90deg, rgba(20, 184, 166, 0.05), rgba(59, 130, 246, 0.03));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7), 0 14px 26px -24px rgba(15, 23, 42, 0.34);
  backdrop-filter: saturate(140%) blur(10px);
}

.header-announcement::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: linear-gradient(180deg, rgb(20, 184, 166), rgb(59, 130, 246));
  opacity: 0.9;
}

.header-announcement::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(20, 184, 166, 0.07), transparent 30%, transparent 72%, rgba(59, 130, 246, 0.07));
  pointer-events: none;
}

.header-announcement__identity {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 0.42rem;
  flex-shrink: 0;
}

.header-announcement__dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  background: radial-gradient(circle at 35% 35%, rgb(153, 246, 228), rgb(20, 184, 166) 72%);
  box-shadow: 0 0 0 0 rgba(20, 184, 166, 0.28);
  animation: header-announcement-pulse 2.3s ease-out infinite;
}

.header-announcement__label {
  display: inline-flex;
  align-items: center;
  height: 1.35rem;
  padding: 0 0.5rem;
  border: 1px solid rgba(20, 184, 166, 0.14);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.68);
  color: rgb(15, 118, 110);
  font-size: 0.68rem;
  font-weight: 700;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.78);
}

.header-announcement__viewport {
  position: relative;
  z-index: 1;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  mask-image: linear-gradient(90deg, transparent, black 0.8rem, black calc(100% - 0.9rem), transparent);
}

.header-announcement__viewport::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3.6rem;
  background: linear-gradient(90deg, rgba(240, 253, 250, 0.86), rgba(240, 253, 250, 0.34), transparent);
  pointer-events: none;
  z-index: 1;
}

.header-announcement__track {
  display: inline-block;
  min-width: 100%;
  padding-left: 100%;
  white-space: nowrap;
  animation: header-announcement-scroll 18s linear infinite;
  will-change: transform;
}

.header-announcement__message {
  font-size: 0.78rem;
  font-weight: 700;
  color: rgb(30, 41, 59);
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.65);
}

@keyframes header-announcement-scroll {
  0% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(-100%);
  }
}

@keyframes header-announcement-pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(20, 184, 166, 0.28);
  }
  70% {
    box-shadow: 0 0 0 0.35rem rgba(20, 184, 166, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(20, 184, 166, 0);
  }
}

:global(.dark) .header-announcement {
  border-color: rgba(71, 85, 105, 0.42);
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.9), rgba(15, 23, 42, 0.82)),
    linear-gradient(90deg, rgba(45, 212, 191, 0.08), rgba(56, 189, 248, 0.05));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03), 0 14px 26px -24px rgba(0, 0, 0, 0.58);
}

:global(.dark) .header-announcement__label {
  border-color: rgba(45, 212, 191, 0.14);
  background: linear-gradient(180deg, rgba(15, 118, 110, 0.16), rgba(255, 255, 255, 0.03));
  color: rgb(153, 246, 228);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

:global(.dark) .header-announcement__dot {
  background: radial-gradient(circle at 35% 35%, rgb(165, 243, 252), rgb(34, 211, 238) 72%);
  box-shadow: 0 0 0 0 rgba(34, 211, 238, 0.24);
}

:global(.dark) .header-announcement__viewport::before {
  background: linear-gradient(90deg, rgba(8, 47, 73, 0.88), rgba(8, 47, 73, 0.3), transparent);
}

:global(.dark) .header-announcement__message {
  color: rgb(226, 232, 240);
  text-shadow: none;
}

@media (prefers-reduced-motion: reduce) {
  .header-announcement__track {
    animation: none;
    min-width: auto;
    padding-left: 0;
  }
}
</style>
