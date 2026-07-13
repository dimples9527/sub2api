import { watch, type Ref } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'

export interface RouteQueryFilterBinding {
  queryKey: string
  state: Ref<any>
  fromQuery?: (value: string) => string
  toQuery?: (value: string) => string | undefined
}

export function useRouteQueryFilters(bindings: RouteQueryFilterBinding[]) {
  const route = useRoute()
  const router = useRouter()
  const routeQuery = route?.query || {}

  for (const binding of bindings) {
    const raw = routeQuery[binding.queryKey]
    const value = Array.isArray(raw) ? raw[0] : raw
    if (typeof value !== 'string') continue
    binding.state.value = binding.fromQuery ? binding.fromQuery(value) : value
  }

  watch(
    bindings.map(binding => binding.state),
    () => {
      const query: LocationQueryRaw = { ...routeQuery }
      let changed = false
      for (const binding of bindings) {
        const next = binding.toQuery ? binding.toQuery(binding.state.value) : binding.state.value || undefined
        const currentRaw = routeQuery[binding.queryKey]
        const current = Array.isArray(currentRaw) ? currentRaw[0] : currentRaw
        if (next === current) continue
        changed = true
        if (next === undefined) delete query[binding.queryKey]
        else query[binding.queryKey] = next
      }
      if (changed && router?.replace) void router.replace({ query })
    }
  )
}
