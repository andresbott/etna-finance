import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/lib/user/userstore.js'

const router = createRouter({
    // history: createWebHistory(),
    history: createWebHistory('/'),
    routes: [
        {
            path: '/', // Root path
            redirect: '/start' // Redirect to start
        },
        {
            path: '/start',
            name: 'start',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/dashboard/DashboardView.vue')
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
            path: '/categories',
            name: 'categories',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/categories/CategoriesView.vue')
        },
        {
            path: '/playground',
            name: 'playground',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/playgroundView.vue')
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
                next({ name: 'start' }) // hide logged-in users from hitting the login page
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
