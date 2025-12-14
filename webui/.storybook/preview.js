import { setup } from '@storybook/vue3'
import PrimeVue from 'primevue/config'
import CustomTheme from '../src/theme.js'
import { createPinia } from 'pinia'
import FocusTrap from 'primevue/focustrap'
import { createRouter, createMemoryHistory } from 'vue-router'

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

// Setup PrimeVue, Pinia, Router, and directives for all stories
setup((app) => {
  // Install Pinia for state management
  app.use(createPinia())
  
  // Install router
  app.use(mockRouter)
  
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