<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { computed, onMounted, onBeforeUnmount, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import { resolveDocumentTitle } from '@/router/title'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()
const globalBannerMessage = computed(() => (appStore.globalBannerMessage || '').trim())

/**
 * Update favicon dynamically
 * @param logoUrl - URL of the logo to use as favicon
 */
function updateFavicon(logoUrl: string) {
  // Find existing favicon link or create new one
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = logoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = logoUrl
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()

      // Announcements: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch
        setTimeout(() => announcementStore.fetchAnnouncements(true), 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
      announcementStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

onMounted(async () => {
  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Re-resolve document title now that siteName is available
  document.title = resolveDocumentTitle(route.meta.title, appStore.siteName, route.meta.titleKey as string)
})
</script>

<template>
  <NavigationProgress />
  <div v-if="globalBannerMessage" class="global-banner">
    <div class="global-banner__inner">
      <div class="global-banner__identity">
        <div class="global-banner__badge" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 13.5V10.5C4 9.67 4.67 9 5.5 9H8L14.43 4.18C15.09 3.68 16 4.15 16 4.97V19.03C16 19.85 15.09 20.32 14.43 19.82L8 15H5.5C4.67 15 4 14.33 4 13.5Z" />
            <path d="M19 9.5C20.21 10.3 21 11.68 21 13.25C21 14.82 20.21 16.2 19 17" />
            <path d="M17.5 11.25C18.11 11.67 18.5 12.39 18.5 13.2C18.5 14.01 18.11 14.73 17.5 15.15" />
          </svg>
        </div>
        <span class="global-banner__label">系统公告</span>
      </div>
      <div class="global-banner__marquee" role="status" aria-live="polite" :aria-label="globalBannerMessage">
        <span class="global-banner__hint" aria-hidden="true">
          <span class="global-banner__hint-dot"></span>
          最新通知
        </span>
        <div class="global-banner__viewport">
          <div class="global-banner__track">
            <span class="global-banner__message">{{ globalBannerMessage }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
  <RouterView />
  <Toast />
  <AnnouncementPopup />
</template>

<style scoped>
.global-banner {
  position: relative;
  overflow: hidden;
  border-bottom: 1px solid rgba(226, 232, 240, 0.9);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.96)),
    linear-gradient(90deg, rgba(20, 184, 166, 0.04), rgba(245, 158, 11, 0.03));
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.7), 0 6px 18px -18px rgba(15, 23, 42, 0.28);
  backdrop-filter: saturate(140%) blur(10px);
}

.global-banner::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: linear-gradient(180deg, rgb(20, 184, 166), rgb(245, 158, 11));
  opacity: 0.78;
}

.global-banner::after {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(20, 184, 166, 0.08), transparent 24%, transparent 76%, rgba(245, 158, 11, 0.08)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.18), transparent 65%);
  opacity: 1;
  pointer-events: none;
}

.global-banner__inner {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-height: 2.3rem;
  padding: 0.35rem 1rem 0.35rem 1.1rem;
}

.global-banner__identity {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 0.45rem;
  position: relative;
}

.global-banner__identity::after {
  content: '';
  position: absolute;
  top: 50%;
  right: -0.45rem;
  width: 1px;
  height: 1.2rem;
  background: linear-gradient(180deg, transparent, rgba(148, 163, 184, 0.4), transparent);
  transform: translateY(-50%);
}

.global-banner__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.55rem;
  height: 1.55rem;
  border: 1px solid rgba(20, 184, 166, 0.24);
  border-radius: 0.55rem;
  color: rgb(13, 148, 136);
  background: linear-gradient(180deg, rgba(240, 253, 250, 0.96), rgba(236, 253, 245, 0.88));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72), 0 10px 20px -16px rgba(13, 148, 136, 0.5);
}

.global-banner__badge svg {
  width: 0.85rem;
  height: 0.85rem;
}

.global-banner__label {
  display: inline-flex;
  align-items: center;
  height: 1.45rem;
  padding: 0 0.62rem;
  border: 1px solid rgba(20, 184, 166, 0.16);
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(240, 253, 250, 0.72));
  font-size: 0.69rem;
  font-weight: 700;
  letter-spacing: 0;
  color: rgb(15, 118, 110);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.75), 0 8px 18px -18px rgba(13, 148, 136, 0.45);
}

.global-banner__marquee {
  position: relative;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
}

.global-banner__hint {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  flex-shrink: 0;
  color: rgb(14, 116, 144);
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0;
}

