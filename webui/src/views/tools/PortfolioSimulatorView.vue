<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import VChart from 'vue-echarts'
import Button from 'primevue/button'
import { getCase, updateCase } from '@/lib/api/ToolsData'
import { computePortfolioProjection, computePortfolioExpectedReturn } from '@/lib/simulators/portfolio'
import { useToast } from 'primevue/usetoast'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const props = defineProps({ caseId: { type: Number, required: true } })

const router = useRouter()
const leftSidebarCollapsed = ref(true)

// Form inputs (defaults)
const durationYears = ref(20)
const initialContribution = ref(10000)
const monthlyContribution = ref(500)
const growthRatePct = ref(6)
const inflationPct = ref(2)
const capitalGainTaxPct = ref(3)

const TOOL_TYPE = 'portfolio-simulator'

const activeCaseName = ref('')
const activeCaseDescription = ref('')
const toast = useToast()

function getCurrentParams() {
    return {
        durationYears: durationYears.value,
        initialContribution: initialContribution.value,
        monthlyContribution: monthlyContribution.value,
        growthRatePct: growthRatePct.value,
        inflationPct: inflationPct.value,
        capitalGainTaxPct: capitalGainTaxPct.value,
    }
}

function computeExpectedAnnualReturn() {
    return computePortfolioExpectedReturn(getCurrentParams())
}

async function handleSave() {
    const payload = {
        expectedAnnualReturn: computeExpectedAnnualReturn(),
        params: getCurrentParams(),
    }
    try {
        await updateCase(TOOL_TYPE, props.caseId, {
            ...payload,
            name: activeCaseName.value,
        })
        toast.add({ severity: 'success', summary: 'Saved', detail: `"${activeCaseName.value}" updated`, life: 3000 })
    } catch (e) {
        console.error('Failed to save scenario:', e)
    }
}

function loadCaseData(cs) {
    const p = cs.params
    if (p) {
        durationYears.value = p.durationYears ?? durationYears.value
        initialContribution.value = p.initialContribution ?? initialContribution.value
        monthlyContribution.value = p.monthlyContribution ?? monthlyContribution.value
        growthRatePct.value = p.growthRatePct ?? growthRatePct.value
        inflationPct.value = p.inflationPct ?? inflationPct.value
        capitalGainTaxPct.value = p.capitalGainTaxPct ?? capitalGainTaxPct.value
    }
    activeCaseName.value = cs.name
    activeCaseDescription.value = cs.description ?? ''
}

onMounted(async () => {
    try {
        const cs = await getCase(TOOL_TYPE, props.caseId)
        loadCaseData(cs)
    } catch (e) {
        console.error('Failed to load case:', e)
    }
})

/**
 * Project portfolio value year by year with compound growth and monthly contributions.
 * Delegates to the shared pure function.
 */
const projection = computed(() => computePortfolioProjection(getCurrentParams()))

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
                <div class="flex align-items-center justify-content-between mb-3">
                    <div class="flex align-items-center gap-2">
                        <Button icon="pi pi-arrow-left" label="Back" text @click="router.push('/tools')" />
                        <span class="text-xl font-bold">Portfolio Simulator : {{ activeCaseName }}</span>
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button label="Save" icon="pi pi-save" size="small" @click="handleSave()" />
                    </div>
                </div>

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
