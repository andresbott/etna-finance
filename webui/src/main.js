import { createApp } from 'vue'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'
import App from './App.vue'
const app = createApp(App)

import CustomTheme from '@/theme.js'
import 'primeflex/primeflex.css'
import 'primeicons/primeicons.css'

import '@/assets/style.scss'

import PrimeVue from 'primevue/config'

app.use(PrimeVue, {
    // Default theme configuration
    theme: {
        preset: CustomTheme,
        options: {
            prefix: 'c',
            darkModeSelector: 'system',
            cssLayer: false
        }
    }
})

// pinia store
import { createPinia } from 'pinia'
app.use(createPinia())

// // initialize toast service
// import ToastService from 'primevue/toastservice';
// app.use(ToastService);

// add the app router
import router from './router'
app.use(router)

// focus trap
import FocusTrap from 'primevue/focustrap'
app.directive('focustrap', FocusTrap)

// vue query
const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            // Global default settings for queries
            refetchOnWindowFocus: false, // Disable refetching when window regains focus
            retry: 3, // Number of retries if query fails
            staleTime: 1000 * 60 * 5, // Data considered fresh for 5 minutes
            cacheTime: 1000 * 60 * 30 // Cache data for 30 minutes
        },
        mutations: {
            // Global default settings for mutations
            retry: false
        }
    }
})

app.use(VueQueryPlugin, {
    queryClient
})

app.mount('#app')