.global-banner__hint-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  background: radial-gradient(circle at 35% 35%, rgb(153, 246, 228), rgb(20, 184, 166) 72%);
  box-shadow: 0 0 0 0 rgba(20, 184, 166, 0.34);
  animation: global-banner-pulse 2.2s ease-out infinite;
}

.global-banner__viewport {
  position: relative;
  flex: 1;
  overflow: hidden;
  min-width: 0;
  mask-image: linear-gradient(90deg, transparent, black 0.9rem, black calc(100% - 1.1rem), transparent);
}

.global-banner__viewport::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 6.5rem;
  background: linear-gradient(90deg, rgba(240, 253, 250, 0.92), rgba(240, 253, 250, 0.52), transparent);
  pointer-events: none;
  z-index: 1;
}

.global-banner__track {
  display: inline-block;
  min-width: 100%;
  padding-left: 100%;
  white-space: nowrap;
  animation: global-banner-scroll 24s linear infinite;
  will-change: transform;
}

.global-banner__message {
  font-size: 0.785rem;
  font-weight: 700;
  letter-spacing: 0;
  color: rgb(30, 41, 59);
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.65);
}

@keyframes global-banner-pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(20, 184, 166, 0.34);
  }
  70% {
    box-shadow: 0 0 0 0.42rem rgba(20, 184, 166, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(20, 184, 166, 0);
  }
}

@keyframes global-banner-scroll {
  0% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(-100%);
  }
}

@media (max-width: 640px) {
  .global-banner__inner {
    min-height: 2.2rem;
    gap: 0.5rem;
    padding: 0.32rem 0.75rem 0.32rem 0.85rem;
  }

  .global-banner__identity::after {
    right: -0.28rem;
    height: 1rem;
  }

  .global-banner__badge {
    width: 1.45rem;
    height: 1.45rem;
  }

  .global-banner__label {
    padding: 0 0.45rem;
    font-size: 0.64rem;
  }

  .global-banner__marquee {
    gap: 0.45rem;
  }

  .global-banner__hint {
    font-size: 0.66rem;
  }

  .global-banner__viewport::before {
    width: 4.6rem;
  }

  .global-banner__message {
    font-size: 0.72rem;
  }

  .global-banner__track {
    animation-duration: 20s;
  }
}

:global(.dark) .global-banner {
  border-bottom-color: rgba(51, 65, 85, 0.72);
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.98), rgba(15, 23, 42, 0.96)),
    linear-gradient(90deg, rgba(45, 212, 191, 0.06), rgba(251, 191, 36, 0.04));
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.02), 0 10px 24px -22px rgba(0, 0, 0, 0.65);
}

:global(.dark) .global-banner::before {
  background: linear-gradient(180deg, rgb(45, 212, 191), rgb(251, 191, 36));
  opacity: 0.64;
}

:global(.dark) .global-banner::after {
  background:
    linear-gradient(90deg, rgba(45, 212, 191, 0.08), transparent 30%, transparent 74%, rgba(251, 191, 36, 0.06)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 68%);
}

:global(.dark) .global-banner__label {
  border-color: rgba(45, 212, 191, 0.14);
  background: linear-gradient(180deg, rgba(15, 118, 110, 0.14), rgba(255, 255, 255, 0.04));
  color: rgb(153, 246, 228);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

:global(.dark) .global-banner__badge {
  border-color: rgba(45, 212, 191, 0.22);
  color: rgb(94, 234, 212);
  background: linear-gradient(180deg, rgba(15, 118, 110, 0.18), rgba(8, 47, 73, 0.22));
}

:global(.dark) .global-banner__identity::after {
  background: linear-gradient(180deg, transparent, rgba(71, 85, 105, 0.7), transparent);
}

:global(.dark) .global-banner__hint {
  color: rgb(103, 232, 249);
}

:global(.dark) .global-banner__hint-dot {
  background: radial-gradient(circle at 35% 35%, rgb(165, 243, 252), rgb(34, 211, 238) 72%);
  box-shadow: 0 0 0 0 rgba(34, 211, 238, 0.28);
}

:global(.dark) .global-banner__viewport::before {
  background: linear-gradient(90deg, rgba(8, 47, 73, 0.92), rgba(8, 47, 73, 0.48), transparent);
}

:global(.dark) .global-banner__message {
  color: rgb(226, 232, 240);
  text-shadow: none;
}

@media (prefers-reduced-motion: reduce) {
  .global-banner__track {
    animation: none;
    min-width: auto;
    padding-left: 0;
  }
}

</style>
