<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const leftSidebarCollapsed = ref(true)

// Form inputs (defaults)
const durationYears = ref(20)
const initialContribution = ref(10000)
const monthlyContribution = ref(500)
const growthRatePct = ref(6)
const inflationPct = ref(2)
const capitalGainTaxPct = ref(3)

/**
 * Project portfolio value month by month with compound growth and monthly contributions.
 * Returns { years: number[], values: number[], totalContributions, finalValue, finalValueAfterTax }.
 */
const projection = computed(() => {
    const years = durationYears.value
    const initial = initialContribution.value ?? 0
    const monthly = monthlyContribution.value ?? 0
    const growthPct = growthRatePct.value ?? 0
    const taxPct = capitalGainTaxPct.value ?? 0

    if (years <= 0) {
        return {
            years: [0],
            values: [initial],
            totalContributions: initial,
            finalValue: initial,
            finalValueAfterTax: initial,
            realFinalValue: initial,
            totalGain: 0,
            taxPaid: 0,
            inflationImpact: 0,
            inflationAdjustedGains: 0,
            series: {
                totalInvested: [initial],
                netWorth: [initial],
                inflationAdjustedNetWorth: [initial],
                totalGains: [0],
                taxImpact: [0],
                inflationAdjustedGains: [0]
            }
        }
    }

    const monthlyRate = Math.pow(1 + growthPct / 100, 1 / 12) - 1
    const totalContributions = initial + monthly * 12 * years
    const taxRate = taxPct / 100

    // Year-over-year tax: each year pay (tax rate)% of end-of-year net worth (before tax).
    // Next year starts with (end-of-year after tax). E.g. 100/mo, 0% growth, 1% tax:
    //   Year 1: net worth 1200, tax 12, after tax 1188.
    //   Year 2: 1188 + 12*100 = 2388, tax 23.88, after tax 2364.12.
    const yearLabels = [0]
    const yearValues = [initial]
    const taxPerYear = [0] // tax paid in that year only (chart series)
    let balance = initial
    let cumulativeTax = 0

    for (let y = 1; y <= years; y++) {
        // 12 months: add monthly contribution then apply growth each month
        for (let m = 0; m < 12; m++) {
            balance = (balance + monthly) * (1 + monthlyRate)
        }
        const netWorthBeforeTax = balance
        const taxThisYear = netWorthBeforeTax * taxRate // e.g. 1% of 1200 = 12, 1% of 2388 = 23.88
        cumulativeTax += taxThisYear
        balance = netWorthBeforeTax - taxThisYear // carry after-tax balance into next year

        yearLabels.push(y)
        yearValues.push(balance)
        taxPerYear.push(taxThisYear)
    }

    const finalValueAfterTax = balance
    const taxPaid = cumulativeTax
    const gain = finalValueAfterTax + taxPaid - totalContributions // pre-tax total gain
    const inflationPctVal = inflationPct.value ?? 0
    const inflationFactor = 1 + inflationPctVal / 100
    const realFinalValue = inflationPctVal > 0
        ? finalValueAfterTax / Math.pow(inflationFactor, years)
        : finalValueAfterTax
    const inflationImpactTotal = finalValueAfterTax - realFinalValue
    const realCostBasis = inflationPctVal > 0
        ? totalContributions / Math.pow(inflationFactor, years)
        : totalContributions
    const inflationAdjustedGainsFinal = realFinalValue - realCostBasis

    // Per-year series for the chart
    const totalInvestedSeries = yearLabels.map((yr) => initial + monthly * 12 * yr)
    const netWorthSeries = yearValues
    const cumulativeTaxByYear = taxPerYear.map((_, i) => taxPerYear.slice(0, i + 1).reduce((a, b) => a + b, 0))
    const totalGainsSeries = yearLabels.map((_, i) => netWorthSeries[i] + cumulativeTaxByYear[i] - totalInvestedSeries[i])
    const taxImpactSeries = cumulativeTaxByYear // cumulative tax so chart last value matches table (e.g. 0, 12, 35.88)
    // After-tax value at each year, in year-0 purchasing power (so final point = realFinalValue)
    const inflationAdjustedNetWorthSeries = yearLabels.map((y, i) =>
        inflationPctVal > 0 ? netWorthSeries[i] / Math.pow(inflationFactor, y) : netWorthSeries[i]
    )
    const realCostBasisSeries = yearLabels.map((y, i) =>
        inflationPctVal > 0 ? totalInvestedSeries[i] / Math.pow(inflationFactor, y) : totalInvestedSeries[i]
    )
    const inflationAdjustedGainsSeries = yearLabels.map((_, i) =>
        inflationAdjustedNetWorthSeries[i] - realCostBasisSeries[i]
    )

    return {
        years: yearLabels,
        values: yearValues,
        totalContributions,
        finalValue: finalValueAfterTax,
        finalValueAfterTax,
        realFinalValue,
        totalGain: gain,
        taxPaid,
        inflationImpact: inflationImpactTotal,
        inflationAdjustedGains: inflationAdjustedGainsFinal,
        series: {
            totalInvested: totalInvestedSeries,
            netWorth: netWorthSeries,
            inflationAdjustedNetWorth: inflationAdjustedNetWorthSeries,
            totalGains: totalGainsSeries,
            taxImpact: taxImpactSeries,
            inflationAdjustedGains: inflationAdjustedGainsSeries
        }
    }
})

