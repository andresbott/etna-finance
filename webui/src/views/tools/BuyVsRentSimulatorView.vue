<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import ToggleSwitch from 'primevue/toggleswitch'
import Divider from 'primevue/divider'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { getCase, createCase, updateCase, listCases } from '@/lib/api/ToolsData'
import type { BuyVsRentSimulatorParams, RealEstateSimulatorParams, CaseStudy } from '@/lib/api/ToolsData'
import { computeBuyVsRentProjection } from '@/lib/simulators/buyVsRent'
import { useToast } from 'primevue/usetoast'

const props = defineProps<{ caseId: number }>()

const router = useRouter()

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const leftSidebarCollapsed = ref(true)

// ── Form state ──────────────────────────────────────────────────────
// Buy scenario
const purchasePrice = ref(1000000)
const cashEquity = ref(100000)
const additionalEquity = ref<Array<{ name: string; amount: number }>>([])
const mortgages = ref<Array<{
    name: string
    splitPct: number
    interestRate: number
    termYears: number
    amortize: boolean
}>>([
    { name: '1st Mortgage', splitPct: 66, interestRate: 1.5, termYears: 25, amortize: false },
    { name: '2nd Mortgage', splitPct: 34, interestRate: 1.8, termYears: 15, amortize: true }
])
const propertyTax = ref(0)
const insurance = ref(0)
const maintenanceCost = ref(0)
const otherCosts = ref(0)
const incidentalPct = ref(1)
const housingPriceIncreasePct = ref(2)

// Rent scenario
const currentMonthlyRent = ref(2300)
const rentIncreasePct = ref(1.5)
const etfReturnPct = ref(7)

// ── Save / load state ───────────────────────────────────────────────
const activeCaseName = ref('New Scenario')
const activeCaseDescription = ref('')
const showSaveDialog = ref(false)
const saveName = ref('')
const saveDescription = ref('')
const showEditDialog = ref(false)
const showHelpDialog = ref(false)
const helpDialogTitle = ref('')
const helpDialogContent = ref('')
const toast = useToast()

// ── Import from Real Estate ──────────────────────────────────────────
const showImportDialog = ref(false)
const realEstateCases = ref<CaseStudy<RealEstateSimulatorParams>[]>([])
const selectedImportCase = ref<CaseStudy<RealEstateSimulatorParams> | null>(null)
const loadingImport = ref(false)

async function openImportDialog() {
    selectedImportCase.value = null
    showImportDialog.value = true
    loadingImport.value = true
    try {
        realEstateCases.value = await listCases<RealEstateSimulatorParams>('real-estate-simulator')
    } catch (e) {
        console.error('Failed to load real estate cases:', e)
    } finally {
        loadingImport.value = false
    }
}

function handleImportConfirm() {
    if (!selectedImportCase.value) return
    importFromRealEstate(selectedImportCase.value)
}

function importFromRealEstate(cs: CaseStudy<RealEstateSimulatorParams>) {
    const p = cs.params as RealEstateSimulatorParams
    purchasePrice.value = p.purchasePrice ?? purchasePrice.value
    cashEquity.value = p.cashEquity ?? cashEquity.value
    additionalEquity.value = (p.additionalEquity ?? []).map(e => ({ ...e }))
    mortgages.value = (p.mortgages ?? []).map(m => ({
        name: m.name,
        splitPct: m.splitPct ?? 100,
        interestRate: m.interestRate,
        termYears: m.termYears,
        amortize: m.amortize,
    }))
    propertyTax.value = p.propertyTax ?? propertyTax.value
    insurance.value = p.insurance ?? insurance.value
    maintenanceCost.value = p.maintenance ?? maintenanceCost.value
    otherCosts.value = p.otherCosts ?? otherCosts.value
    incidentalPct.value = p.incidentalPct ?? 1
    housingPriceIncreasePct.value = p.housingPriceIncreasePct ?? 2
    showImportDialog.value = false
    toast.add({ severity: 'success', summary: `Imported from "${cs.name}"`, life: 2000 })
}

function openHelp(title: string, content: string) {
    helpDialogTitle.value = title
    helpDialogContent.value = content
    showHelpDialog.value = true
}

// ── Dynamic list helpers ────────────────────────────────────────────
function addEquitySource() {
    additionalEquity.value.push({ name: '', amount: 0 })
}

function removeEquitySource(index: number) {
    additionalEquity.value.splice(index, 1)
}

