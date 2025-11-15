import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/lib/user/userstore.js'

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
            path: '/backup-restore',
            name: 'backup-restore',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/backup/BackupRestoreView.vue')
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
            path: '/market-data/stock-market',
            name: 'stock-market',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/marketdata/StockMarketView.vue')
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

    const navigate = function (to, next) {
        if (to.matched.some((record) => record.meta.requiresAuth)) {
            if (!user.isLoggedIn) {
                next({ name: 'login' })
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
    if (user.isFirstLogin) {
        user.setFirstLoginFalse()
        const p = user.checkState()
        p.then(() => {
            navigate(to, next)
        })
    } else {
        navigate(to, next)
    }
})

export default router
