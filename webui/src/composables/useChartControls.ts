// webui/src/composables/useChartControls.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { IndicatorParams } from '@/composables/useIndicators'
import type { PriceHistoryRange } from '@/utils/dateRange'

export const useChartControls = defineStore('chartControls', () => {
    const selectedRange = ref<PriceHistoryRange>('6m')

    const sma = ref({ enabled: false, period1: 50, period2: 200, showSecond: false })
    const ema = ref({ enabled: false, period1: 12, period2: 26, showSecond: false })
    const bollinger = ref({ enabled: false, period: 20, stdDev: 2 })
    const rsi = ref({ enabled: false, period: 14 })
    const macd = ref({ enabled: false, fast: 12, slow: 26, signal: 9 })
    const pe = ref({ enabled: false })

    const indicatorParams = computed<IndicatorParams>(() => ({
        sma: sma.value,
        ema: ema.value,
        bollinger: bollinger.value,
        rsi: rsi.value,
        macd: macd.value
    }))

    const hasSubPanels = computed(() => rsi.value.enabled || macd.value.enabled || pe.value.enabled)

    return {
        selectedRange,
        sma,
        ema,
        bollinger,
        rsi,
        macd,
        pe,
        indicatorParams,
        hasSubPanels
    }
})
