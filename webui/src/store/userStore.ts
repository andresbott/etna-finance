import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'
import { useSettingsStore } from '@/store/settingsStore'

const statusPath = import.meta.env.VITE_AUTH_PATH + '/status'
const loginPath = import.meta.env.VITE_AUTH_PATH + '/login'
const logoutPath = import.meta.env.VITE_AUTH_PATH + '/logout'

export const useUserStore = defineStore('user', () => {
    const settingsStore = useSettingsStore()
    const isLoggedIn = ref(false)
    const loggedInUser = ref('')
    const isFirstLogin = ref(true)
    const isLoading = ref(false)
    const wrongPwErr = ref(false)

    const user = ref('')

    const logoutCallbacks: Array<() => void> = []

    const registerLogoutAction = (callback: () => void) => {
        if (typeof callback === 'function') {
            logoutCallbacks.push(callback)
        } else {
            console.error('Callback must be a function')
        }
    }

    const setFirstLoginFalse = () => {
        isFirstLogin.value = false
    }

    const checkState = () => {
        return axios
            .get(statusPath)
            .then((res) => {
                if (res.status === 200) {
                    loggedInUser.value = res.data['username']
                    isLoggedIn.value = res.data['logged-in']
                    if (res.data['logged-in']) {
                        settingsStore.fetchSettings()
                    }
                }
            })
            .catch((err) => {
                console.error(err)
            })
    }

    const login = (
        user: string,
        pass: string,
        keepMeLoggedIn: boolean | undefined,
        onSuccessNavigate?: () => void
    ) => {
        if (!keepMeLoggedIn) {
            keepMeLoggedIn = false
        }
        const data = {
            username: user,
            password: pass,
            sessionRenew: keepMeLoggedIn
        }

        const authAxios = axios.create()
        authAxios.interceptors.response.use(
            (response) => {
                return response
            },
            (error) => {
                if (error.response?.status === 401) {
                    isLoggedIn.value = false
                    wrongPwErr.value = true
                }
                return error
            }
        )
        isLoading.value = true

        authAxios
            .post(loginPath, data)
            .then((res) => {
                if (res.status === 200) {
                    loggedInUser.value = user
                    isLoggedIn.value = true
                    wrongPwErr.value = false

                    settingsStore.fetchSettings()

                    if (onSuccessNavigate) {
                        onSuccessNavigate()
                    }
                }
            })
            .catch((err) => {
                console.error(err)
                // todo propagate login error
            })
            .finally(() => {
                isLoading.value = false
            })
    }

    const logout = () => {
        isLoading.value = true
        axios
            .post(logoutPath, '')
            .then((res) => {
                loggedInUser.value = ''
                isLoggedIn.value = false
                settingsStore.$reset()

                // Call all registered callbacks
                logoutCallbacks.forEach((callback) => {
                    try {
                        callback()
                    } catch (error) {
                        console.error('Error executing logout callback:', error)
                    }
                })

                // router.push('/login')
            })
            .catch((err) => {
                console.error(err)
                // todo propagate login error
            })
            .finally(() => {
                isLoading.value = false
            })
    }

    return {
        isFirstLogin,
        setFirstLoginFalse,

        isLoggedIn,
        loggedInUser,
        checkState,

        isLoading,
        login,
        wrongPwErr,

        logout,
        registerLogoutAction
    }
})
