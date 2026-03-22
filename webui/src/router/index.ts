import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'
import { useUserStore } from '@/store/userStore'
import { useSettingsStore } from '@/store/settingsStore'

const router = createRouter({
    // history: createWebHistory(),
    history: createWebHistory('/'),
    routes: [
        {
            path: '/', // Root path
            redirect: '/reports/overview' // Redirect to overview
        },
        {
            path: '/start',
            redirect: '/reports/overview' // Redirect old start to new overview
        },
        {
            path: '/reports/overview',
            name: 'reports-overview',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/reports/DashboardView.vue')
        },
        {
            path: '/accounts',
            redirect: { name: 'settings-accounts' }
        },
        {
            path: '/entries',
            name: 'entries',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/entries/EntriesView.vue')
        },
        {
            path: '/financial-transactions',
            name: 'financial-transactions',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/entries/EntriesView.vue'),
            props: { financialOnly: true }
        },
        {
            path: '/entries/:id',
            name: 'entries-by-account',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/entries/AccountEntriesView.vue')
        },
        {
            path: '/reports/income-expense',
            name: 'reports-income-expense',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/reports/IncomeExpenseView.vue')
        },
        {
            path: '/reports/investment',
            name: 'reports-investment',
            meta: {
                requiresAuth: true,
                requiresInstruments: true
            },
            component: () => import('@/views/reports/InvestmentReportView.vue')
        },
        {
            path: '/reports',
            redirect: '/reports/income-expense' // Redirect old reports to new route
        },
        {
            path: '/categories',
            name: 'categories',
            meta: { requiresAuth: true },
            component: () => import('@/views/categories/CategoriesView.vue'),
            children: [
                { path: '', redirect: { name: 'settings-categories' } },
                { path: 'expense', redirect: { name: 'settings-categories' } },
                { path: 'income', redirect: { name: 'settings-categories' } },
                { path: 'rules', redirect: { name: 'settings-category-rules' } },
            ]
        },
        {
            path: '/instruments',
            redirect: { name: 'settings-instruments' }
        },
        {
            path: '/securities',
            redirect: { name: 'settings-instruments' }
        },
        {
            path: '/backup-restore',
            redirect: { name: 'settings-backup-restore' }
        },
        {
            path: '/financial-simulator',
            name: 'financial-simulator',
            meta: {
                requiresAuth: true,
                requiresTools: true
            },
            component: () => import('@/views/tools/FinancialSimulatorView.vue')
        },
        {
            path: '/tools',
            redirect: '/financial-simulator'
        },
        {
            path: '/financial-simulator/:toolType/:id',
            name: 'simulation-editor',
            meta: {
                requiresAuth: true,
                requiresTools: true
            },
            component: () => import('@/views/tools/SimulationEditorView.vue')
        },
        {
            path: '/tasks',
            redirect: { name: 'settings-tasks' }
        },
        {
            path: '/tasks/:id',
            redirect: { name: 'settings-tasks' }
        },
        {
            path: '/setup/csv-profiles',
            redirect: '/settings/csv-profiles'
        },
        {
            path: '/market-data/currency-exchange',
            name: 'currency-exchange',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/marketdata/CurrencyExchangeView.vue')
        },
        {
            path: '/market-data/currency-exchange/:currency',
            redirect: (to) => ({ path: `/market-data/currency-exchange/${to.params.currency}/overview` })
        },
        {
            path: '/market-data/currency-exchange/:currency/:tab',
            name: 'currency-detail',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/marketdata/CurrencyDetailView.vue'),
            beforeEnter: (to, _from, next) => {
                const validTabs = ['overview', 'raw-data']
                if (to.params.tab && !validTabs.includes(to.params.tab as string)) {
                    next({ path: `/market-data/currency-exchange/${to.params.currency}/overview` })
                } else {
                    next()
                }
            }
        },
        {
            path: '/market-data/stock-market',
            name: 'stock-market',
            meta: {
                requiresAuth: true,
                requiresInstruments: true
            },
            component: () => import('@/views/marketdata/StockMarketView.vue')
        },
        {
            path: '/market-data/stock-market/:id',
            redirect: (to) => ({ path: `/market-data/stock-market/${to.params.id}/overview` })
        },
        {
            path: '/market-data/stock-market/:id/:tab',
            name: 'stock-detail',
            meta: {
                requiresAuth: true,
                requiresInstruments: true
            },
            component: () => import('@/views/marketdata/StockDetailView.vue'),
            beforeEnter: (to, _from, next) => {
                const validTabs = ['overview', 'raw-data']
                if (to.params.tab && !validTabs.includes(to.params.tab as string)) {
                    next({ path: `/market-data/stock-market/${to.params.id}/overview` })
                } else {
                    next()
                }
            }
        },
        {
            path: '/docs',
            name: 'docs',
            meta: { requiresAuth: true },
            component: () => import('@/views/docs/DocsView.vue'),
            children: [
                { path: '', redirect: { name: 'docs-overview' } },
                { path: 'overview', name: 'docs-overview', component: () => import('@/views/docs/DocsOverviewView.vue') },
                { path: 'getting-started/configuration', name: 'docs-configuration', component: () => import('@/views/docs/getting-started/ConfigurationView.vue') },
                { path: 'guides/handling-rsus', name: 'docs-handling-rsus', component: () => import('@/views/docs/guides/HandlingRsusView.vue') },
                { path: 'guides/handling-espp', name: 'docs-handling-espp', component: () => import('@/views/docs/guides/HandlingEsppView.vue') },
                { path: 'concepts/accounts', name: 'docs-concepts-accounts', component: () => import('@/views/docs/concepts/AccountsView.vue') },
                { path: 'concepts/categories', name: 'docs-concepts-categories', component: () => import('@/views/docs/concepts/CategoriesView.vue') },
                { path: 'concepts/category-rules', name: 'docs-concepts-category-rules', component: () => import('@/views/docs/concepts/CategoryRulesView.vue') },
                { path: 'concepts/csv-import-profiles', name: 'docs-concepts-csv-import-profiles', component: () => import('@/views/docs/concepts/CsvImportProfilesView.vue') },
            ]
        },
        {
            path: '/settings',
            name: 'settings',
            meta: { requiresAuth: true },
            component: () => import('@/views/settings/SettingsView.vue'),
            children: [
                { path: '', redirect: { name: 'settings-configuration' } },
                { path: 'configuration', name: 'settings-configuration', component: () => import('@/views/settings/ConfigurationView.vue') },
                { path: 'csv-profiles', name: 'csv-profiles', component: () => import('@/views/csvimport/CsvImportProfileView.vue') },
                { path: 'categories', name: 'settings-categories', component: () => import('@/views/settings/SettingsCategoriesView.vue') },
                { path: 'category-rules', name: 'settings-category-rules', component: () => import('@/views/categories/CategoryRulesView.vue') },
                { path: 'accounts', name: 'settings-accounts', component: () => import('@/views/accounts/accounts.vue') },
                { path: 'instruments', name: 'settings-instruments', meta: { requiresInstruments: true }, component: () => import('@/views/instruments/InstrumentsView.vue') },
                { path: 'backup-restore', name: 'settings-backup-restore', component: () => import('@/views/backup/BackupRestoreView.vue') },
                { path: 'tasks', name: 'settings-tasks', component: () => import('@/views/tasks/TasksView.vue') },
            ]
        },
        {
            path: '/login',
            name: 'login',
            meta: {
                hideFromAuth: true
            },
            component: () => import('@/views/LoginView.vue')
        },
        {
            path: '/import/:accountId',
            name: 'csv-import',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/csvimport/ImportView.vue')
        },
        {
            path: '/setup/reapply-rules',
            redirect: { name: 'reapply-rules' }
        },
        {
            path: '/settings/reapply-rules',
            name: 'reapply-rules',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/csvimport/ReapplyRulesView.vue')
        },
        {
            path: '/:pathMatch(.*)*',
            name: 'NotFound',
            component: () => import('@/views/404.vue')
        }

        // { path: "*", component: {        template: '<p>Page Not Found</p>'      }
    ]
})

