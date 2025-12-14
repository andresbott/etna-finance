import { setup } from '@storybook/vue3'
import PrimeVue from 'primevue/config'
import CustomTheme from '../src/theme.js'
import { createPinia } from 'pinia'
import FocusTrap from 'primevue/focustrap'
import { createRouter, createMemoryHistory } from 'vue-router'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'

import 'primeflex/primeflex.css'
import 'primeicons/primeicons.css'
import '../src/assets/style.scss'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

// Create a mock router for stories
const mockRouter = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', name: 'home', component: { template: '<div>Home</div>' } },
    { path: '/login', name: 'login', component: { template: '<div>Login</div>' } },
  ]
})

// Create a query client for Vue Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: Infinity, // Keep data fresh indefinitely in stories
    },
  },
})

// Setup PrimeVue, Pinia, Router, Vue Query, and directives for all stories
setup((app) => {
  // Install Pinia for state management
  app.use(createPinia())
  
  // Install router
  app.use(mockRouter)
  
  // Install Vue Query
  app.use(VueQueryPlugin, {
    queryClient
  })
  
  // Install PrimeVue with theme
  app.use(PrimeVue, {
    theme: {
      preset: CustomTheme,
      options: {
        prefix: 'c',
        darkModeSelector: '.dark-mode',
        cssLayer: false
      }
    },
    locale: {
      firstDayOfWeek: 1
    }
  })
  
  // Register directives needed by components
  app.directive('focustrap', FocusTrap)
})

// Export queryClient so it can be accessed in stories
export { queryClient }

/** @type { import('@storybook/vue3-vite').Preview } */
const preview = {
  parameters: {
    controls: {
      matchers: {
       color: /(background|color)$/i,
       date: /Date$/i,
      },
    },
  },
};

export default preview;