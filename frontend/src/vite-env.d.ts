/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module '@airwallex/components-sdk' {
  interface InitOptions {
    env: 'demo' | 'prod'
    enabledElements: string[]
    locale?: string
  }

  interface RedirectToCheckoutOptions {
    intent_id: string
    client_secret: string
    currency: string
    country_code: string
    successUrl: string
  }

  interface AirwallexPayments {
    redirectToCheckout(options: RedirectToCheckoutOptions): string | void
  }

  export function init(options: InitOptions): Promise<{ payments?: AirwallexPayments }>
}