// this checks for metadata in the router and redirects to login page if the user is not logged in
// the same happens if the user is logged in he is redirected to the entry away from the login page
// this relies on the user store
// based on: https://stackoverflow.com/questions/52653337/vuejs-redirect-from-login-register-to-home-if-already-loggedin-redirect-from
router.beforeEach((to, from, next) => {
    const user = useUserStore()
    const settings = useSettingsStore()

    const navigate = function (to: RouteLocationNormalized, next: NavigationGuardNext) {
        if (to.matched.some((record) => record.meta.requiresAuth)) {
            if (!user.isLoggedIn) {
                next({ name: 'login' })
            } else if (to.matched.some((record) => record.meta.requiresInstruments) && !settings.instruments) {
                next({ name: 'reports-overview' })
            } else if (to.matched.some((record) => record.meta.requiresTools) && !settings.tools) {
                next({ name: 'reports-overview' })
            } else {
                next() // go to wherever I'm going
            }
        } else if (to.matched.some((record) => record.meta.hideFromAuth)) {
            if (user.isLoggedIn) {
                next({ name: 'reports-overview' }) // hide logged-in users from hitting the login page
            } else {
                next()
            }
        } else {
            next() // does not require auth, make sure to always call next()!
        }
    }

    // When the route needs the instruments flag, ensure settings are loaded first (avoids
    // redirect on F5 when settings were not yet fetched).
    const needsSettingsCheck = to.matched.some((record) => record.meta.requiresInstruments || record.meta.requiresTools)
    const ensureSettingsThenNavigate = () => {
        if (needsSettingsCheck && !settings.isLoaded) {
            settings.fetchSettings().then(() => navigate(to, next)).catch(() => navigate(to, next))
        } else {
            navigate(to, next)
        }
    }

    if (user.isFirstLogin) {
        user.setFirstLoginFalse()
        user.checkState().then(() => ensureSettingsThenNavigate())
    } else {
        ensureSettingsThenNavigate()
    }
})

export default router
