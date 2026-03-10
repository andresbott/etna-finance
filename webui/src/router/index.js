import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/store/userStore'
import { useSettingsStore } from '@/store/settingsStore.js'

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
            name: 'accounts',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/accounts/accounts.vue')
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
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/categories/CategoriesView.vue')
        },
        {
            path: '/instruments',
            name: 'instruments',
            meta: {
                requiresAuth: true,
                requiresInstruments: true,
                title: 'Investment Products'
            },
            component: () => import('@/views/instruments/InstrumentsView.vue')
        },
        {
            path: '/securities',
            redirect: '/instruments'
        },
        {
            path: '/backup-restore',
            name: 'backup-restore',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/backup/BackupRestoreView.vue')
        },
        {
            path: '/tools',
            redirect: '/tools/portfolio-simulator'
        },
        {
            path: '/tools/portfolio-simulator',
            name: 'portfolio-simulator',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/tools/PortfolioSimulatorView.vue')
        },
        {
            path: '/tools/real-estate-simulator',
            name: 'real-estate-simulator',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/tools/RealEstateSimulatorView.vue')
        },
        {
            path: '/tasks',
            name: 'tasks',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/tasks/TasksView.vue')
        },
        {
            path: '/tasks/:id',
            redirect: () => ({ name: 'tasks' })
        },
        {
            path: '/setup/csv-profiles',
            name: 'csv-profiles',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/csvimport/CsvImportProfileView.vue')
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
                if (to.params.tab && !validTabs.includes(to.params.tab)) {
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
                if (to.params.tab && !validTabs.includes(to.params.tab)) {
                    next({ path: `/market-data/stock-market/${to.params.id}/overview` })
                } else {
                    next()
                }
            }
        },
        {
            path: '/settings',
            name: 'settings',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/settings/ConfigurationView.vue')
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
            path: '/setup/category-rules',
            name: 'category-rules',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/csvimport/CategoryRulesView.vue')
        },
        {
            path: '/setup/reapply-rules',
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

    const navigate = function (to, next) {
        if (to.matched.some((record) => record.meta.requiresAuth)) {
            if (!user.isLoggedIn) {
                next({ name: 'login' })
            } else if (to.matched.some((record) => record.meta.requiresInstruments) && !settings.instruments) {
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
    const needsInstrumentsCheck = to.matched.some((record) => record.meta.requiresInstruments)
    const ensureSettingsThenNavigate = () => {
        if (needsInstrumentsCheck && !settings.isLoaded) {
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
