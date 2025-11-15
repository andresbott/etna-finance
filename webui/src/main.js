import { createApp, nextTick } from 'vue'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'
import App from './App.vue'
import CustomTheme from '@/theme.js'

import 'primeflex/primeflex.css'
import 'primeicons/primeicons.css'
import '@/assets/style.scss'

import PrimeVue from 'primevue/config'
import StyleClass from 'primevue/styleclass'

const app = createApp(App)

// Detect and apply system color scheme preference
const applyTheme = () => {
    const darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const htmlElement = document.documentElement
    
    if (darkModeMediaQuery.matches) {
        htmlElement.classList.add('dark-mode')
    } else {
        htmlElement.classList.remove('dark-mode')
    }
}

// Apply theme on initial load
applyTheme()

// Listen for changes in system color scheme
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme)

// https://github.com/primefaces/primevue/issues/2397
// allows to use <InputText v-model="value" v-focus /> to focus on an input item
app.directive('focus', {
    mounted(el) {
        el.focus()

        setTimeout(() => {
            el.focus()
        }, 300)
    }
})

app.use(PrimeVue, {
    // Default theme configuration
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

app.directive('styleclass', StyleClass)

// pinia store
import { createPinia } from 'pinia'
app.use(createPinia())

// initialize toast service
import ToastService from 'primevue/toastservice'
app.use(ToastService)

// initialize tooltip directive
import Tooltip from 'primevue/tooltip'
app.directive('tooltip', Tooltip)

// add the app router
import router from './router'
app.use(router)

// focus trap
import FocusTrap from 'primevue/focustrap'
import { Ripple } from 'primevue'
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
