import { createApp, type App } from 'vue'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
    },
  })
}

export function renderComposable<T>(
  composable: () => T,
  options?: { queryClient?: QueryClient }
): { result: T; unmount: () => void; app: App } {
  const pinia = createPinia()
  setActivePinia(pinia)

  let result: T
  const app = createApp({
    setup() {
      result = composable()
      return () => {}
    },
  })
  const qc = options?.queryClient ?? createTestQueryClient()
  app.use(VueQueryPlugin, { queryClient: qc })
  app.use(pinia)
  app.mount(document.createElement('div'))

  return { result: result!, unmount: () => app.unmount(), app }
}
