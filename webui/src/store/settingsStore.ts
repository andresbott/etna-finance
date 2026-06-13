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
    const investmentInstruments = ref<boolean>(false)
    const rsu = ref<boolean>(false)
    const financialSimulator = ref<boolean>(false)
    // Feature keys the server turned on at startup despite the config disabling them.
    // The flags above remain the source of truth for gating; this is display-only provenance.
    const autoEnabled = ref<string[]>([])
    const marketDataSymbols = ref<string[]>([])
    const version = ref<string>('')
    const maxAttachmentSizeMB = ref<number>(10)

    const hasMultipleCurrencies = computed(() => currencies.value.length > 1)

    const fetchSettings = async () => {
        isLoading.value = true
        error.value = null
        try {
            const res = await axios.get(SETTINGS_ENDPOINT)
            dateFormat.value = res.data.dateFormat
            mainCurrency.value = res.data.mainCurrency
            currencies.value = res.data.currencies ?? []
            investmentInstruments.value = res.data.investmentInstruments
            rsu.value = res.data.rsu
            financialSimulator.value = res.data.financialSimulator
            autoEnabled.value = res.data.autoEnabled ?? []
            marketDataSymbols.value = res.data.marketDataSymbols ?? []
            version.value = res.data.version ?? ''
            maxAttachmentSizeMB.value = res.data.maxAttachmentSizeMB || 10
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
        investmentInstruments.value = false
        rsu.value = false
        financialSimulator.value = false
        autoEnabled.value = []
        marketDataSymbols.value = []
        version.value = ''
        maxAttachmentSizeMB.value = 10
    }

    return {
        isLoaded,
        isLoading,
        error,

        dateFormat,
        mainCurrency,
        currencies,
        investmentInstruments,
        rsu,
        financialSimulator,
        autoEnabled,
        marketDataSymbols,
        version,
        maxAttachmentSizeMB,
        hasMultipleCurrencies,

        fetchSettings,
        $reset
    }
})
