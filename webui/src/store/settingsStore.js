import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const SETTINGS_ENDPOINT = `${API_BASE_URL}/settings`

export const useSettingsStore = defineStore('settings', () => {
    const isLoaded = ref(false)
    const isLoading = ref(false)
    const error = ref(null)

    const dateFormat = ref('')
    const mainCurrency = ref('')
    const currencies = ref([])
    const instruments = ref(false)
    const marketDataSymbols = ref([])
    const version = ref('')

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
            marketDataSymbols.value = res.data.marketDataSymbols ?? []
            version.value = res.data.version ?? ''
            isLoaded.value = true
        } catch (err) {
            console.error('Failed to fetch application settings:', err)
            error.value = err.message || 'Failed to load settings'
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
        marketDataSymbols,
        version,
        hasMultipleCurrencies,

        fetchSettings,
        $reset
    }
})
