import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import axios from 'axios'

vi.mock('axios')

const mockedAxios = vi.mocked(axios, true)

describe('settingsStore', () => {
    let consoleErrorSpy: ReturnType<typeof vi.spyOn>

    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
        consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    })

    afterEach(() => {
        consoleErrorSpy.mockRestore()
    })

    async function importStore() {
        const { useSettingsStore } = await import('./settingsStore')
        return useSettingsStore()
    }

    describe('initial state', () => {
        it('has correct default values', async () => {
            const store = await importStore()

            expect(store.isLoaded).toBe(false)
            expect(store.isLoading).toBe(false)
            expect(store.error).toBeNull()
            expect(store.dateFormat).toBe('')
            expect(store.mainCurrency).toBe('')
            expect(store.currencies).toEqual([])
            expect(store.instruments).toBe(false)
            expect(store.marketDataSymbols).toEqual([])
            expect(store.version).toBe('')
        })

        it('hasMultipleCurrencies is false when currencies is empty', async () => {
            const store = await importStore()
            expect(store.hasMultipleCurrencies).toBe(false)
        })
    })

    describe('fetchSettings', () => {
        const mockSettingsResponse = {
            data: {
                dateFormat: 'YYYY-MM-DD',
                mainCurrency: 'USD',
                currencies: ['USD', 'EUR', 'GBP'],
                instruments: true,
                marketDataSymbols: ['AAPL', 'GOOGL'],
                version: '1.2.3',
            },
        }

        it('sets all fields on successful fetch', async () => {
            mockedAxios.get.mockResolvedValue(mockSettingsResponse)
            const store = await importStore()

            await store.fetchSettings()

            expect(store.isLoaded).toBe(true)
            expect(store.isLoading).toBe(false)
            expect(store.error).toBeNull()
            expect(store.dateFormat).toBe('YYYY-MM-DD')
            expect(store.mainCurrency).toBe('USD')
            expect(store.currencies).toEqual(['USD', 'EUR', 'GBP'])
            expect(store.instruments).toBe(true)
            expect(store.marketDataSymbols).toEqual(['AAPL', 'GOOGL'])
            expect(store.version).toBe('1.2.3')
        })

        it('calls the correct endpoint', async () => {
            mockedAxios.get.mockResolvedValue(mockSettingsResponse)
            const store = await importStore()

            await store.fetchSettings()

            expect(mockedAxios.get).toHaveBeenCalledOnce()
            expect(mockedAxios.get).toHaveBeenCalledWith(
                expect.stringContaining('/settings'),
            )
        })

        it('sets isLoading to true during fetch', async () => {
            let resolvePromise: (value: unknown) => void
            const pendingPromise = new Promise((resolve) => {
                resolvePromise = resolve
            })
            mockedAxios.get.mockReturnValue(pendingPromise as any)

            const store = await importStore()
            const fetchPromise = store.fetchSettings()

            expect(store.isLoading).toBe(true)
            expect(store.isLoaded).toBe(false)

            resolvePromise!(mockSettingsResponse)
            await fetchPromise

            expect(store.isLoading).toBe(false)
            expect(store.isLoaded).toBe(true)
        })

        it('defaults currencies to empty array when not provided', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: 'DD/MM/YYYY',
                    mainCurrency: 'EUR',
                    instruments: false,
                },
            })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.currencies).toEqual([])
            expect(store.marketDataSymbols).toEqual([])
            expect(store.version).toBe('')
        })

        it('defaults marketDataSymbols to empty array when not provided', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: 'DD/MM/YYYY',
                    mainCurrency: 'EUR',
                    currencies: ['EUR'],
                    instruments: false,
                },
            })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.marketDataSymbols).toEqual([])
        })

        it('defaults version to empty string when not provided', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: 'DD/MM/YYYY',
                    mainCurrency: 'EUR',
                    currencies: ['EUR'],
                    instruments: false,
                    marketDataSymbols: [],
                },
            })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.version).toBe('')
        })

        it('sets error on fetch failure', async () => {
            mockedAxios.get.mockRejectedValue(new Error('Network Error'))
            const store = await importStore()

            await store.fetchSettings()

            expect(store.isLoaded).toBe(false)
            expect(store.isLoading).toBe(false)
            expect(store.error).toBe('Network Error')
        })

        it('sets fallback error message when err.message is empty', async () => {
            mockedAxios.get.mockRejectedValue({ message: '' })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.error).toBe('Failed to load settings')
        })

        it('clears previous error on new fetch attempt', async () => {
            mockedAxios.get.mockRejectedValueOnce(new Error('First error'))
            const store = await importStore()

            await store.fetchSettings()
            expect(store.error).toBe('First error')

            mockedAxios.get.mockResolvedValueOnce(mockSettingsResponse)
            await store.fetchSettings()

            expect(store.error).toBeNull()
            expect(store.isLoaded).toBe(true)
        })

        it('sets isLoading back to false on error', async () => {
            mockedAxios.get.mockRejectedValue(new Error('fail'))
            const store = await importStore()

            await store.fetchSettings()

            expect(store.isLoading).toBe(false)
        })

        it('logs error to console on failure', async () => {
            const err = new Error('Network Error')
            mockedAxios.get.mockRejectedValue(err)
            const store = await importStore()

            await store.fetchSettings()

            expect(consoleErrorSpy).toHaveBeenCalledWith(
                'Failed to fetch application settings:',
                err,
            )
        })
    })

    describe('hasMultipleCurrencies', () => {
        it('returns false with zero currencies', async () => {
            const store = await importStore()
            expect(store.hasMultipleCurrencies).toBe(false)
        })

        it('returns false with exactly one currency', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: '',
                    mainCurrency: 'USD',
                    currencies: ['USD'],
                    instruments: false,
                },
            })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.hasMultipleCurrencies).toBe(false)
        })

        it('returns true with multiple currencies', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: '',
                    mainCurrency: 'USD',
                    currencies: ['USD', 'EUR'],
                    instruments: false,
                },
            })
            const store = await importStore()

            await store.fetchSettings()

            expect(store.hasMultipleCurrencies).toBe(true)
        })
    })

    describe('$reset', () => {
        it('resets all state to initial values', async () => {
            mockedAxios.get.mockResolvedValue({
                data: {
                    dateFormat: 'YYYY-MM-DD',
                    mainCurrency: 'USD',
                    currencies: ['USD', 'EUR'],
                    instruments: true,
                    marketDataSymbols: ['AAPL'],
                    version: '2.0.0',
                },
            })
            const store = await importStore()

            await store.fetchSettings()
            expect(store.isLoaded).toBe(true)
            expect(store.mainCurrency).toBe('USD')

            store.$reset()

            expect(store.isLoaded).toBe(false)
            expect(store.isLoading).toBe(false)
            expect(store.error).toBeNull()
            expect(store.dateFormat).toBe('')
            expect(store.mainCurrency).toBe('')
            expect(store.currencies).toEqual([])
            expect(store.instruments).toBe(false)
            expect(store.marketDataSymbols).toEqual([])
            expect(store.version).toBe('')
            expect(store.hasMultipleCurrencies).toBe(false)
        })

        it('clears error state', async () => {
            mockedAxios.get.mockRejectedValue(new Error('fail'))
            const store = await importStore()

            await store.fetchSettings()
            expect(store.error).toBe('fail')

            store.$reset()

            expect(store.error).toBeNull()
        })
    })
})
