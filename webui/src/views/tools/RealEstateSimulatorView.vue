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
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import ToggleSwitch from 'primevue/toggleswitch'
import Divider from 'primevue/divider'
import ProgressBar from 'primevue/progressbar'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { getCase, createCase, updateCase, uploadCaseAttachment, getCaseAttachmentUrl, deleteCaseAttachment } from '@/lib/api/ToolsData'
import type { RealEstateSimulatorParams } from '@/lib/api/ToolsData'
import {
    calcMonthlyPayment,
    calcTotalInterest,
    computeAmortizationSchedule,
    computeRealEstateProjection,
} from '@/lib/simulators/realEstate'
import FileUpload from 'primevue/fileupload'
import { useToast } from 'primevue/usetoast'

const props = defineProps<{ caseId: number }>()

const router = useRouter()

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const leftSidebarCollapsed = ref(true)

// ── Form inputs (defaults) ──────────────────────────────────────────
const purchasePrice = ref(1000000)
const marketValue = ref(1000000)
const squareMeters = ref(80)
const monthlyRent = ref(1500)
const propertyTax = ref(1000)
const insurance = ref(500)
const maintenanceCost = ref(1000)
const otherCosts = ref(0)
const incidentalPct = ref(1)
const cashEquity = ref(100000)
const additionalEquity = ref<Array<{ name: string; amount: number }>>([])
const mortgages = ref<Array<{
    name: string
    splitPct: number
    interestRate: number
    termYears: number
    amortize: boolean
}>>([
    { name: '1st Mortgage', splitPct: 100, interestRate: 1.5, termYears: 25, amortize: true }
])
const grossAnnualIncome = ref(96000)
const housingPriceIncreasePct = ref(2)

// ── Dynamic list helpers ────────────────────────────────────────────
function addEquitySource() {
    additionalEquity.value.push({ name: '', amount: 0 })
}

function removeEquitySource(index: number) {
    additionalEquity.value.splice(index, 1)
}

