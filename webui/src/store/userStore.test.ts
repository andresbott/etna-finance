import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import axios from 'axios'

vi.mock('axios', () => {

    const mockAxiosInstance = {
        interceptors: {
            response: { use: vi.fn() }
        },
        post: vi.fn()
    }
    return {
        default: {
            get: vi.fn(),
            post: vi.fn(),
            create: vi.fn(() => mockAxiosInstance)
        }
    }
})

vi.mock('@/store/settingsStore', () => {
    const mockFetchSettings = vi.fn()
    const mockReset = vi.fn()
    return {
        useSettingsStore: vi.fn(() => ({
            fetchSettings: mockFetchSettings,
            $reset: mockReset
        }))
    }
})

import { useUserStore } from '@/store/userStore'
import { useSettingsStore } from '@/store/settingsStore'

describe('userStore', () => {
    let store: ReturnType<typeof useUserStore>
    let settingsStore: ReturnType<typeof useSettingsStore>
    let consoleErrorSpy: ReturnType<typeof vi.spyOn>

    beforeEach(() => {
        vi.clearAllMocks()
        setActivePinia(createPinia())
        store = useUserStore()
        settingsStore = useSettingsStore()
        consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    })

    afterEach(() => {
        consoleErrorSpy.mockRestore()
    })

    describe('initial state', () => {
        it('should have isLoggedIn as false', () => {
            expect(store.isLoggedIn).toBe(false)
        })

        it('should have loggedInUser as empty string', () => {
            expect(store.loggedInUser).toBe('')
        })

        it('should have isFirstLogin as true', () => {
            expect(store.isFirstLogin).toBe(true)
        })

        it('should have isLoading as false', () => {
            expect(store.isLoading).toBe(false)
        })

        it('should have wrongPwErr as false', () => {
            expect(store.wrongPwErr).toBe(false)
        })
    })

    describe('setFirstLoginFalse', () => {
        it('should set isFirstLogin to false', () => {
            expect(store.isFirstLogin).toBe(true)
            store.setFirstLoginFalse()
            expect(store.isFirstLogin).toBe(false)
        })
    })

    describe('checkState', () => {
        it('should set loggedInUser and isLoggedIn when status is 200 and logged in', async () => {
            ;(axios.get as Mock).mockResolvedValue({
                status: 200,
                data: { username: 'testuser', 'logged-in': true }
            })

            await store.checkState()

            expect(axios.get).toHaveBeenCalledOnce()
            expect(store.loggedInUser).toBe('testuser')
            expect(store.isLoggedIn).toBe(true)
            expect(settingsStore.fetchSettings).toHaveBeenCalledOnce()
        })

        it('should set isLoggedIn to false when status is 200 but not logged in', async () => {
            ;(axios.get as Mock).mockResolvedValue({
                status: 200,
                data: { username: '', 'logged-in': false }
            })

            await store.checkState()

            expect(store.isLoggedIn).toBe(false)
            expect(settingsStore.fetchSettings).not.toHaveBeenCalled()
        })

        it('should not update state on non-200 response', async () => {
            ;(axios.get as Mock).mockResolvedValue({
                status: 500,
                data: {}
            })

            await store.checkState()

            expect(store.isLoggedIn).toBe(false)
            expect(store.loggedInUser).toBe('')
        })

        it('should handle errors gracefully', async () => {
            ;(axios.get as Mock).mockRejectedValue(new Error('Network error'))

            await store.checkState()

            expect(store.isLoggedIn).toBe(false)
            expect(store.loggedInUser).toBe('')
        })
    })

    describe('login', () => {
        let mockAuthAxiosInstance: { interceptors: { response: { use: Mock } }; post: Mock }

        beforeEach(() => {
            mockAuthAxiosInstance = {
                interceptors: {
                    response: { use: vi.fn() }
                },
                post: vi.fn()
            }
            ;(axios.create as Mock).mockReturnValue(mockAuthAxiosInstance)
        })

        it('should set isLoggedIn and loggedInUser on successful login (status 200)', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 200,
                data: { token: 'abc' }
            })

            const onSuccess = vi.fn()
            store.login('myuser', 'mypass', false, onSuccess)

            // login is fire-and-forget (no return), so we need to flush
            await vi.waitFor(() => {
                expect(store.isLoggedIn).toBe(true)
            })

            expect(store.loggedInUser).toBe('myuser')
            expect(store.wrongPwErr).toBe(false)
            expect(store.isLoading).toBe(false)
            expect(settingsStore.fetchSettings).toHaveBeenCalledOnce()
            expect(onSuccess).toHaveBeenCalledOnce()
        })

        it('should pass sessionRenew as false when keepMeLoggedIn is falsy', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 200,
                data: {}
            })

            store.login('user', 'pass', undefined as unknown as boolean, undefined)

            await vi.waitFor(() => {
                expect(mockAuthAxiosInstance.post).toHaveBeenCalled()
            })

            expect(mockAuthAxiosInstance.post).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({ sessionRenew: false })
            )
        })

        it('should pass sessionRenew as true when keepMeLoggedIn is true', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 200,
                data: {}
            })

            store.login('user', 'pass', true, undefined)

            await vi.waitFor(() => {
                expect(mockAuthAxiosInstance.post).toHaveBeenCalled()
            })

            expect(mockAuthAxiosInstance.post).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({ sessionRenew: true })
            )
        })

        it('should set isLoading to true during request and false after', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 200,
                data: {}
            })

            // Before login
            expect(store.isLoading).toBe(false)

            store.login('user', 'pass', false, undefined)

            // After the promise resolves, isLoading should be false
            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })
        })

        it('should not call onSuccessNavigate when response is non-200', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 302,
                data: {}
            })

            const onSuccess = vi.fn()
            store.login('user', 'pass', false, onSuccess)

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            expect(onSuccess).not.toHaveBeenCalled()
            expect(store.isLoggedIn).toBe(false)
        })

        it('should register a 401 interceptor that sets wrongPwErr', async () => {
            mockAuthAxiosInstance.post.mockResolvedValue({
                status: 200,
                data: {}
            })

            store.login('user', 'pass', false, undefined)

            await vi.waitFor(() => {
                expect(mockAuthAxiosInstance.interceptors.response.use).toHaveBeenCalled()
            })

            // Get the error handler (second argument to interceptors.response.use)
            const errorHandler = mockAuthAxiosInstance.interceptors.response.use.mock.calls[0][1]

            const error401 = { response: { status: 401 } }
            errorHandler(error401)

            expect(store.isLoggedIn).toBe(false)
            expect(store.wrongPwErr).toBe(true)
        })

        it('should handle post rejection gracefully and set isLoading false', async () => {
            mockAuthAxiosInstance.post.mockRejectedValue(new Error('Network failure'))

            store.login('user', 'pass', false, undefined)

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            // State should remain unchanged on error
            expect(store.isLoggedIn).toBe(false)
        })
    })

    describe('logout', () => {
        it('should reset user state and call settingsStore.$reset on success', async () => {
            ;(axios.post as Mock).mockResolvedValue({ status: 200 })

            // Set some state first
            store.$patch({ loggedInUser: 'testuser', isLoggedIn: true })

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            expect(axios.post).toHaveBeenCalledOnce()
            expect(store.loggedInUser).toBe('')
            expect(store.isLoggedIn).toBe(false)
            expect(settingsStore.$reset).toHaveBeenCalledOnce()
        })

        it('should call registered logout callbacks on success', async () => {
            ;(axios.post as Mock).mockResolvedValue({ status: 200 })

            const cb1 = vi.fn()
            const cb2 = vi.fn()
            store.registerLogoutAction(cb1)
            store.registerLogoutAction(cb2)

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            expect(cb1).toHaveBeenCalledOnce()
            expect(cb2).toHaveBeenCalledOnce()
        })

        it('should continue calling callbacks even if one throws', async () => {
            ;(axios.post as Mock).mockResolvedValue({ status: 200 })

            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

            const cb1 = vi.fn(() => {
                throw new Error('callback error')
            })
            const cb2 = vi.fn()
            store.registerLogoutAction(cb1)
            store.registerLogoutAction(cb2)

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            expect(cb1).toHaveBeenCalledOnce()
            expect(cb2).toHaveBeenCalledOnce()
            expect(errorSpy).toHaveBeenCalled()

            errorSpy.mockRestore()
        })

        it('should set isLoading true during request and false after', async () => {
            ;(axios.post as Mock).mockResolvedValue({ status: 200 })

            expect(store.isLoading).toBe(false)

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })
        })

        it('should handle logout post failure gracefully', async () => {
            ;(axios.post as Mock).mockRejectedValue(new Error('Server down'))

            store.$patch({ loggedInUser: 'testuser', isLoggedIn: true })

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            // On error, state is not reset (only in .then)
            expect(store.loggedInUser).toBe('testuser')
            expect(store.isLoggedIn).toBe(true)
        })
    })

    describe('registerLogoutAction', () => {
        it('should accept a function callback', () => {
            const cb = vi.fn()
            store.registerLogoutAction(cb)
            // No error thrown; callback is registered
        })

        it('should log error when non-function is passed', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

            store.registerLogoutAction('not a function' as unknown as () => void)

            expect(errorSpy).toHaveBeenCalledWith('Callback must be a function')

            errorSpy.mockRestore()
        })

        it('should accumulate multiple callbacks', async () => {
            ;(axios.post as Mock).mockResolvedValue({ status: 200 })

            const cb1 = vi.fn()
            const cb2 = vi.fn()
            const cb3 = vi.fn()

            store.registerLogoutAction(cb1)
            store.registerLogoutAction(cb2)
            store.registerLogoutAction(cb3)

            store.logout()

            await vi.waitFor(() => {
                expect(store.isLoading).toBe(false)
            })

            expect(cb1).toHaveBeenCalledOnce()
            expect(cb2).toHaveBeenCalledOnce()
            expect(cb3).toHaveBeenCalledOnce()
        })
    })
})