function addMortgage() {
    const count = mortgages.value.length + 1
    const evenPct = Math.round(100 / count)
    mortgages.value.forEach(m => { m.splitPct = evenPct })
    mortgages.value.push({
        name: `Mortgage ${count}`,
        splitPct: 100 - evenPct * (count - 1),
        interestRate: 1.5,
        termYears: 25,
        amortize: true
    })
}

function removeMortgage(index: number) {
    mortgages.value.splice(index, 1)
    if (mortgages.value.length === 1) {
        mortgages.value[0].splitPct = 100
    }
}

function updateSplitPct(index: number, value: number) {
    mortgages.value[index].splitPct = value
    if (mortgages.value.length === 2) {
        const other = index === 0 ? 1 : 0
        mortgages.value[other].splitPct = Math.max(0, 100 - value)
    }
}

function updatePrincipal(index: number, value: number) {
    const needed = totalMortgageNeeded.value
    if (needed <= 0) return
    const pct = Math.min(100, Math.max(0, (value / needed) * 100))
    updateSplitPct(index, pct)
}

// ── Derived values ──────────────────────────────────────────────────
const totalEquity = computed(() => {
    const additional = additionalEquity.value.reduce((sum, e) => sum + (e.amount ?? 0), 0)
    return (cashEquity.value ?? 0) + additional
})

const totalMortgageNeeded = computed(() => Math.max(0, (purchasePrice.value ?? 0) - totalEquity.value))

function mortgagePrincipal(m: { splitPct: number }): number {
    return totalMortgageNeeded.value * (m.splitPct ?? 0) / 100
}

const totalSplitPct = computed(() => mortgages.value.reduce((sum, m) => sum + (m.splitPct ?? 0), 0))

const incidentalCost = computed(() => (purchasePrice.value ?? 0) * (incidentalPct.value ?? 0) / 100)

const totalRecurringCosts = computed(() => {
    return (propertyTax.value ?? 0) + (insurance.value ?? 0) + (maintenanceCost.value ?? 0) + (otherCosts.value ?? 0) + incidentalCost.value
})

const totalMonthlyMortgagePayments = computed(() => {
    return mortgages.value.reduce((sum, m) => {
        const principal = mortgagePrincipal(m)
        const annualRate = m.interestRate
        const termYears = m.termYears
        if (principal <= 0 || annualRate < 0 || termYears <= 0) return sum
        if (!m.amortize) return sum + principal * (annualRate / 100) / 12
        const monthlyRate = annualRate / 100 / 12
        if (monthlyRate === 0) return sum + principal / (termYears * 12)
        const months = termYears * 12
        const factor = Math.pow(1 + monthlyRate, months)
        return sum + principal * (monthlyRate * factor) / (factor - 1)
    }, 0)
})

const totalMonthlyBuyCost = computed(() => {
    return totalMonthlyMortgagePayments.value + totalRecurringCosts.value / 12
})

const monthlyCostDifference = computed(() => {
    return totalMonthlyBuyCost.value - (currentMonthlyRent.value ?? 0)
})

// ── Params helper ───────────────────────────────────────────────────
function getCurrentParams(): BuyVsRentSimulatorParams {
    return {
        purchasePrice: purchasePrice.value,
        cashEquity: cashEquity.value,
        additionalEquity: additionalEquity.value.map(e => ({ ...e })),
        mortgages: mortgages.value.map(m => ({ ...m })),
        propertyTax: propertyTax.value,
        insurance: insurance.value,
        maintenance: maintenanceCost.value,
        otherCosts: otherCosts.value,
        incidentalPct: incidentalPct.value,
        housingPriceIncreasePct: housingPriceIncreasePct.value,
        currentMonthlyRent: currentMonthlyRent.value,
        rentIncreasePct: rentIncreasePct.value,
        etfReturnPct: etfReturnPct.value,
    }
}

// ── Chart ───────────────────────────────────────────────────────────
const chartProjection = computed(() => computeBuyVsRentProjection(getCurrentParams()))