function addMortgage() {
    // Split evenly across all mortgages
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

const incidentalCost = computed(() => (purchasePrice.value ?? 0) * (incidentalPct.value ?? 0) / 100)

const recurringCostsWithoutIncidental = computed(() => {
    return (propertyTax.value ?? 0) + (insurance.value ?? 0) + (maintenanceCost.value ?? 0) + (otherCosts.value ?? 0)
})

const totalRecurringCosts = computed(() => {
    return recurringCostsWithoutIncidental.value + incidentalCost.value
})

const annualRent = computed(() => (monthlyRent.value ?? 0) * 12)

const totalMortgageNeeded = computed(() => Math.max(0, (purchasePrice.value ?? 0) - totalEquity.value))

function mortgagePrincipal(m: { splitPct: number }): number {
    return totalMortgageNeeded.value * (m.splitPct ?? 0) / 100
}

const totalMortgagePrincipal = computed(() => {
    return mortgages.value.reduce((sum, m) => sum + mortgagePrincipal(m), 0)
})

const totalSplitPct = computed(() => mortgages.value.reduce((sum, m) => sum + (m.splitPct ?? 0), 0))

const financingGap = computed(() => {
    return (purchasePrice.value ?? 0) - totalEquity.value - totalMortgagePrincipal.value
})

// ── Mortgage payment calculation (delegated to shared utility) ───────

const mortgageDetails = computed(() => {
    return mortgages.value.map((m) => {
        const principal = mortgagePrincipal(m)
        const monthly = calcMonthlyPayment(principal, m.interestRate, m.termYears, m.amortize)
        const totalInterest = calcTotalInterest(principal, m.interestRate, m.termYears, m.amortize)
        return {
            ...m,
            principal,
            monthlyPayment: monthly,
            annualPayment: monthly * 12,
            totalInterest,
            interestToPrincipalRatio: principal > 0 ? (totalInterest / principal) * 100 : 0
        }
    })
})

const totalAnnualMortgagePayments = computed(() => {
    return mortgageDetails.value.reduce((sum, m) => sum + m.annualPayment, 0)
})

const totalMonthlyMortgagePayments = computed(() => {
    return mortgageDetails.value.reduce((sum, m) => sum + m.monthlyPayment, 0)
})

const totalMortgageInterest = computed(() => {
    return mortgageDetails.value.reduce((sum, m) => sum + m.totalInterest, 0)
})

const overallInterestToPrincipalRatio = computed(() => {
    const principal = totalMortgagePrincipal.value
    return principal > 0 ? (totalMortgageInterest.value / principal) * 100 : 0
})

function interestRatioColor(ratio: number): string {
    if (ratio <= 20) return 'var(--c-green-500)'
    if (ratio <= 40) return 'var(--c-orange-500)'
    return 'var(--c-red-500)'
}

// ── Rentability metrics ─────────────────────────────────────────────
const noi = computed(() => annualRent.value - totalRecurringCosts.value)

const grossAnnualReturn = computed(() => {
    const mv = marketValue.value ?? 0
    return mv > 0 ? (annualRent.value / mv) * 100 : 0
})

const capRate = computed(() => {
    const mv = marketValue.value ?? 0
    return mv > 0 ? (noi.value / mv) * 100 : 0
})

const leveragedCashFlow = computed(() => noi.value - totalAnnualMortgagePayments.value)

const year1EquityBuildup = computed(() => {
    const schedule = amortizationSchedule.value
    return schedule.length > 0 ? schedule[0].totalPrincipal : 0
})

const breakevenMonthlyRent = computed(() => {
    return (totalRecurringCosts.value + totalAnnualMortgagePayments.value) / 12
})

const leveredYield = computed(() => {
    const eq = totalEquity.value
    return eq > 0 ? ((leveragedCashFlow.value + year1EquityBuildup.value) / eq) * 100 : 0
})

const avgLeveredYield = computed(() => {
    const eq = totalEquity.value
    const linearEquityBuildup = mortgageDetails.value.reduce((sum, m) => {
        return sum + (m.amortize && m.termYears > 0 ? m.principal / m.termYears : 0)
    }, 0)
    return eq > 0 ? ((leveragedCashFlow.value + linearEquityBuildup) / eq) * 100 : 0
})

const annualPropertyAppreciation = computed(() => {
    return (marketValue.value ?? 0) * (housingPriceIncreasePct.value ?? 0) / 100
})

const totalLeveredYield = computed(() => {
    const eq = totalEquity.value
    return eq > 0 ? ((leveragedCashFlow.value + year1EquityBuildup.value + annualPropertyAppreciation.value) / eq) * 100 : 0
})

const avgTotalLeveredYield = computed(() => {
    const eq = totalEquity.value
    const linearEquityBuildup = mortgageDetails.value.reduce((sum, m) => {
        return sum + (m.amortize && m.termYears > 0 ? m.principal / m.termYears : 0)
    }, 0)
    return eq > 0 ? ((leveragedCashFlow.value + linearEquityBuildup + annualPropertyAppreciation.value) / eq) * 100 : 0
})

// ── Affordability metrics ───────────────────────────────────────────
const totalMonthlyHousingCost = computed(() => {
    return totalMonthlyMortgagePayments.value + totalRecurringCosts.value / 12
})

const affordabilityRatio = computed(() => {
    const monthlyIncome = (grossAnnualIncome.value ?? 0) / 12
    return monthlyIncome > 0 ? (totalMonthlyHousingCost.value / monthlyIncome) * 100 : 0
})

const equityContributionPct = computed(() => {
    const pp = purchasePrice.value ?? 0
    return pp > 0 ? (totalEquity.value / pp) * 100 : 0
})

function affordabilityColor(ratio: number): string {
    if (ratio < 25) return 'var(--c-green-500)'
    if (ratio <= 33) return 'var(--c-orange-500)'
    return 'var(--c-red-500)'
}

function equityColor(pct: number): string {
    if (pct >= 33.3) return 'var(--c-green-500)'
    if (pct >= 20) return 'var(--c-orange-500)'
    return 'var(--c-red-500)'
}

// ── Amortization schedule (delegated to shared utility) ──────────────

const amortizationSchedule = computed(() => computeAmortizationSchedule(getCurrentParams()))

// ── Chart projection (delegated to shared utility) ──────────────────
const chartProjection = computed(() => computeRealEstateProjection(getCurrentParams()))

const chartColors = {
    netWorth: '#22c55e',
    propertyEquity: '#64748b',
    cumulativeCashFlow: '#3b82f6'
}

const chartOption = computed(() => {
    const p = chartProjection.value
    return {
        animation: true,
        legend: {
            type: 'scroll',
            bottom: 0,
            data: ['Real Estate Net Worth', 'Property Equity', 'Cumulative Cash Flow']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params: Array<{ dataIndex: number }>) => {
                const idx = params[0].dataIndex
                const y = p.yearLabels[idx]
                return [
                    `Year <strong>${y}</strong>`,
                    `Real Estate Net Worth: ${formatCurrency(p.netWorth[idx])}`,
                    `Property Equity: ${formatCurrency(p.propertyEquity[idx])}`,
                    `Cumulative Cash Flow: ${formatCurrency(p.cumulativeCashFlow[idx])}`
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
            name: 'Value',
            axisLabel: { formatter: (v: number) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: [
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.netWorth[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.netWorth, width: 2.5 }, itemStyle: { color: chartColors.netWorth }, name: 'Real Estate Net Worth' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.propertyEquity[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.propertyEquity, width: 2 }, itemStyle: { color: chartColors.propertyEquity }, name: 'Property Equity' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.cumulativeCashFlow[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.cumulativeCashFlow, width: 2 }, itemStyle: { color: chartColors.cumulativeCashFlow }, name: 'Cumulative Cash Flow' }
        ]
    }
})

const amortizationChartOption = computed(() => {
    const schedule = amortizationSchedule.value
    if (schedule.length === 0) return {}
    return {
        animation: true,
        legend: {
            bottom: 0,
            data: ['Interest Paid', 'Principal Paid', 'Ending Balance']
        },
        grid: { left: '3%', right: '4%', bottom: '15%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params: Array<{ dataIndex: number }>) => {
                const idx = params[0].dataIndex
                const yr = schedule[idx]
                return [
                    `Year <strong>${yr.year}</strong>`,
                    `Beginning Balance: ${formatCurrency(yr.totalBeginning)}`,
                    `Interest Paid: ${formatCurrency(yr.totalInterest)}`,
                    `Principal Paid: ${formatCurrency(yr.totalPrincipal)}`,
                    `Ending Balance: ${formatCurrency(yr.totalEnding)}`
                ].join('<br/>')
            }
        },
        xAxis: {
            type: 'category',
            name: 'Year',
            nameLocation: 'middle',
            nameGap: 25,
            data: schedule.map(s => s.year),
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        yAxis: {
            type: 'value',
            name: 'Value',
            axisLabel: { formatter: (v: number) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: [
            {
                type: 'bar',
                name: 'Interest Paid',
                stack: 'payment',
                data: schedule.map(s => s.totalInterest),
                itemStyle: { color: '#ef4444' }
            },
            {
                type: 'bar',
                name: 'Principal Paid',
                stack: 'payment',
                data: schedule.map(s => s.totalPrincipal),
                itemStyle: { color: '#3b82f6' }
            },
            {
                type: 'line',
                name: 'Ending Balance',
                data: schedule.map(s => s.totalEnding),
                smooth: 0.2,
                showSymbol: false,
                lineStyle: { color: '#64748b', width: 2 },
                itemStyle: { color: '#64748b' }
            }
        ]
    }
})

// ── Scenario management ─────────────────────────────────────────────
const TOOL_TYPE = 'real-estate-simulator'
const showSaveDialog = ref(false)
const showHelpDialog = ref(false)
const helpDialogTitle = ref('')
const helpDialogContent = ref('')

function openHelp(title: string, content: string) {
    helpDialogTitle.value = title
    helpDialogContent.value = content
    showHelpDialog.value = true
}
const saveName = ref('')
const saveDescription = ref('')
const activeCaseName = ref('')
const activeCaseDescription = ref('')
const activeCaseAttachmentId = ref<number | null>(null)
const toast = useToast()

function getCurrentParams(): RealEstateSimulatorParams {
    return {
        purchasePrice: purchasePrice.value,
        marketValue: marketValue.value,
        squareMeters: squareMeters.value,
        monthlyRent: monthlyRent.value,
        propertyTax: propertyTax.value,
        insurance: insurance.value,
        maintenance: maintenanceCost.value,
        otherCosts: otherCosts.value,
        incidentalPct: incidentalPct.value,
        cashEquity: cashEquity.value,
        additionalEquity: additionalEquity.value.map(e => ({ ...e })),
        mortgages: mortgages.value.map(m => ({ ...m })),
        grossAnnualIncome: grossAnnualIncome.value,
        housingPriceIncreasePct: housingPriceIncreasePct.value
    }
}

const showEditDialog = ref(false)

function openSaveAsDialog() {
    saveName.value = activeCaseName.value ? activeCaseName.value + ' (copy)' : ''
    saveDescription.value = activeCaseDescription.value
    showSaveDialog.value = true
}

async function handleSave() {
    const payload = {
        expectedAnnualReturn: totalLeveredYield.value,
        params: getCurrentParams(),
    }
    try {
        await updateCase<RealEstateSimulatorParams>(TOOL_TYPE, props.caseId, {
            ...payload,
            name: activeCaseName.value,
            description: activeCaseDescription.value,
        })
        toast.add({ severity: 'success', summary: 'Saved', detail: `"${activeCaseName.value}" updated`, life: 3000 })
    } catch (e) {
        console.error('Failed to save scenario:', e)
    }
}

async function handleSaveAs() {
    const payload = {
        expectedAnnualReturn: totalLeveredYield.value,
        params: getCurrentParams(),
    }
    try {
        const created = await createCase<RealEstateSimulatorParams>(TOOL_TYPE, {
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

function loadCaseData(cs: { name: string; description?: string; params: RealEstateSimulatorParams; attachmentId?: number }) {
    const p = cs.params
    if (p) {
        purchasePrice.value = p.purchasePrice ?? purchasePrice.value
        marketValue.value = p.marketValue ?? marketValue.value
        squareMeters.value = p.squareMeters ?? squareMeters.value
        monthlyRent.value = p.monthlyRent ?? monthlyRent.value
        propertyTax.value = p.propertyTax ?? propertyTax.value
        insurance.value = p.insurance ?? insurance.value
        maintenanceCost.value = p.maintenance ?? maintenanceCost.value
        otherCosts.value = p.otherCosts ?? otherCosts.value
        incidentalPct.value = p.incidentalPct ?? 1
        cashEquity.value = p.cashEquity ?? cashEquity.value
        additionalEquity.value = (p.additionalEquity ?? []).map(e => ({ ...e }))
        mortgages.value = (p.mortgages ?? []).map((m: any) => ({
            name: m.name,
            splitPct: m.splitPct ?? (m.principal && p.purchasePrice ? (m.principal / (p.purchasePrice - (p.cashEquity ?? 0))) * 100 : 100),
            interestRate: m.interestRate,
            termYears: m.termYears,
            amortize: m.amortize,
        }))
        grossAnnualIncome.value = p.grossAnnualIncome ?? ((p as any).grossMonthlyIncome ? (p as any).grossMonthlyIncome * 12 : 96000)
        housingPriceIncreasePct.value = p.housingPriceIncreasePct ?? 2
    }
    activeCaseName.value = cs.name
    activeCaseDescription.value = cs.description ?? ''
    activeCaseAttachmentId.value = cs.attachmentId ?? null
}

async function handleAttachmentUpload(event: { files: File | File[] }) {
    const fileList = Array.isArray(event.files) ? event.files : [event.files]
    if (!fileList.length) return
    try {
        const result = await uploadCaseAttachment(TOOL_TYPE, props.caseId, fileList[0])
        activeCaseAttachmentId.value = result.id
        toast.add({ severity: 'success', summary: 'Uploaded', detail: `"${result.originalName}" attached`, life: 3000 })
    } catch (e) {
        console.error('Failed to upload attachment:', e)
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to upload attachment', life: 3000 })
    }
}

async function handleAttachmentDelete() {
    try {
        await deleteCaseAttachment(TOOL_TYPE, props.caseId)
        activeCaseAttachmentId.value = null
        toast.add({ severity: 'success', summary: 'Removed', detail: 'Attachment removed', life: 3000 })
    } catch (e) {
        console.error('Failed to delete attachment:', e)
    }
}

function getAttachmentUrl(): string {
    return getCaseAttachmentUrl(TOOL_TYPE, props.caseId)
}

onMounted(async () => {
    try {
        const cs = await getCase<RealEstateSimulatorParams>(TOOL_TYPE, props.caseId)
        loadCaseData(cs)
    } catch (e) {
        console.error('Failed to load case:', e)
    }
})


// ── Formatters ──────────────────────────────────────────────────────
function formatCurrency(value: number): string {
    const n = Number(value)
    if (n !== n) return '0'
    return new Intl.NumberFormat('en-US', {
        style: 'decimal',
        minimumFractionDigits: 0,
        maximumFractionDigits: 0
    }).format(n)
}

function formatCurrencyShort(value: number): string {
    if (value >= 1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value >= 1_000) return (value / 1_000).toFixed(0) + 'k'
    return formatCurrency(value)
}

function formatPct(value: number): string {
    return value.toFixed(2) + '%'
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="flex align-items-center justify-content-between mb-3">
                    <div class="flex align-items-center gap-2">
                        <Button icon="pi pi-arrow-left" label="Back" text @click="router.push('/tools')" />
                        <span class="text-xl font-bold">Real Estate Simulator : {{ activeCaseName }}</span>
                    </div>
                    <div class="flex align-items-center gap-2">
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
                                                    <Slider v-model="purchasePrice" :min="0" :max="2000000" :step="10000" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Market Value</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="marketValue" :min="0" :max="10000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="marketValue" :min="0" :max="2000000" :step="10000" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Square Meters</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="squareMeters" :min="0" :max="1000" class="field-input" />
                                                    <Slider v-model="squareMeters" :min="0" :max="500" :step="5" class="field-slider" />
                                                </div>
                                            </div>

                                            <Divider />
                                            <div class="section-header">Recurring Costs (yearly)</div>
                                            <div class="field">
                                                <label>Property Tax</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="propertyTax" :min="0" :max="50000" :step="100" mode="decimal" :minFractionDigits="0" :maxFractionDigits="2" class="field-input" />
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
                                            <div class="section-header">Income</div>
                                            <div class="field">
                                                <label>Gross Annual Income</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="grossAnnualIncome" :min="0" :max="1000000" :step="1000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="grossAnnualIncome" :min="0" :max="300000" :step="1000" class="field-slider" />
                                                </div>
                                            </div>

                                            <Divider />
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
                                                        <Slider :modelValue="m.splitPct" @update:modelValue="v => updateSplitPct(idx, v)" :min="0" :max="100" :step="1" class="field-slider" />
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

                                    <!-- Tab: Returns -->
                                    <TabPanel header="Returns" value="returns">
                                        <div class="form-grid">
                                            <div class="field">
                                                <label>Monthly Rent</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="monthlyRent" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                                                    <Slider v-model="monthlyRent" :min="0" :max="10000" :step="100" class="field-slider" />
                                                </div>
                                            </div>
                                            <div class="field">
                                                <label>Housing Price Increase (%/yr)</label>
                                                <div class="field-controls">
                                                    <InputNumber v-model="housingPriceIncreasePct" :min="-10" :max="20" :step="0.1" mode="decimal" :maxFractionDigits="1" suffix="%" class="field-input" />
                                                    <Slider v-model="housingPriceIncreasePct" :min="-5" :max="10" :step="0.1" class="field-slider" />
                                                </div>
                                            </div>
                                        </div>
                                    </TabPanel>
                                </TabView>
                            </template>
                        </Card>
                    </div>

                    <!-- Right panel: Reports -->
                    <div class="col-12 md:col-8">
                        <Card>
                            <template #title>Analysis</template>
                            <template #content>
                                <TabView>
                                    <!-- Tab 1: Overview Chart -->
                                    <TabPanel header="Overview" value="overview">
                                        <div class="chart-wrap">
                                            <VChart :option="chartOption" autoresize class="chart" />
                                        </div>
                                        <div class="results">
                                            <h4>Financing</h4>
                                            <div class="result-row">
                                                <span class="result-label flex align-items-center gap-2">
                                                    Affordability Ratio
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Affordability Ratio', '<p>All monthly costs for your property should not be more than <strong>33%</strong> of your gross income.</p>')" />
                                                    <ProgressBar :value="Math.min(affordabilityRatio, 100)" :showValue="false" :pt="{ root: { style: { width: '5rem', height: '0.5rem' } }, value: { style: { background: affordabilityColor(affordabilityRatio) } } }" />
                                                </span>
                                                <span class="result-value">{{ formatPct(affordabilityRatio) }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label flex align-items-center gap-2">
                                                    Equity Contribution
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Equity Contribution', '<p>Amount of equity you contribute.</p><p><strong>Considerations:</strong></p><ul><li>As of 2014, Swiss banking guidelines prohibit the financing of mortgages without a minimum of <strong>10%</strong> of a home\'s collateral value as a down payment.</li><li>Most banks will require a <strong>20%</strong> down payment.</li><li>While you can use the 2nd pillar to finance, at least <strong>10%</strong> needs to be a direct contribution.</li></ul>')" />
                                                    <ProgressBar :value="Math.min(equityContributionPct, 100)" :showValue="false" :pt="{ root: { style: { width: '5rem', height: '0.5rem' } }, value: { style: { background: equityColor(equityContributionPct) } } }" />
                                                </span>
                                                <span class="result-value">{{ formatPct(equityContributionPct) }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Total Invested (Equity)</span>
                                                <span class="result-value">{{ formatCurrency(totalEquity) }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Total Housing Cost</span>
                                                <span class="result-value">{{ formatCurrency(totalMonthlyHousingCost * 12) }} / yr</span>
                                            </div>
                                            <h4>Property Reference</h4>
                                            <div class="result-row">
                                                <span class="result-label">Price / m²</span>
                                                <span class="result-value">{{ squareMeters > 0 ? formatCurrency(purchasePrice / squareMeters) : '—' }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Breakeven Rent
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Breakeven Rent', '<p>The minimum monthly rent needed to cover all costs (recurring costs + mortgage payments) and achieve a net cash flow of zero.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatCurrency(breakevenMonthlyRent) }} / mo</span>
                                            </div>
                                            <h4>Investment Returns</h4>
                                            <div class="result-row">
                                                <span class="result-label">Monthly Cash Flow</span>
                                                <span class="result-value">{{ formatCurrency(leveragedCashFlow / 12) }} / mo</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label font-bold">Total Levered Yield (ROI)
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Total Levered Yield (ROI)', '<p>The total year-1 return on your cash invested, including net cash flow, equity buildup, and property appreciation.</p><p><strong>Formula:</strong> (Net Cash Flow + Year-1 Equity Buildup + Annual Appreciation) / Total Equity</p><p>This is the most complete metric for comparing against alternative investments (e.g. putting the same equity into an ETF).</p>')" />
                                                </span>
                                                <span class="result-value font-bold">{{ formatPct(totalLeveredYield) }}</span>
                                            </div>
                                        </div>
                                    </TabPanel>

                                    <!-- Tab 2: Affordability -->
                                    <TabPanel header="Affordability" value="affordability">
                                        <div class="report-section">
                                            <h4>Total Housing Cost</h4>
                                            <div class="cost-table">
                                                <div class="cost-table-header">
                                                    <span></span>
                                                    <span class="cost-col-header">Month</span>
                                                    <span class="cost-col-header">Year</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Mortgage Payments</span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalMonthlyMortgagePayments) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalAnnualMortgagePayments) }}</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Recurring Costs</span>
                                                    <span class="cost-table-value">{{ formatCurrency(recurringCostsWithoutIncidental / 12) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(recurringCostsWithoutIncidental) }}</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Incidental
                                                        <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                           @click="openHelp('Incidental Expenses', '<p>In general, you can assume that maintenance and ancillary costs will amount to <strong>1%</strong> of the real estate value.</p><p>The cost includes water, electric, garbage disposal, heating and upkeep. The maintenance costs are expenses for maintaining your property, for example, for small repairs and taking care of the surrounding area and garden.</p>')" />
                                                    </span>
                                                    <span class="cost-table-value">{{ formatCurrency(incidentalCost / 12) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(incidentalCost) }}</span>
                                                </div>
                                                <div class="cost-table-row font-bold">
                                                    <span class="cost-table-label">Total</span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalMonthlyHousingCost) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalMonthlyHousingCost * 12) }}</span>
                                                </div>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <div class="result-row">
                                                <span class="result-label flex align-items-center gap-2">
                                                    Affordability Ratio
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Affordability Ratio', '<p>All monthly costs for your property should not be more than <strong>33%</strong> of your gross income.</p>')" />
                                                    <ProgressBar :value="Math.min(affordabilityRatio, 100)" :showValue="false" :pt="{ root: { style: { width: '5rem', height: '0.5rem' } }, value: { style: { background: affordabilityColor(affordabilityRatio) } } }" />
                                                </span>
                                                <span class="result-value">{{ formatPct(affordabilityRatio) }}</span>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <div class="result-row">
                                                <span class="result-label flex align-items-center gap-2">
                                                    Equity Contribution
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Equity Contribution', '<p>Amount of equity you contribute.</p><p><strong>Considerations:</strong></p><ul><li>As of 2014, Swiss banking guidelines prohibit the financing of mortgages without a minimum of <strong>10%</strong> of a home\'s collateral value as a down payment.</li><li>Most banks will require a <strong>20%</strong> down payment.</li><li>While you can use the 2nd pillar to finance, at least <strong>10%</strong> needs to be a direct contribution.</li></ul>')" />
                                                    <ProgressBar :value="Math.min(equityContributionPct, 100)" :showValue="false" :pt="{ root: { style: { width: '5rem', height: '0.5rem' } }, value: { style: { background: equityColor(equityContributionPct) } } }" />
                                                </span>
                                                <span class="result-value">{{ formatPct(equityContributionPct) }}</span>
                                            </div>
                                        </div>
                                    </TabPanel>

                                    <!-- Tab 3: Amortization -->
                                    <TabPanel header="Amortization" value="amortization">
                                        <div class="report-section" v-if="amortizationSchedule.length > 0">
                                            <div class="chart-wrap">
                                                <VChart :option="amortizationChartOption" autoresize class="chart" />
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <h4>
                                                Swiss Mortgage Rules
                                                <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer ml-2"
                                                   @click="openHelp('Swiss Mortgage Rules', '<p>In Switzerland, when a mortgage covers more than <strong>2/3</strong> of the property value, it is split into two tranches:</p><h4>First Mortgage (up to 2/3)</h4><ul><li>Covers up to 66.7% of the property value.</li><li>Can be amortized, but the borrower is <strong>under no obligation</strong> to do so — you can just pay interest.</li></ul><h4>Second Mortgage (above 2/3)</h4><ul><li>Covers the portion between your equity and the 2/3 mark.</li><li>Banks often charge <strong>higher interest rates</strong> on this tranche.</li><li>Must be <strong>fully amortized within 15 years</strong>.</li><li>Must be paid off before the borrower turns <strong>65</strong> (retirement age).</li></ul><h4>Equity Requirements</h4><ul><li>Minimum <strong>20%</strong> equity required (some banks accept 10% in specific cases).</li><li>At least <strong>10%</strong> must be a direct cash contribution (not from pension funds).</li><li>2nd pillar (pension) funds can be used for the remaining equity.</li></ul>')" />
                                            </h4>
                                        </div>

                                        <div class="report-section">
                                            <div class="cost-table">
                                                <div class="cost-table-header">
                                                    <span class="cost-table-label"><h4 style="margin: 0">Total Payments</h4></span>
                                                    <span class="cost-col-header">Month</span>
                                                    <span class="cost-col-header">Year</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label"></span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalMonthlyMortgagePayments) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(totalAnnualMortgagePayments) }}</span>
                                                </div>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <h4>Mortgage Summary</h4>
                                            <div v-for="md in mortgageDetails" :key="'sum-' + md.name" class="mortgage-summary">
                                                <div class="result-row font-bold">
                                                    <span>{{ md.name }}</span>
                                                </div>
                                                <div class="result-row">
                                                    <span class="result-label">Principal</span>
                                                    <span class="result-value">{{ formatCurrency(md.principal) }}</span>
                                                </div>
                                                <div class="result-row">
                                                    <span class="result-label">{{ md.amortize ? 'Total Interest' : 'Annual Interest' }}</span>
                                                    <span class="result-value">{{ formatCurrency(md.totalInterest) }}</span>
                                                </div>
                                                <div class="result-row" v-if="md.amortize">
                                                    <span class="result-label flex align-items-center gap-2">
                                                        Interest / Principal
                                                        <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                           @click="openHelp('Interest / Principal', '<p>Shows how much total interest you pay relative to the borrowed principal.</p><p>Lower is better — e.g. <strong>30%</strong> means you pay 30 cents of interest for every euro borrowed.</p>')" />
                                                        <ProgressBar :value="Math.min(md.interestToPrincipalRatio, 100)" :showValue="false" :pt="{ root: { style: { width: '5rem', height: '0.5rem' } }, value: { style: { background: interestRatioColor(md.interestToPrincipalRatio) } } }" />
                                                    </span>
                                                    <span class="result-value">{{ md.interestToPrincipalRatio.toFixed(1) }}%</span>
                                                </div>
                                                <div class="cost-table">
                                                    <div class="cost-table-header">
                                                        <span></span>
                                                        <span class="cost-col-header">Month</span>
                                                        <span class="cost-col-header">Year</span>
                                                    </div>
                                                    <div class="cost-table-row">
                                                        <span class="cost-table-label">Payment</span>
                                                        <span class="cost-table-value">{{ formatCurrency(md.monthlyPayment) }}</span>
                                                        <span class="cost-table-value">{{ formatCurrency(md.annualPayment) }}</span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>

                                    </TabPanel>

                                    <!-- Tab 4: Rentability -->
                                    <TabPanel header="Rentability" value="rentability">
                                        <div class="report-section">
                                            <h4>Income vs Expenses</h4>
                                            <div class="cost-table">
                                                <div class="cost-table-header">
                                                    <span></span>
                                                    <span class="cost-col-header">Month</span>
                                                    <span class="cost-col-header">Year</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Rent Income</span>
                                                    <span class="cost-table-value">{{ formatCurrency(monthlyRent) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(annualRent) }}</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Recurring Costs
                                                        <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                           @click="openHelp('Recurring Costs', '<p>Includes property tax, insurance, maintenance, incidental (' + incidentalPct.toFixed(1) + '% = ' + formatCurrency(incidentalCost) + '/yr), and other costs.</p><p><strong>Incidental</strong> is a yearly reserve for bigger expenses (e.g. repairs). A common rule of thumb is <strong>1%</strong> of the property value, though this should be evaluated more thoroughly.</p>')" />
                                                    </span>
                                                    <span class="cost-table-value">−{{ formatCurrency(totalRecurringCosts / 12) }}</span>
                                                    <span class="cost-table-value">−{{ formatCurrency(totalRecurringCosts) }}</span>
                                                </div>
                                                <div class="cost-table-row">
                                                    <span class="cost-table-label">Mortgage Payments</span>
                                                    <span class="cost-table-value">−{{ formatCurrency(totalMonthlyMortgagePayments) }}</span>
                                                    <span class="cost-table-value">−{{ formatCurrency(totalAnnualMortgagePayments) }}</span>
                                                </div>
                                                <div class="cost-table-row font-bold">
                                                    <span class="cost-table-label">Net Cash Flow</span>
                                                    <span class="cost-table-value">{{ formatCurrency(leveragedCashFlow / 12) }}</span>
                                                    <span class="cost-table-value">{{ formatCurrency(leveragedCashFlow) }}</span>
                                                </div>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <h4>Property Metrics (unlevered)</h4>
                                            <div class="result-row">
                                                <span class="result-label">Gross Annual Return
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Gross Annual Return', '<p>Annual rent income as a percentage of the purchase price, before any expenses.</p><p><strong>Formula:</strong> Annual Rent / Purchase Price</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(grossAnnualReturn) }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Net Operating Income (NOI)
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Net Operating Income (NOI)', '<p>Rental income minus all recurring operating costs, but before mortgage payments.</p><p><strong>Formula:</strong> Annual Rent − Recurring Costs</p><p>NOI is useful for comparing properties regardless of how they are financed.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatCurrency(noi) }} / yr</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Cap Rate
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Cap Rate', '<p>The capitalization rate measures the property\'s return independent of financing.</p><p><strong>Formula:</strong> NOI / Market Value</p><p>A higher cap rate indicates a potentially better investment, but may also reflect higher risk.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(capRate) }}</span>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <h4>Levered Yield (cash flow + equity buildup)</h4>
                                            <div class="result-row">
                                                <span class="result-label">Year-1
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Levered Yield (Year-1)', '<p>The return on your actual cash invested (equity), including net cash flow and equity buildup from mortgage principal repayment.</p><p><strong>Formula:</strong> (Net Cash Flow + Annual Equity Buildup) / Total Equity</p><p><strong>Note:</strong> Equity buildup uses the actual year-1 principal repayment from the amortization schedule. In early years, most of the mortgage payment goes to interest, so this value will be lower than a simple average over the loan term.</p><p>This metric is useful for comparing against alternative investments (e.g. putting the same equity into an ETF).</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(leveredYield) }}</span>
                                            </div>
                                            <div class="result-row">
                                                <span class="result-label">Average
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Levered Yield (Average)', '<p>Same as year-1 Levered Yield but using a linear average of equity buildup over the full mortgage term, instead of the year-1 value.</p><p><strong>Formula:</strong> (Net Cash Flow + Principal / Term Years) / Total Equity</p><p>This gives a sense of the average annual return over the life of the mortgage. Compare with the year-1 value to see how returns improve as more principal is repaid each year.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(avgLeveredYield) }}</span>
                                            </div>
                                        </div>

                                        <div class="report-section">
                                            <h4>Total Levered Yield (+ appreciation)</h4>
                                            <div class="result-row font-bold">
                                                <span class="result-label">Year-1
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Total Levered Yield (Year-1)', '<p>Levered Yield plus property appreciation, using year-1 equity buildup.</p><p><strong>Formula:</strong> (Net Cash Flow + Year-1 Equity Buildup + Annual Appreciation) / Total Equity</p><p>Annual appreciation is calculated as Market Value × Housing Price Increase %. This is the most complete year-1 return metric for comparing against an ETF.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(totalLeveredYield) }}</span>
                                            </div>
                                            <div class="result-row font-bold">
                                                <span class="result-label">Average
                                                    <i class="pi pi-question-circle text-color-secondary text-sm cursor-pointer"
                                                       @click="openHelp('Total Levered Yield (Average)', '<p>Same as year-1 Total Levered Yield but using a linear average of equity buildup over the full mortgage term.</p><p><strong>Formula:</strong> (Net Cash Flow + Principal / Term Years + Annual Appreciation) / Total Equity</p><p>This represents the average total annual return over the life of the mortgage, including property appreciation.</p>')" />
                                                </span>
                                                <span class="result-value">{{ formatPct(avgTotalLeveredYield) }}</span>
                                            </div>
                                        </div>
                                    </TabPanel>
                                </TabView>
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
                        <Button label="Cancel" text @click="showSaveDialog = false" />
                        <Button label="Save" @click="handleSaveAs" :disabled="!saveName" />
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
                        <div class="field">
                            <label>Attachment</label>
                            <div v-if="activeCaseAttachmentId" class="flex align-items-center gap-2">
                                <a :href="getAttachmentUrl()" target="_blank" class="flex align-items-center gap-1">
                                    <i class="pi pi-file" /> View attachment
                                </a>
                                <Button icon="pi pi-trash" size="small" text severity="danger" @click="handleAttachmentDelete" title="Remove attachment" />
                            </div>
                            <FileUpload
                                v-else
                                mode="basic"
                                :auto="true"
                                :maxFileSize="10000000"
                                accept="image/*,application/pdf"
                                chooseLabel="Upload file"
                                :customUpload="true"
                                @uploader="handleAttachmentUpload"
                            />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Close" text @click="showEditDialog = false" />
                    </template>
                </Dialog>

                <Dialog v-model:visible="showHelpDialog" :header="helpDialogTitle" :modal="true" :style="{ width: '40rem' }">
                    <div v-html="helpDialogContent"></div>
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

.result-label-indent {
    padding-left: 1rem;
}

.result-value {
    font-weight: 600;
}

.report-section {
    margin-bottom: 1.5rem;
}

.report-section h4 {
    margin: 0 0 0.75rem 0;
    font-size: 1rem;
}

.mortgage-summary {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border: 1px solid var(--surface-200);
    border-radius: 6px;
}

.cost-table {
    display: flex;
    flex-direction: column;
}

.cost-table-header,
.cost-table-row {
    display: grid;
    grid-template-columns: 1fr 7rem 7rem;
    align-items: center;
    min-height: 1.75rem;
}

.cost-col-header {
    font-weight: 700;
    text-align: right;
}

.cost-table-label {
    color: var(--text-color-secondary);
}

.cost-table-indent {
    padding-left: 1rem;
}

.cost-table-value {
    text-align: right;
    font-weight: 600;
}
</style>
