<!-- webui/src/components/marketdata/ChartToolbar.vue -->
<script setup lang="ts">
import { ref } from 'vue'
import { storeToRefs } from 'pinia'
import SelectButton from 'primevue/selectbutton'
import ToggleButton from 'primevue/togglebutton'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import { useChartControls } from '@/composables/useChartControls'
import IndicatorSettings from '@/components/marketdata/IndicatorSettings.vue'
import type { PriceHistoryRange } from '@/utils/dateRange'

const store = useChartControls()
const { selectedRange, sma, ema, bollinger, rsi, macd, pe } = storeToRefs(store)

const rangeOptions: { value: PriceHistoryRange; label: string }[] = [
    { value: '1m', label: '1M' },
    { value: '3m', label: '3M' },
    { value: '6m', label: '6M' },
    { value: '1y', label: '1Y' },
    { value: 'max', label: 'ALL' }
]

const helpVisible = ref(false)

const indicatorHelp = [
    {
        title: 'Simple Moving Average (SMA)',
        description: 'The SMA calculates the average closing price over a specified number of periods. For example, a 50-day SMA adds up the last 50 closing prices and divides by 50. Each day, the oldest price drops off and the newest is added.',
        whyItMatters: 'SMA smooths out price noise to reveal the underlying trend. When the price is above the SMA, the trend is generally bullish; below it, bearish. Crossovers between a short-period SMA (e.g. 50) and a long-period SMA (e.g. 200) are classic buy/sell signals — the "golden cross" (50 crosses above 200) and "death cross" (50 crosses below 200).'
    },
    {
        title: 'Exponential Moving Average (EMA)',
        description: 'The EMA is similar to the SMA but gives more weight to recent prices, making it react faster to price changes. It uses an exponential smoothing factor based on the period length.',
        whyItMatters: 'Because EMA responds faster to recent price action, it is preferred by short-term traders who need quicker signals. EMA crossovers (e.g. 12 and 26 periods) are used similarly to SMA crossovers but trigger earlier. The EMA is also the basis for other indicators like MACD.'
    },
    {
        title: 'Bollinger Bands',
        description: 'Bollinger Bands consist of three lines: a middle SMA and two bands plotted at a specified number of standard deviations above and below it. The default is a 20-period SMA with bands at 2 standard deviations.',
        whyItMatters: 'The bands expand when volatility is high and contract when it is low. Prices touching or breaking the upper band may indicate overbought conditions, while touching the lower band may indicate oversold conditions. A "squeeze" (narrow bands) often precedes a strong price move in either direction.'
    },
    {
        title: 'Relative Strength Index (RSI)',
        description: 'RSI measures the speed and magnitude of recent price changes on a scale of 0 to 100. It compares average gains to average losses over a specified period (typically 14 days).',
        whyItMatters: 'RSI above 70 suggests the asset has been bought aggressively and may be due for a pullback or consolidation — this is called "overbought." RSI below 30 suggests heavy selling pressure that may be exhausted, meaning the price could recover — this is called "oversold." These are not automatic buy/sell signals; a strong trend can keep RSI overbought or oversold for extended periods. A more reliable signal is divergence: if the price makes a new high but RSI makes a lower high, it means each rally is carried by less momentum, warning that the uptrend may be losing steam. The reverse (price makes a new low, RSI makes a higher low) can signal a potential bottom.'
    },
    {
        title: 'Moving Average Convergence Divergence (MACD)',
        description: 'MACD is calculated by subtracting the 26-period EMA from the 12-period EMA. A 9-period EMA of the MACD line (the "signal line") is plotted on top. The histogram shows the difference between the MACD line and signal line.',
        whyItMatters: 'When the MACD line crosses above the signal line, it is a bullish signal; crossing below is bearish. The histogram visualizes the momentum — growing bars mean strengthening momentum, shrinking bars mean it is fading. MACD is one of the most widely used momentum indicators for identifying trend direction and strength.'
    },
    {
        title: 'Price-to-Earnings Ratio (P/E)',
        description: 'The P/E ratio is calculated by dividing the current stock price by the trailing twelve months (TTM) earnings per share (EPS). TTM EPS is the sum of the four most recent quarterly EPS values from SEC filings. Because EPS updates quarterly while price moves daily, the P/E ratio changes every trading day.',
        whyItMatters: 'P/E measures how much investors are willing to pay per dollar of earnings. A high P/E (e.g. 30+) can mean the market expects strong future growth, or that the stock is overvalued. A low P/E (e.g. below 15) may indicate a bargain or declining earnings expectations. Comparing P/E over time for the same stock shows how its valuation has shifted. Comparing across stocks in the same sector helps identify relative value. Note: P/E is only meaningful for profitable companies — it is not shown when TTM earnings are zero or negative.'
    }
]
</script>

