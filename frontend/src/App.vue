<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { onMounted, onBeforeUnmount, watch } from 'vue'
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
const globalBannerMessage = '欢迎亲测，消耗很慢，10$相当于别的中转1亿Token'

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
  <div class="global-banner">
    <div class="global-banner__glow" aria-hidden="true"></div>
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
          <span class="global-banner__separator" aria-hidden="true">//</span>
          <span class="global-banner__message" aria-hidden="true">{{ globalBannerMessage }}</span>
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
  border-bottom: 1px solid rgba(251, 146, 60, 0.35);
  background:
    radial-gradient(circle at 12% 50%, rgba(253, 186, 116, 0.28), transparent 28%),
    radial-gradient(circle at 88% 50%, rgba(239, 68, 68, 0.24), transparent 24%),
    linear-gradient(90deg, rgba(255, 251, 235, 0.96), rgba(255, 237, 213, 0.98), rgba(254, 242, 242, 0.96));
  box-shadow: 0 10px 30px rgba(249, 115, 22, 0.08);
  backdrop-filter: blur(14px);
}

.global-banner::after {
  content: '';
  position: absolute;
  inset: 0;
  background-image: linear-gradient(120deg, rgba(255, 255, 255, 0.16) 0, rgba(255, 255, 255, 0.16) 1px, transparent 1px, transparent 12px);
  background-size: 18px 100%;
  opacity: 0.32;
  pointer-events: none;
}

.global-banner__glow {
  position: absolute;
  inset: auto 0 0 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(239, 68, 68, 0.8), transparent);
  opacity: 0.85;
}

.global-banner__inner {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  min-height: 3.5rem;
  padding: 0.65rem 1rem;
}

.global-banner__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 2.25rem;
  height: 2.25rem;
  border: 1px solid rgba(248, 113, 113, 0.35);
  border-radius: 9999px;
  color: rgb(220 38 38);
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.92), rgba(254, 226, 226, 0.9));
  box-shadow: 0 8px 20px rgba(239, 68, 68, 0.16);
}

.global-banner__badge svg {
  width: 1rem;
  height: 1rem;
}

.global-banner__marquee {
  position: relative;
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(90deg, transparent, black 6%, black 94%, transparent);
}

.global-banner__track {
  display: inline-flex;
  align-items: center;
  min-width: max-content;
  white-space: nowrap;
  animation: global-banner-scroll 16s linear infinite;
  will-change: transform;
}

.global-banner__message {
  font-size: 0.97rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: rgb(185 28 28);
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.7);
}

.global-banner__separator {
  margin: 0 1.5rem;
  font-size: 1.1rem;
  font-weight: 900;
  color: rgba(234, 88, 12, 0.9);
}

@keyframes global-banner-scroll {
  0% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(calc(-50% - 0.75rem));
  }
}

@media (max-width: 640px) {
  .global-banner__inner {
    min-height: 3.2rem;
    gap: 0.7rem;
    padding: 0.55rem 0.8rem;
  }

  .global-banner__badge {
    width: 2rem;
    height: 2rem;
  }

  .global-banner__message {
    font-size: 0.88rem;
    letter-spacing: 0.02em;
  }

  .global-banner__track {
    animation-duration: 13s;
  }
}

:global(.dark) .global-banner {
  border-bottom-color: rgba(249, 115, 22, 0.26);
  background:
    radial-gradient(circle at 12% 50%, rgba(251, 146, 60, 0.18), transparent 28%),
    radial-gradient(circle at 88% 50%, rgba(239, 68, 68, 0.16), transparent 24%),
    linear-gradient(90deg, rgba(67, 20, 7, 0.96), rgba(88, 28, 17, 0.94), rgba(69, 10, 10, 0.94));
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.26);
}

:global(.dark) .global-banner::after {
  opacity: 0.16;
}

:global(.dark) .global-banner__badge {
  border-color: rgba(248, 113, 113, 0.28);
  color: rgb(254 202 202);
  background: linear-gradient(135deg, rgba(127, 29, 29, 0.92), rgba(69, 10, 10, 0.92));
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.28);
}

:global(.dark) .global-banner__message {
  color: rgb(254 226 226);
  text-shadow: none;
}

:global(.dark) .global-banner__separator {
  color: rgba(253, 186, 116, 0.9);
}
</style>
