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
import SelectButton from 'primevue/selectbutton'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Dialog from 'primevue/dialog'
import { getCase, createCase, updateCase } from '@/lib/api/ToolsData'
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
const initialContribution = ref(10000)
const growthRatePct = ref(7)
const expenseRatioPct = ref(0.2)
const capitalGainTaxPct = ref(19)
const taxModel = ref('exit')

const TOOL_TYPE = 'portfolio-simulator'

const activeCaseName = ref('')
const activeCaseDescription = ref('')
const toast = useToast()

function getCurrentParams() {
    return {
        initialContribution: initialContribution.value,
        growthRatePct: growthRatePct.value,
        expenseRatioPct: expenseRatioPct.value,
        capitalGainTaxPct: capitalGainTaxPct.value,
        taxModel: taxModel.value,
    }
}

const showSaveDialog = ref(false)
const saveName = ref('')
const saveDescription = ref('')
const showEditDialog = ref(false)

function openSaveAsDialog() {
    saveName.value = activeCaseName.value ? activeCaseName.value + ' (copy)' : ''
    saveDescription.value = activeCaseDescription.value
    showSaveDialog.value = true
}

async function handleSaveAs() {
    const payload = {
        expectedAnnualReturn: computeExpectedAnnualReturn(),
        params: getCurrentParams(),
    }
    try {
        const created = await createCase(TOOL_TYPE, {
            ...payload,
            name: saveName.value,
            description: saveDescription.value,
        })
        showSaveDialog.value = false
        toast.add({ severity: 'success', summary: 'Created', detail: `"${created.name}" saved`, life: 3000 })
        router.push(`/financial-simulator/${TOOL_TYPE}/${created.id}`)
    } catch (e) {
        console.error('Failed to save scenario:', e)
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
            description: activeCaseDescription.value,
        })
        toast.add({ severity: 'success', summary: 'Saved', detail: `"${activeCaseName.value}" updated`, life: 3000 })
    } catch (e) {
        console.error('Failed to save scenario:', e)
    }
}

function loadCaseData(cs) {
    const p = cs.params
    if (p) {
        initialContribution.value = p.initialContribution ?? initialContribution.value
        growthRatePct.value = p.growthRatePct ?? growthRatePct.value
        expenseRatioPct.value = p.expenseRatioPct ?? 0
        capitalGainTaxPct.value = p.capitalGainTaxPct ?? capitalGainTaxPct.value
        taxModel.value = p.taxModel ?? 'exit'
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

const projection = computed(() => computePortfolioProjection(getCurrentParams()))

const chartColors = {
    totalInvested: '#64748b',
    netWorth: '#22c55e',
    totalGains: '#3b82f6',
    taxImpact: '#ef4444',
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
            data: ['Total Invested', 'Net Worth', 'Total Gains', 'Tax Impact']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                const idx = params[0].dataIndex
                const y = years[idx]
                const lines = [
                    `Year <strong>${y}</strong>`,
                    `Total Invested: ${formatCurrency(s.totalInvested[idx])}`,
                    `Net Worth: ${formatCurrency(s.netWorth[idx])}`,
                    `Total Gains: ${formatCurrency(s.totalGains[idx])}`,
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
            { type: 'line', data: years.map((y, i) => [y, s.totalGains[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.totalGains, width: 2 }, itemStyle: { color: chartColors.totalGains }, name: 'Total Gains' },
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
                        <Button icon="pi pi-arrow-left" label="Back" text @click="router.push('/financial-simulator')" />
                        <span class="text-xl font-bold">Portfolio Simulator : {{ activeCaseName }}</span>
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button label="Edit" icon="pi pi-pencil" size="small" outlined @click="showEditDialog = true" />
                        <Button label="Save" icon="pi pi-save" size="small" @click="handleSave()" />
                        <Button label="Save As" icon="pi pi-copy" size="small" outlined @click="openSaveAsDialog()" />
                    </div>
                </div>

                <div class="grid">
                    <div class="col-12 md:col-4">
                        <Card>
                            <template #title>Parameters</template>
                            <template #content>
                                <div class="form-grid">
                                    <div class="field">
                                        <label for="initial">Initial contribution</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="initial"
                                                v-model="initialContribution"
                                                :min="0"
                                                :max="1000000"
                                                mode="decimal"
                                                :minFractionDigits="0"
                                                :maxFractionDigits="0"
                                                class="field-input"
                                            />
                                            <Slider v-model="initialContribution" :min="0" :max="1000000" :step="5000" class="field-slider" />
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
                                            <Slider v-model="growthRatePct" :min="0" :max="30" :step="0.5" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="expense">Expense ratio / TER (%)</label>
                                        <div class="field-controls">
                                            <InputNumber
                                                id="expense"
                                                v-model="expenseRatioPct"
                                                :min="0"
                                                :max="5"
                                                :minFractionDigits="2"
                                                :maxFractionDigits="2"
                                                class="field-input"
                                            />
                                            <Slider v-model="expenseRatioPct" :min="0" :max="5" :step="0.05" class="field-slider" />
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
                                    <div class="field">
                                        <label>Tax model</label>
                                        <SelectButton
                                            v-model="taxModel"
                                            :options="[
                                                { label: 'Exit Tax', value: 'exit' },
                                                { label: 'Annual Tax', value: 'annual' },
                                            ]"
                                            optionLabel="label"
                                            optionValue="value"
                                        />
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
                                        <span class="result-label">Total Gains</span>
                                        <span class="result-value">{{ formatCurrency(projection.totalGain) }}</span>
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

                <!-- Save As Dialog -->
                <Dialog v-model:visible="showSaveDialog" header="Save as New Scenario" :modal="true" :style="{ width: '30rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="caseName">Name</label>
                            <InputText id="caseName" v-model="saveName" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="caseDesc">Description</label>
                            <Textarea id="caseDesc" v-model="saveDescription" rows="3" class="w-full" />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Save" @click="handleSaveAs" :disabled="!saveName" />
                        <Button label="Cancel" text @click="showSaveDialog = false" />
                    </template>
                </Dialog>

                <!-- Edit Name & Description Dialog -->
                <Dialog v-model:visible="showEditDialog" header="Edit Scenario" :modal="true" :style="{ width: '35rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="editName">Name</label>
                            <InputText id="editName" v-model="activeCaseName" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="editDesc">Description</label>
                            <Textarea id="editDesc" v-model="activeCaseDescription" rows="3" class="w-full" />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Close" text @click="showEditDialog = false" />
                    </template>
                </Dialog>
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