<template>
    <div class="chart-toolbar">
        <div class="toolbar-row">
            <SelectButton
                v-model="selectedRange"
                :options="rangeOptions"
                optionLabel="label"
                optionValue="value"
                class="range-selector"
            />
        </div>
        <div class="toolbar-row">
            <div class="indicator-group">
                <ToggleButton
                    v-model="sma.enabled"
                    onLabel="SMA" offLabel="SMA"
                    class="indicator-toggle"
                />
                <IndicatorSettings
                    v-if="sma.enabled"
                    type="sma"
                    :params="sma"
                    @update:params="(v) => Object.assign(sma, v)"
                />
            </div>
            <div class="indicator-group">
                <ToggleButton
                    v-model="ema.enabled"
                    onLabel="EMA" offLabel="EMA"
                    class="indicator-toggle"
                />
                <IndicatorSettings
                    v-if="ema.enabled"
                    type="ema"
                    :params="ema"
                    @update:params="(v) => Object.assign(ema, v)"
                />
            </div>
            <div class="indicator-group">
                <ToggleButton
                    v-model="bollinger.enabled"
                    onLabel="Bollinger" offLabel="Bollinger"
                    class="indicator-toggle"
                />
                <IndicatorSettings
                    v-if="bollinger.enabled"
                    type="bollinger"
                    :params="bollinger"
                    @update:params="(v) => Object.assign(bollinger, v)"
                />
            </div>
            <div class="indicator-group">
                <ToggleButton
                    v-model="rsi.enabled"
                    onLabel="RSI" offLabel="RSI"
                    class="indicator-toggle"
                />
                <IndicatorSettings
                    v-if="rsi.enabled"
                    type="rsi"
                    :params="rsi"
                    @update:params="(v) => Object.assign(rsi, v)"
                />
            </div>
            <div class="indicator-group">
                <ToggleButton
                    v-model="macd.enabled"
                    onLabel="MACD" offLabel="MACD"
                    class="indicator-toggle"
                />
                <IndicatorSettings
                    v-if="macd.enabled"
                    type="macd"
                    :params="macd"
                    @update:params="(v) => Object.assign(macd, v)"
                />
            </div>
            <div class="indicator-group">
                <ToggleButton
                    v-model="pe.enabled"
                    onLabel="P/E" offLabel="P/E"
                    class="indicator-toggle"
                />
            </div>
            <Button
                label="?"
                severity="secondary"
                text
                class="indicator-toggle"
                @click="helpVisible = true"
            />
        </div>

        <Dialog
            v-model:visible="helpVisible"
            header="Technical Indicators"
            modal
            :style="{ width: '70rem', maxWidth: '90vw' }"
        >
            <div class="indicator-help">
                <div v-for="(item, idx) in indicatorHelp" :key="idx" class="indicator-help-section">
                    <h4>{{ item.title }}</h4>
                    <p>{{ item.description }}</p>
                    <p class="why-label">Why it matters</p>
                    <p>{{ item.whyItMatters }}</p>
                </div>
            </div>
        </Dialog>
    </div>
</template>

<style scoped>
.chart-toolbar {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
    flex-shrink: 0;
}

.toolbar-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
}

.indicator-group {
    display: flex;
    align-items: center;
    gap: 0.15rem;
}

.indicator-toggle {
    font-size: 0.8rem;
}

.indicator-toggle :deep(.p-togglebutton) {
    padding: 0.3rem 0.6rem;
    font-size: 0.8rem;
}

.indicator-help-section {
    padding-bottom: 1rem;
    margin-bottom: 1rem;
    border-bottom: 1px solid var(--p-surface-200);
}

.indicator-help-section:last-child {
    border-bottom: none;
    margin-bottom: 0;
    padding-bottom: 0;
}

.indicator-help h4 {
    margin: 0 0 0.35rem 0;
    font-size: 0.95rem;
    color: var(--p-text-color);
}

.indicator-help p {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.5;
    color: var(--p-text-muted-color);
}

.indicator-help .why-label {
    margin-top: 0.5rem;
    font-weight: 600;
    color: var(--p-text-color);
    font-size: 0.85rem;
}
</style>
