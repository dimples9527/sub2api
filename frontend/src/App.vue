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
      <div class="global-banner__badge" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 13.5V10.5C4 9.67 4.67 9 5.5 9H8L14.43 4.18C15.09 3.68 16 4.15 16 4.97V19.03C16 19.85 15.09 20.32 14.43 19.82L8 15H5.5C4.67 15 4 14.33 4 13.5Z" />
          <path d="M19 9.5C20.21 10.3 21 11.68 21 13.25C21 14.82 20.21 16.2 19 17" />
          <path d="M17.5 11.25C18.11 11.67 18.5 12.39 18.5 13.2C18.5 14.01 18.11 14.73 17.5 15.15" />
        </svg>
      </div>
      <div class="global-banner__marquee" role="status" aria-live="polite" :aria-label="globalBannerMessage">
        <div class="global-banner__track">
          <span class="global-banner__message">{{ globalBannerMessage }}</span>
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
  border-bottom: 1px solid rgba(229, 231, 235, 0.9);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(249, 250, 251, 0.96));
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.global-banner::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: linear-gradient(180deg, rgb(20, 184, 166), rgb(245, 158, 11));
  opacity: 0.72;
}

.global-banner::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(20, 184, 166, 0.07), transparent 32%, transparent 68%, rgba(245, 158, 11, 0.07));
  opacity: 1;
  pointer-events: none;
}

.global-banner__inner {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 0.625rem;
  min-height: 2.5rem;
  padding: 0.45rem 1rem 0.45rem 1.1rem;
}

.global-banner__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 1.75rem;
  height: 1.75rem;
  border: 1px solid rgba(20, 184, 166, 0.2);
  border-radius: 0.625rem;
  color: rgb(13, 148, 136);
  background: rgba(240, 253, 250, 0.92);
}

.global-banner__badge svg {
  width: 0.95rem;
  height: 0.95rem;
}

.global-banner__marquee {
  position: relative;
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(90deg, transparent, black 2rem, black calc(100% - 2rem), transparent);
}

.global-banner__track {
  display: inline-block;
  min-width: 100%;
  padding-left: 100%;
  white-space: nowrap;
  animation: global-banner-scroll 22s linear infinite;
  will-change: transform;
}

.global-banner__message {
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0;
  color: rgb(55, 65, 81);
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
    min-height: 2.375rem;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem 0.4rem 0.85rem;
  }

  .global-banner__badge {
    width: 1.6rem;
    height: 1.6rem;
  }

  .global-banner__message {
    font-size: 0.75rem;
  }

  .global-banner__track {
    animation-duration: 18s;
  }
}

:global(.dark) .global-banner {
  border-bottom-color: rgba(51, 65, 85, 0.72);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.98), rgba(17, 24, 39, 0.96));
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.02);
}

:global(.dark) .global-banner::before {
  background: linear-gradient(180deg, rgb(45, 212, 191), rgb(251, 191, 36));
  opacity: 0.58;
}

:global(.dark) .global-banner::after {
  background: linear-gradient(90deg, rgba(45, 212, 191, 0.08), transparent 36%, transparent 70%, rgba(251, 191, 36, 0.06));
}

:global(.dark) .global-banner__badge {
  border-color: rgba(45, 212, 191, 0.22);
  color: rgb(94, 234, 212);
  background: rgba(15, 118, 110, 0.16);
}

:global(.dark) .global-banner__message {
  color: rgb(203, 213, 225);
}

@media (prefers-reduced-motion: reduce) {
  .global-banner__track {
    animation: none;
    min-width: auto;
    padding-left: 0;
  }
}

</style>