const chartOption = computed(() => {
    const p = chartProjection.value
    return {
        animation: true,
        legend: {
            type: 'scroll',
            bottom: 0,
            data: ['Buy: Net Worth', 'Rent + Invest: Net Worth']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params: Array<{ dataIndex: number }>) => {
                const idx = params[0].dataIndex
                const y = p.yearLabels[idx]
                const diff = p.buyNetWorth[idx] - p.rentNetWorth[idx]
                return [
                    `Year <strong>${y}</strong>`,
                    `Buy: ${formatCurrency(p.buyNetWorth[idx])}`,
                    `Rent + Invest: ${formatCurrency(p.rentNetWorth[idx])}`,
                    `<strong>Difference: ${formatCurrency(diff)}</strong>`
                ].join('<br/>')
            }
        },
        xAxis: {
            type: 'value',
            name: 'Year',
            nameLocation: 'middle',
            nameGap: 25,
            axisLabel: { formatter: (v: number) => v + 'y' },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        yAxis: {
            type: 'value',
            name: 'Net Worth',
            axisLabel: { formatter: (v: number) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: [
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.buyNetWorth[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: '#22c55e', width: 2.5 }, itemStyle: { color: '#22c55e' }, name: 'Buy: Net Worth' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.rentNetWorth[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: '#3b82f6', width: 2.5 }, itemStyle: { color: '#3b82f6' }, name: 'Rent + Invest: Net Worth' },
        ]
    }
})

// ── Crossover point ─────────────────────────────────────────────────
const crossoverYear = computed(() => {
    const p = chartProjection.value
    for (let i = 1; i < p.yearLabels.length; i++) {
        const prevDiff = p.buyNetWorth[i - 1] - p.rentNetWorth[i - 1]
        const currDiff = p.buyNetWorth[i] - p.rentNetWorth[i]
        if (prevDiff <= 0 && currDiff > 0) return p.yearLabels[i]
    }
    return null
})

const finalBuy = computed(() => chartProjection.value.buyNetWorth[chartProjection.value.buyNetWorth.length - 1])
const finalRent = computed(() => chartProjection.value.rentNetWorth[chartProjection.value.rentNetWorth.length - 1])
const finalDifference = computed(() => finalBuy.value - finalRent.value)
const buyWinsAt20 = computed(() => finalDifference.value > 0)

// ── Formatting helpers ──────────────────────────────────────────────
function formatCurrency(value: number): string {
    return new Intl.NumberFormat('de-CH', { style: 'currency', currency: 'CHF', maximumFractionDigits: 0 }).format(value)
}

function formatCurrencyShort(value: number): string {
    if (value >= 1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value >= 1_000) return (value / 1_000).toFixed(0) + 'k'
    if (value <= -1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value <= -1_000) return (value / 1_000).toFixed(0) + 'k'
    return value.toFixed(0)
}

// ── Save / Load ─────────────────────────────────────────────────────
const TOOL_TYPE = 'buy-vs-rent-simulator'

let autoSaveTimer: ReturnType<typeof setTimeout> | null = null

watch(
    [purchasePrice, cashEquity, additionalEquity, mortgages, propertyTax, insurance, maintenanceCost,
     otherCosts, incidentalPct, housingPriceIncreasePct, currentMonthlyRent, rentIncreasePct, etfReturnPct],
    () => {
        if (autoSaveTimer) clearTimeout(autoSaveTimer)
        autoSaveTimer = setTimeout(() => handleAutoSave(), 1000)
    },
    { deep: true }
)

async function handleAutoSave() {
    if (props.caseId <= 0) return
    try {
        await updateCase(TOOL_TYPE, props.caseId, {
            expectedAnnualReturn: 0,
            params: getCurrentParams(),
        })
    } catch (e) {
        console.error('Auto-save failed:', e)
    }
}

async function handleSaveAs() {
    try {
        const cs = await createCase(TOOL_TYPE, {
            name: saveName.value,
            description: saveDescription.value,
            expectedAnnualReturn: 0,
            params: getCurrentParams(),
        })
        showSaveDialog.value = false
        router.push(`/financial-simulator/${TOOL_TYPE}/${cs.id}`)
    } catch (e) {
        console.error('Save failed:', e)
    }
}

function openSaveAsDialog() {
    saveName.value = ''
    saveDescription.value = ''
    showSaveDialog.value = true
}

async function handleSave() {
    if (props.caseId <= 0) return
    try {
        await updateCase(TOOL_TYPE, props.caseId, {
            name: activeCaseName.value,
            description: activeCaseDescription.value,
            expectedAnnualReturn: 0,
            params: getCurrentParams(),
        })
        toast.add({ severity: 'success', summary: 'Saved', life: 2000 })
    } catch (e) {
        console.error('Save failed:', e)
    }
}

async function handleEditSave() {
    if (props.caseId <= 0) return
    try {
        await updateCase(TOOL_TYPE, props.caseId, {
            name: activeCaseName.value,
            description: activeCaseDescription.value,
        })
        showEditDialog.value = false
        toast.add({ severity: 'success', summary: 'Saved', life: 2000 })
    } catch (e) {
        console.error('Edit save failed:', e)
    }
}

onMounted(async () => {
    if (props.caseId > 0) {
        try {
            const cs = await getCase<BuyVsRentSimulatorParams>(TOOL_TYPE, props.caseId)
            const p = cs.params
            if (p) {
                purchasePrice.value = p.purchasePrice ?? purchasePrice.value
                cashEquity.value = p.cashEquity ?? cashEquity.value
                additionalEquity.value = (p.additionalEquity ?? []).map(e => ({ ...e }))
                mortgages.value = (p.mortgages ?? []).map((m: any) => ({
                    name: m.name,
                    splitPct: m.splitPct ?? 100,
                    interestRate: m.interestRate,
                    termYears: m.termYears,
                    amortize: m.amortize,
                }))
                propertyTax.value = p.propertyTax ?? propertyTax.value
                insurance.value = p.insurance ?? insurance.value
                maintenanceCost.value = p.maintenance ?? maintenanceCost.value
                otherCosts.value = p.otherCosts ?? otherCosts.value
                incidentalPct.value = p.incidentalPct ?? 1
                housingPriceIncreasePct.value = p.housingPriceIncreasePct ?? 2
                currentMonthlyRent.value = p.currentMonthlyRent ?? currentMonthlyRent.value
                rentIncreasePct.value = p.rentIncreasePct ?? 1.5
                etfReturnPct.value = p.etfReturnPct ?? 7
            }
            activeCaseName.value = cs.name
            activeCaseDescription.value = cs.description ?? ''
        } catch (e) {
            console.error('Failed to load case:', e)
        }
    }
})
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="flex align-items-center justify-content-between mb-3">
                    <div class="flex align-items-center gap-2">
                        <Button icon="pi pi-arrow-left" label="Back" text @click="router.push('/tools')" />
                        <span class="text-xl font-bold">Buy vs Rent : {{ activeCaseName }}</span>
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button label="Import" icon="pi pi-download" size="small" outlined @click="openImportDialog()" />
                        <Button label="Edit" icon="pi pi-pencil" size="small" outlined @click="showEditDialog = true" />
                        <Button label="Save" icon="pi pi-save" size="small" @click="handleSave()" />
                        <Button label="Save As" icon="pi pi-copy" size="small" outlined @click="openSaveAsDialog()" />
                    </div>
                </div>

                <div class="grid">
                    <!-- Left panel: Inputs -->
                    <div class="col-12 md:col-4">
                        <Card>
                            <template #title>Parameters</template>
                            <template #content>
                                <TabView>
                                    <!-- Tab: Property -->
                                    <TabPanel header="Property" value="property">
                                        <div class="form-grid">
                                            <div class="field">
                                                <label>Purchase Price</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="purchasePrice" :min="0" :max="10000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="purchasePrice" :min="0" :max="3000000" :step="10000" class="field-slider" />
                                                </div>
                                            </div>

                                            <div class="field">
                                                <label>Property Appreciation (%/yr)</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="housingPriceIncreasePct" :min="-10" :max="20" :step="0.1" mode="decimal" :maxFractionDigits="1" suffix="%" class="field-input" />
                                                    <Slider v-model="housingPriceIncreasePct" :min="-5" :max="10" :step="0.1" class="field-slider" />
                                                </div>
                                            </div>

                                            <Divider />
                                            <div class="section-header">Recurring Costs (yearly)</div>
                                            <div class="field">
                                                <label>Property Tax</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="propertyTax" :min="0" :max="50000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="propertyTax" :min="0" :max="10000" :step="100" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Insurance</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="insurance" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="insurance" :min="0" :max="5000" :step="100" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Maintenance</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="maintenanceCost" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="maintenanceCost" :min="0" :max="5000" :step="100" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Incidental (%)</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="incidentalPct" :min="0" :max="5" :step="0.1" :minFractionDigits="1" :maxFractionDigits="1" suffix="%" class="field-input" />
                                                    <Slider v-model="incidentalPct" :min="0" :max="5" :step="0.1" class="field-slider" />
                                                </div>
                                                <span class="text-color-secondary text-sm">= {{ formatCurrency(incidentalCost) }} / yr</span>
                                            </div>
                                            <div class="field">
                                                <label>Other Costs</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="otherCosts" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="otherCosts" :min="0" :max="5000" :step="100" class="field-slider" />
                                                </div>
                                            </div>
                                        </div>
                                    </TabPanel>

                                    <!-- Tab: Financing -->
                                    <TabPanel header="Financing" value="financing">
                                        <div class="form-grid">
                                            <div class="section-header">Personal Contribution</div>
                                            <div class="field">
                                                <label>Cash Equity</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="cashEquity" :min="0" :max="5000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="cashEquity" :min="0" :max="1000000" :step="10000" class="field-slider" />
                                                </div>
                                            </div>
                                            <div v-for="(eq, idx) in additionalEquity" :key="'eq-' + idx" class="field dynamic-item">
                                                <div class="flex justify-content-between align-items-center">
                                                    <label>Source Name</label>
                                                    <Button icon="pi pi-trash" severity="danger" text size="small" @click="removeEquitySource(idx)" />
                                                </div>
                                                <InputText v-model="eq.name" class="w-full" placeholder="e.g. 2nd Pillar" />
                                                <label>Amount</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="eq.amount" :min="0" :max="5000000" :step="1000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="eq.amount" :min="0" :max="500000" :step="1000" class="field-slider" />
                                                </div>
                                            </div>
                                            <Button label="Add Equity Source" icon="pi pi-plus" size="small" text @click="addEquitySource" />
                                            <div class="field-summary">
                                                Total Equity: <strong>{{ formatCurrency(totalEquity) }}</strong>
                                            </div>

                                            <Divider />
                                            <div class="section-header">Mortgages</div>
                                            <div v-if="mortgages.length > 1 && Math.abs(totalSplitPct - 100) > 0.5" class="financing-warning">
                                                Split total: {{ totalSplitPct.toFixed(0) }}% (should be 100%)
                                            </div>
                                            <div class="field-summary" v-if="mortgages.length > 0">
                                                Total to finance: <strong>{{ formatCurrency(totalMortgageNeeded) }}</strong>
                                            </div>
                                            <div v-for="(m, idx) in mortgages" :key="'m-' + idx" class="mortgage-block">
                                                <div class="flex justify-content-between align-items-center mb-2">
                                                    <InputText v-model="m.name" class="mortgage-name" />
                                                    <Button icon="pi pi-trash" severity="danger" text size="small" @click="removeMortgage(idx)" />
                                                </div>
                                                <div class="field" v-if="mortgages.length > 1">
                                                    <label>Principal</label>
                                                    <div class="field-controls">
                                                        <InputNumber :modelValue="Math.round(mortgagePrincipal(m))" @update:modelValue="v => updatePrincipal(idx, v ?? 0)" :min="0" :max="10000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    </div>
                                                </div>
                                                <div class="field" v-if="mortgages.length > 1">
                                                    <label>Split (%)</label>
                                                    <div class="field-controls">
                                                        <InputNumber :modelValue="m.splitPct" @update:modelValue="v => updateSplitPct(idx, v ?? 0)" :min="0" :max="100" :step="1" :maxFractionDigits="0" suffix="%" class="field-input" />
                                                        <Slider :modelValue="m.splitPct" @update:modelValue="v => updateSplitPct(idx, Array.isArray(v) ? v[0] : v)" :min="0" :max="100" :step="1" class="field-slider" />
                                                    </div>
                                                </div>
                                                <div class="field-summary" v-if="mortgages.length === 1">
                                                    Principal: <strong>{{ formatCurrency(mortgagePrincipal(m)) }}</strong>
                                                </div>
                                                <div class="field">
                                                    <label>Interest Rate (%)</label>
                                                    <div class="field-controls">
                                                        <InputNumber v-model="m.interestRate" :min="0" :max="15" :minFractionDigits="1" :maxFractionDigits="2" class="field-input" />
                                                        <Slider v-model="m.interestRate" :min="0" :max="10" :step="0.1" class="field-slider" />
                                                    </div>
                                                </div>
                                                <div class="field" v-if="m.amortize">
                                                    <label>Term (years)</label>
                                                    <div class="field-controls">
                                                        <InputNumber v-model="m.termYears" :min="1" :max="50" class="field-input" />
                                                        <Slider v-model="m.termYears" :min="1" :max="50" :step="1" class="field-slider" />
                                                    </div>
                                                </div>
                                                <div class="field flex align-items-center gap-2">
                                                    <ToggleSwitch v-model="m.amortize" />
                                                    <label>Amortize (pay down principal)</label>
                                                </div>
                                            </div>
                                            <Button label="Add Mortgage" icon="pi pi-plus" size="small" text @click="addMortgage" />
                                        </div>
                                    </TabPanel>

                                    <!-- Tab: Rent -->
                                    <TabPanel header="Rent" value="rent">
                                        <div class="form-grid">
                                            <div class="field">
                                                <label>Current Monthly Rent</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="currentMonthlyRent" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="currentMonthlyRent" :min="0" :max="10000" :step="50" class="field-slider" />
                                                </div>
                                            </div>

                                            <div class="field">
                                                <label>Annual Rent Increase (%)</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="rentIncreasePct" :min="0" :max="10" :step="0.1" :maxFractionDigits="1" suffix="%" class="field-input" />
                                                    <Slider v-model="rentIncreasePct" :min="0" :max="5" :step="0.1" class="field-slider" />
                                                </div>
                                            </div>

                                            <Divider />

                                            <div class="field">
                                                <label>ETF Expected Return (%/yr)
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('ETF Expected Return', '<p>The annual return you expect from investing in an ETF instead of buying property.</p><p>Historical average for a diversified world ETF is around <strong>7-8%</strong> nominal.</p><p>This is used to calculate the opportunity cost: what your equity + monthly savings would grow to if invested.</p>')" />
                                                </label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="etfReturnPct" :min="0" :max="20" :step="0.5" :maxFractionDigits="1" suffix="%" class="field-input" />
                                                    <Slider v-model="etfReturnPct" :min="0" :max="15" :step="0.5" class="field-slider" />
                                                </div>
                                            </div>
                                        </div>
                                    </TabPanel>
                                </TabView>
                            </template>
                        </Card>
                    </div>

                    <!-- Right panel: Chart + Results -->
                    <div class="col-12 md:col-8">
                        <Card>
                            <template #title>Buy vs Rent Comparison (20 years)</template>
                            <template #content>
                                <div class="chart-wrap">
                                    <VChart :option="chartOption" autoresize class="chart" />
                                </div>
                                <div class="verdict-banner" :class="buyWinsAt20 ? 'verdict-buy' : 'verdict-rent'">
                                    <div class="verdict-icon">
                                        <i :class="buyWinsAt20 ? 'pi pi-home' : 'pi pi-chart-line'" style="font-size: 1.5rem"></i>
                                    </div>
                                    <div class="verdict-text">
                                        <strong>{{ buyWinsAt20 ? 'Buying wins' : 'Renting + Investing wins' }}</strong> after 20 years
                                        <span class="verdict-detail">
                                            by {{ formatCurrency(Math.abs(finalDifference)) }}
                                        </span>
                                    </div>
                                    <div class="verdict-crossover" v-if="crossoverYear">
                                        Crossover at year {{ crossoverYear }}
                                    </div>
                                </div>

                                <div class="results">
                                    <h4>Monthly Costs</h4>
                                    <div class="result-row">
                                        <span class="result-label">Buying (mortgage + costs)</span>
                                        <span class="result-value">{{ formatCurrency(totalMonthlyBuyCost) }} / mo</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Renting</span>
                                        <span class="result-value">{{ formatCurrency(currentMonthlyRent) }} / mo</span>
                                    </div>
                                    <div class="result-row font-bold">
                                        <span class="result-label">
                                            Monthly Difference
                                            <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                               @click="openHelp('Monthly Difference', '<p>The difference between your monthly cost of buying vs renting.</p><p>If <strong>positive</strong>, buying costs more — the extra amount is what you could invest in an ETF if you rent instead.</p><p>If <strong>negative</strong>, buying is cheaper monthly than renting.</p>')" />
                                        </span>
                                        <span class="result-value" :style="{ color: monthlyCostDifference > 0 ? 'var(--c-red-500)' : 'var(--c-green-500)' }">
                                            {{ monthlyCostDifference > 0 ? '+' : '' }}{{ formatCurrency(monthlyCostDifference) }} / mo
                                        </span>
                                    </div>

                                    <h4>20-Year Net Worth</h4>
                                    <div class="result-row">
                                        <span class="result-label">
                                            <i class="pi pi-circle-fill" style="color: #22c55e; font-size: 0.6rem; vertical-align: middle; margin-right: 0.3rem"></i>
                                            Buy
                                        </span>
                                        <span class="result-value">{{ formatCurrency(finalBuy) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">
                                            <i class="pi pi-circle-fill" style="color: #3b82f6; font-size: 0.6rem; vertical-align: middle; margin-right: 0.3rem"></i>
                                            Rent + Invest
                                        </span>
                                        <span class="result-value">{{ formatCurrency(finalRent) }}</span>
                                    </div>
                                    <div class="result-row font-bold">
                                        <span class="result-label">Difference</span>
                                        <span class="result-value" :style="{ color: buyWinsAt20 ? 'var(--c-green-500)' : 'var(--c-red-500)' }">
                                            {{ finalDifference > 0 ? 'Buy +' : 'Rent +' }}{{ formatCurrency(Math.abs(finalDifference)) }}
                                        </span>
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

                <!-- Edit Dialog -->
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
                        <Button label="Save" @click="handleEditSave" />
                        <Button label="Cancel" text @click="showEditDialog = false" />
                    </template>
                </Dialog>

                <Dialog v-model:visible="showHelpDialog" :header="helpDialogTitle" :modal="true" :style="{ width: '40rem' }">
                    <div v-html="helpDialogContent"></div>
                </Dialog>

                <!-- Import from Real Estate Dialog -->
                <Dialog v-model:visible="showImportDialog" header="Import from Real Estate Simulation" :modal="true" :style="{ width: '30rem' }">
                    <p class="mt-0 text-color-secondary">Load property, financing and cost data from an existing real estate simulation.</p>
                    <div v-if="loadingImport" class="text-center p-3">Loading...</div>
                    <div v-else-if="realEstateCases.length === 0" class="text-color-secondary p-3">No real estate simulations found.</div>
                    <Select
                        v-else
                        v-model="selectedImportCase"
                        :options="realEstateCases"
                        optionLabel="name"
                        placeholder="Select a simulation"
                        class="w-full"
                    />
                    <template #footer>
                        <Button label="Import" icon="pi pi-download" @click="handleImportConfirm" :disabled="!selectedImportCase" />
                        <Button label="Cancel" text @click="showImportDialog = false" />
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
    gap: 1rem;
}

.section-header {
    font-weight: 700;
    font-size: 0.95rem;
    color: var(--primary-color);
    border-bottom: 1px solid var(--surface-200);
    padding-bottom: 0.25rem;
    margin-top: 0.5rem;
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

.field-summary {
    font-size: 0.9rem;
    color: var(--text-color-secondary);
    padding: 0.25rem 0;
}

.mortgage-block {
    border: 1px solid var(--surface-200);
    border-radius: 6px;
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.mortgage-name {
    font-weight: 600;
    width: 10rem;
}

.dynamic-item {
    padding-left: 0.5rem;
    border-left: 3px solid var(--surface-200);
}

.financing-warning {
    color: var(--c-orange-500);
    font-weight: 600;
    font-size: 0.9rem;
    padding: 0.5rem;
    background: var(--c-orange-50);
    border-radius: 4px;
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
}

.result-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    min-height: 1.5rem;
}

.result-label {
    color: var(--text-color-secondary);
}

.result-value {
    font-weight: 600;
}

.verdict-banner {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem 1.25rem;
    border-radius: 8px;
    margin-bottom: 1rem;
}

.verdict-buy {
    background: var(--c-green-50, #f0fdf4);
    border: 2px solid var(--c-green-500, #22c55e);
}

.verdict-rent {
    background: var(--c-blue-50, #eff6ff);
    border: 2px solid var(--c-blue-500, #3b82f6);
}

.verdict-icon {
    flex-shrink: 0;
}

.verdict-buy .verdict-icon {
    color: var(--c-green-500, #22c55e);
}

.verdict-rent .verdict-icon {
    color: var(--c-blue-500, #3b82f6);
}

.verdict-text {
    flex: 1;
    font-size: 1.05rem;
}

.verdict-detail {
    display: block;
    font-size: 0.9rem;
    opacity: 0.8;
}

.verdict-crossover {
    font-size: 0.85rem;
    opacity: 0.7;
    white-space: nowrap;
}
</style>
