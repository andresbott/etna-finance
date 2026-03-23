import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const SETTINGS_ENDPOINT = `${API_BASE_URL}/settings`

export const useSettingsStore = defineStore('settings', () => {
    const isLoaded = ref<boolean>(false)
    const isLoading = ref<boolean>(false)
    const error = ref<string | null>(null)

    const dateFormat = ref<string>('')
    const mainCurrency = ref<string>('')
    const currencies = ref<string[]>([])
    const instruments = ref<boolean>(false)
    const rsu = ref<boolean>(false)
    const tools = ref<boolean>(false)
    const marketDataSymbols = ref<string[]>([])
    const version = ref<string>('')

    const hasMultipleCurrencies = computed(() => currencies.value.length > 1)

    const fetchSettings = async () => {
        isLoading.value = true
        error.value = null
        try {
            const res = await axios.get(SETTINGS_ENDPOINT)
            dateFormat.value = res.data.dateFormat
            mainCurrency.value = res.data.mainCurrency
            currencies.value = res.data.currencies ?? []
            instruments.value = res.data.instruments
            rsu.value = res.data.rsu
            tools.value = res.data.tools
            marketDataSymbols.value = res.data.marketDataSymbols ?? []
            version.value = res.data.version ?? ''
            isLoaded.value = true
        } catch (err: unknown) {
            console.error('Failed to fetch application settings:', err)
            error.value = err instanceof Error ? err.message : 'Failed to load settings'
        } finally {
            isLoading.value = false
        }
    }

    const $reset = () => {
        isLoaded.value = false
        isLoading.value = false
        error.value = null
        dateFormat.value = ''
        mainCurrency.value = ''
        currencies.value = []
        instruments.value = false
        rsu.value = false
        tools.value = false
        marketDataSymbols.value = []
        version.value = ''
    }

    return {
        isLoaded,
        isLoading,
        error,

        dateFormat,
        mainCurrency,
        currencies,
        instruments,
        rsu,
        tools,
        marketDataSymbols,
        version,
        hasMultipleCurrencies,

        fetchSettings,
        $reset
    }
})