const chartColors = {
    totalInvested: '#64748b',
    netWorth: '#22c55e',
    inflationAdjustedNetWorth: '#0d9488',
    totalGains: '#3b82f6',
    taxImpact: '#ef4444',
    inflationAdjustedGains: '#8b5cf6'
}

const chartOption = computed(() => {
    const { years, series } = projection.value
    if (!series) return {}
    const s = series
    return {
        animation: true,
        legend: {
            type: 'scroll',
            bottom: 0,
            data: ['Total Invested', 'Net Worth', 'Inflation Adjusted Net Worth', 'Total Gains', 'Inflation Adjusted Gains', 'Tax Impact']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                const idx = params[0].dataIndex
                const y = years[idx]
                const lines = [
                    `Year <strong>${y.toFixed(1)}</strong>`,
                    `Total Invested: ${formatCurrency(s.totalInvested[idx])}`,
                    `Net Worth: ${formatCurrency(s.netWorth[idx])}`,
                    `Inflation Adjusted Net Worth: ${formatCurrency(s.inflationAdjustedNetWorth[idx])}`,
                    `Total Gains: ${formatCurrency(s.totalGains[idx])}`,
                    `Inflation Adjusted Gains: ${formatCurrency(s.inflationAdjustedGains[idx])}`,
                    `Tax Impact: −${formatCurrency(s.taxImpact[idx])}`
                ]
                return lines.join('<br/>')
            }
        },
        xAxis: {
            type: 'value',
            name: 'Year',
            nameLocation: 'middle',
            nameGap: 25,
            axisLabel: { formatter: (v) => v + 'y' },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        yAxis: {
            type: 'value',
            name: 'Value',
            axisLabel: { formatter: (v) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: [
            { type: 'line', data: years.map((y, i) => [y, s.totalInvested[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.totalInvested, width: 2 }, itemStyle: { color: chartColors.totalInvested }, name: 'Total Invested' },
            { type: 'line', data: years.map((y, i) => [y, s.netWorth[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.netWorth, width: 2.5 }, itemStyle: { color: chartColors.netWorth }, name: 'Net Worth' },
            { type: 'line', data: years.map((y, i) => [y, s.inflationAdjustedNetWorth[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.inflationAdjustedNetWorth, width: 2 }, itemStyle: { color: chartColors.inflationAdjustedNetWorth }, name: 'Inflation Adjusted Net Worth' },
            { type: 'line', data: years.map((y, i) => [y, s.totalGains[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.totalGains, width: 2 }, itemStyle: { color: chartColors.totalGains }, name: 'Total Gains' },
            { type: 'line', data: years.map((y, i) => [y, s.inflationAdjustedGains[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.inflationAdjustedGains, width: 2 }, itemStyle: { color: chartColors.inflationAdjustedGains }, name: 'Inflation Adjusted Gains' },
            { type: 'line', data: years.map((y, i) => [y, s.taxImpact[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.taxImpact, width: 2 }, itemStyle: { color: chartColors.taxImpact }, name: 'Tax Impact' }
        ]
    }
})

function formatCurrency(value) {
    const n = Number(value)
    if (n !== n) return '0' // NaN
    return new Intl.NumberFormat('en-US', {
        style: 'decimal',
        minimumFractionDigits: 0,
        maximumFractionDigits: 0
    }).format(n)
}

function formatCurrencyShort(value) {
    if (value >= 1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value >= 1_000) return (value / 1_000).toFixed(0) + 'k'
    return formatCurrency(value)
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="grid">
                    <div class="col-12 md:col-4">
                        <Card>
                            <template #title>Parameters</template>
                            <template #content>
                                <div class="form-grid">
                                    <div class="field">
                                        <label for="duration">Duration (years)</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="duration"
                                                v-model="durationYears"
                                                :min="1"
                                                :max="50"
                                                class="field-input"
                                            />
                                            <Slider v-model="durationYears" :min="1" :max="50" :step="1" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="initial">Initial contribution</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="initial"
                                                v-model="initialContribution"
                                                :min="0"
                                                :max="200000"
                                                mode="decimal"
                                                :minFractionDigits="0"
                                                :maxFractionDigits="0"
                                                class="field-input"
                                            />
                                            <Slider v-model="initialContribution" :min="0" :max="200000" :step="1000" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="monthly">Monthly contribution</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="monthly"
                                                v-model="monthlyContribution"
                                                :min="0"
                                                :max="5000"
                                                mode="decimal"
                                                :minFractionDigits="0"
                                                :maxFractionDigits="0"
                                                class="field-input"
                                            />
                                            <Slider v-model="monthlyContribution" :min="0" :max="5000" :step="50" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="growth">Annual growth rate (%)</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="growth"
                                                v-model="growthRatePct"
                                                :min="0"
                                                :max="100"
                                                :minFractionDigits="1"
                                                :maxFractionDigits="2"
                                                class="field-input"
                                            />
                                            <Slider v-model="growthRatePct" :min="0" :max="20" :step="0.5" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="inflation">Inflation (%)</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="inflation"
                                                v-model="inflationPct"
                                                :min="0"
                                                :max="50"
                                                :minFractionDigits="1"
                                                :maxFractionDigits="2"
                                                class="field-input"
                                            />
                                            <Slider v-model="inflationPct" :min="0" :max="10" :step="0.5" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="tax">Capital gain tax (%)</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="tax"
                                                v-model="capitalGainTaxPct"
                                                :min="0"
                                                :max="100"
                                                :minFractionDigits="1"
                                                :maxFractionDigits="2"
                                                class="field-input"
                                            />
                                            <Slider v-model="capitalGainTaxPct" :min="0" :max="50" :step="1" class="field-slider" />
                                        </div>
                                    </div>
                                </div>
                            </template>
                        </Card>
                    </div>
                    <div class="col-12 md:col-8">
                        <Card>
                            <template #title>Projection</template>
                            <template #content>
                                <div class="chart-wrap">
                                    <VChart :option="chartOption" autoresize class="chart" />
                                </div>
                                <div class="results">
                                    <div class="result-row">
                                        <span class="result-label">Total Invested</span>
                                        <span class="result-value">{{ formatCurrency(projection.totalContributions) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Net Worth</span>
                                        <span class="result-value font-bold">{{ formatCurrency(projection.finalValueAfterTax) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Inflation adjusted net worth</span>
                                        <span class="result-value">{{ formatCurrency(projection.realFinalValue) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Total Gains</span>
                                        <span class="result-value">{{ formatCurrency(projection.totalGain) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Inflation adjusted gains</span>
                                        <span class="result-value">{{ formatCurrency(projection.inflationAdjustedGains) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Inflation Impact</span>
                                        <span class="result-value">−{{ formatCurrency(projection.inflationImpact) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Tax Impact</span>
                                        <span class="result-value">−{{ formatCurrency(projection.taxPaid) }}</span>
                                    </div>
                                </div>
                            </template>
                        </Card>
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.form-grid {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
}

.field-controls {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    width: 100%;
}

.field-input {
    width: 100%;
}

.field-input :deep(.p-inputnumber) {
    width: 100%;
}

.field-slider {
    width: 100%;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}

:deep(.p-card-content) {
    overflow: visible;
}

.chart-wrap {
    margin-bottom: 1.5rem;
}

.chart {
    height: 380px;
    width: 100%;
}

.results {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding-top: 1rem;
    margin-top: 1rem;
    border-top: 1px solid var(--surface-200);
    width: 100%;
}

.result-row {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    min-height: 1.5rem;
    width: 100%;
}

.result-label {
    color: var(--text-color-secondary);
}

.result-value {
    font-weight: 600;
}

.result-final .result-value {
    font-size: 1.05rem;
}
</style>
