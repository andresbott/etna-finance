# Real Estate Simulator Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a real estate investment simulator as a new tool view in etna-finance, replicating invest-calc functionality with generic (non-Swiss-specific) design.

**Architecture:** Single Vue component (`RealEstateSimulatorView.vue`) with all calculation logic inline as computed properties. Left panel with grouped inputs (property, rent, costs, equity, mortgages, affordability). Right panel with tabbed reports (Overview Chart, Affordability, Rentability, Amortization). Uses existing ToolsData API for case study persistence.

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), TypeScript, PrimeVue 4 (Card, InputNumber, Slider, TabView, TabPanel, DataTable, ToggleSwitch, Button, Dialog), ECharts (LineChart via vue-echarts), existing ToolsData API client.

---

## File Structure

- **Modify:** `webui/src/views/tools/RealEstateSimulatorView.vue` — replace stub with full implementation
- **Modify:** `webui/src/lib/api/ToolsData.ts` — add `RealEstateSimulatorParams` interface

No backend changes. No router changes (route exists at `/tools/real-estate-simulator`).

---

## Chunk 1: TypeScript Interface and Calculation Core

### Task 1: Add RealEstateSimulatorParams interface

**Files:**
- Modify: `webui/src/lib/api/ToolsData.ts:14-21`

- [ ] **Step 1: Add the interface after PortfolioSimulatorParams**

In `webui/src/lib/api/ToolsData.ts`, add after the `PortfolioSimulatorParams` interface (after line 21):

```typescript
export interface RealEstateSimulatorParams {
    purchasePrice: number
    marketValue: number
    squareMeters: number
    monthlyRent: number
    propertyTax: number
    insurance: number
    maintenance: number
    otherCosts: number
    cashEquity: number
    additionalEquity: Array<{ name: string; amount: number }>
    mortgages: Array<{
        name: string
        principal: number
        interestRate: number
        termYears: number
        amortize: boolean
    }>
    grossMonthlyIncome: number
}
```

- [ ] **Step 2: Verify the file compiles**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors related to ToolsData.ts

- [ ] **Step 3: Commit**

```bash
git add webui/src/lib/api/ToolsData.ts
git commit -m "feat(real-estate): add RealEstateSimulatorParams interface"
```

---

### Task 2: Scaffold RealEstateSimulatorView with state and calculations

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (replace entire stub)

- [ ] **Step 1: Write the script setup section with imports, state, and calculation logic**

Replace the entire `RealEstateSimulatorView.vue` with the script section below. The template and style will be added in subsequent tasks.

```vue
<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import ToggleSwitch from 'primevue/toggleswitch'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { listCases, createCase, updateCase, deleteCase } from '@/lib/api/ToolsData'
import type { RealEstateSimulatorParams } from '@/lib/api/ToolsData'
import { useToast } from 'primevue/usetoast'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const leftSidebarCollapsed = ref(true)

// ── Form inputs (defaults) ──────────────────────────────────────────
const purchasePrice = ref(500000)
const marketValue = ref(500000)
const squareMeters = ref(80)
const monthlyRent = ref(1500)
const propertyTax = ref(1000)
const insurance = ref(500)
const maintenanceCost = ref(1000)
const otherCosts = ref(0)
const cashEquity = ref(100000)
const additionalEquity = ref<Array<{ name: string; amount: number }>>([])
const mortgages = ref<Array<{
    name: string
    principal: number
    interestRate: number
    termYears: number
    amortize: boolean
}>>([
    { name: '1st Mortgage', principal: 0, interestRate: 1.5, termYears: 25, amortize: true }
])
const grossMonthlyIncome = ref(8000)
const firstMortgageManualOverride = ref(false)

// ── Dynamic list helpers ────────────────────────────────────────────
function addEquitySource() {
    additionalEquity.value.push({ name: '', amount: 0 })
}

function removeEquitySource(index: number) {
    additionalEquity.value.splice(index, 1)
}

function addMortgage() {
    mortgages.value.push({
        name: `Mortgage ${mortgages.value.length + 1}`,
        principal: 0,
        interestRate: 1.5,
        termYears: 25,
        amortize: true
    })
}

function removeMortgage(index: number) {
    mortgages.value.splice(index, 1)
}

// ── Derived values ──────────────────────────────────────────────────
const totalEquity = computed(() => {
    const additional = additionalEquity.value.reduce((sum, e) => sum + (e.amount ?? 0), 0)
    return (cashEquity.value ?? 0) + additional
})

const totalRecurringCosts = computed(() => {
    return (propertyTax.value ?? 0) + (insurance.value ?? 0) + (maintenanceCost.value ?? 0) + (otherCosts.value ?? 0)
})

const annualRent = computed(() => (monthlyRent.value ?? 0) * 12)

const totalMortgagePrincipal = computed(() => {
    return mortgages.value.reduce((sum, m) => sum + (m.principal ?? 0), 0)
})

// Auto-adjust first mortgage principal to cover the gap
const firstMortgageAutoCalc = computed(() => {
    if (mortgages.value.length === 0) return 0
    const otherPrincipals = mortgages.value.slice(1).reduce((sum, m) => sum + (m.principal ?? 0), 0)
    return Math.max(0, (purchasePrice.value ?? 0) - totalEquity.value - otherPrincipals)
})

// Warning if financing doesn't add up
const financingGap = computed(() => {
    return (purchasePrice.value ?? 0) - totalEquity.value - totalMortgagePrincipal.value
})

// ── Mortgage payment calculation ────────────────────────────────────
function calcMonthlyPayment(principal: number, annualRate: number, termYears: number, amortize: boolean): number {
    if (principal <= 0 || annualRate < 0 || termYears <= 0) return 0
    if (!amortize) {
        return principal * (annualRate / 100) / 12
    }
    const monthlyRate = annualRate / 100 / 12
    if (monthlyRate === 0) {
        return principal / (termYears * 12)
    }
    const months = termYears * 12
    const factor = Math.pow(1 + monthlyRate, months)
    return principal * (monthlyRate * factor) / (factor - 1)
}

function calcTotalInterest(principal: number, annualRate: number, termYears: number, amortize: boolean): number {
    const monthly = calcMonthlyPayment(principal, annualRate, termYears, amortize)
    if (amortize) {
        return monthly * termYears * 12 - principal
    }
    // Interest-only: all payments are interest
    return monthly * termYears * 12
}

const mortgageDetails = computed(() => {
    return mortgages.value.map((m) => {
        const monthly = calcMonthlyPayment(m.principal, m.interestRate, m.termYears, m.amortize)
        const totalInterest = calcTotalInterest(m.principal, m.interestRate, m.termYears, m.amortize)
        return {
            ...m,
            monthlyPayment: monthly,
            annualPayment: monthly * 12,
            totalInterest,
            interestToPrincipalRatio: m.principal > 0 ? (totalInterest / m.principal) * 100 : 0
        }
    })
})

const totalAnnualMortgagePayments = computed(() => {
    return mortgageDetails.value.reduce((sum, m) => sum + m.annualPayment, 0)
})

const totalMonthlyMortgagePayments = computed(() => {
    return mortgageDetails.value.reduce((sum, m) => sum + m.monthlyPayment, 0)
})

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

const leveragedCapRate = computed(() => {
    const mv = marketValue.value ?? 0
    return mv > 0 ? ((noi.value - totalAnnualMortgagePayments.value) / mv) * 100 : 0
})

const leveragedCashFlow = computed(() => noi.value - totalAnnualMortgagePayments.value)

const leveredYield = computed(() => {
    const eq = totalEquity.value
    return eq > 0 ? (leveragedCashFlow.value / eq) * 100 : 0
})

// ── Affordability metrics ───────────────────────────────────────────
const totalMonthlyHousingCost = computed(() => {
    return totalMonthlyMortgagePayments.value + totalRecurringCosts.value / 12
})

const affordabilityRatio = computed(() => {
    const income = grossMonthlyIncome.value ?? 0
    return income > 0 ? (totalMonthlyHousingCost.value / income) * 100 : 0
})

const equityContributionPct = computed(() => {
    const pp = purchasePrice.value ?? 0
    return pp > 0 ? (totalEquity.value / pp) * 100 : 0
})

function affordabilityColor(ratio: number): string {
    if (ratio < 25) return 'var(--green-500)'
    if (ratio <= 33) return 'var(--orange-500)'
    return 'var(--red-500)'
}

function equityColor(pct: number): string {
    if (pct >= 33.3) return 'var(--green-500)'
    if (pct >= 20) return 'var(--orange-500)'
    return 'var(--red-500)'
}

// ── Amortization schedule (year-by-year, per mortgage) ──────────────
const maxTerm = computed(() => {
    if (mortgages.value.length === 0) return 1
    return Math.max(...mortgages.value.map(m => m.termYears), 1)
})

const amortizationSchedule = computed(() => {
    const years: Array<{
        year: number
        mortgages: Array<{
            name: string
            beginningBalance: number
            interestPaid: number
            principalPaid: number
            endingBalance: number
        }>
        totalBeginning: number
        totalInterest: number
        totalPrincipal: number
        totalEnding: number
    }> = []

    // Track balance per mortgage
    const balances = mortgages.value.map(m => m.principal)

    for (let y = 1; y <= maxTerm.value; y++) {
        const yearData = mortgages.value.map((m, i) => {
            const bal = balances[i]
            if (bal <= 0 || y > m.termYears) {
                return {
                    name: m.name,
                    beginningBalance: Math.max(0, bal),
                    interestPaid: 0,
                    principalPaid: 0,
                    endingBalance: m.amortize ? 0 : Math.max(0, bal)
                }
            }

            const monthlyRate = m.interestRate / 100 / 12
            let yearInterest = 0
            let yearPrincipal = 0
            let currentBal = bal

            const monthlyPayment = calcMonthlyPayment(m.principal, m.interestRate, m.termYears, m.amortize)

            for (let month = 0; month < 12; month++) {
                if (currentBal <= 0) break
                const interestThisMonth = currentBal * monthlyRate
                let principalThisMonth: number
                if (m.amortize) {
                    principalThisMonth = Math.min(monthlyPayment - interestThisMonth, currentBal)
                } else {
                    principalThisMonth = 0
                }
                yearInterest += interestThisMonth
                yearPrincipal += principalThisMonth
                currentBal -= principalThisMonth
            }

            balances[i] = currentBal

            return {
                name: m.name,
                beginningBalance: bal,
                interestPaid: yearInterest,
                principalPaid: yearPrincipal,
                endingBalance: currentBal
            }
        })

        years.push({
            year: y,
            mortgages: yearData,
            totalBeginning: yearData.reduce((s, m) => s + m.beginningBalance, 0),
            totalInterest: yearData.reduce((s, m) => s + m.interestPaid, 0),
            totalPrincipal: yearData.reduce((s, m) => s + m.principalPaid, 0),
            totalEnding: yearData.reduce((s, m) => s + m.endingBalance, 0)
        })
    }

    return years
})

// ── Chart projection ────────────────────────────────────────────────
const chartProjection = computed(() => {
    const schedule = amortizationSchedule.value
    const yearLabels = [0, ...schedule.map(s => s.year)]

    const initialMortgageBalance = totalMortgagePrincipal.value
    const remainingMortgage = [initialMortgageBalance, ...schedule.map(s => s.totalEnding)]
    const mv = marketValue.value ?? 0
    const propertyEquity = yearLabels.map((_, i) => mv - remainingMortgage[i])

    let cumulativeInterest = 0
    const cumulativeInterestSeries = [0]
    for (const yr of schedule) {
        cumulativeInterest += yr.totalInterest
        cumulativeInterestSeries.push(cumulativeInterest)
    }

    let cumCashFlow = 0
    const cumulativeCashFlow = [0]
    for (const yr of schedule) {
        const yearMortgagePayments = yr.totalInterest + yr.totalPrincipal
        cumCashFlow += annualRent.value - totalRecurringCosts.value - yearMortgagePayments
        cumulativeCashFlow.push(cumCashFlow)
    }

    return {
        yearLabels,
        propertyEquity,
        remainingMortgage,
        cumulativeInterest: cumulativeInterestSeries,
        cumulativeCashFlow
    }
})

const chartColors = {
    propertyEquity: '#22c55e',
    remainingMortgage: '#64748b',
    cumulativeInterest: '#ef4444',
    cumulativeCashFlow: '#3b82f6'
}

const chartOption = computed(() => {
    const p = chartProjection.value
    return {
        animation: true,
        legend: {
            type: 'scroll',
            bottom: 0,
            data: ['Property Equity', 'Remaining Mortgage', 'Cumulative Interest', 'Cumulative Cash Flow']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params: Array<{ dataIndex: number }>) => {
                const idx = params[0].dataIndex
                const y = p.yearLabels[idx]
                return [
                    `Year <strong>${y}</strong>`,
                    `Property Equity: ${formatCurrency(p.propertyEquity[idx])}`,
                    `Remaining Mortgage: ${formatCurrency(p.remainingMortgage[idx])}`,
                    `Cumulative Interest: ${formatCurrency(p.cumulativeInterest[idx])}`,
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
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.propertyEquity[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.propertyEquity, width: 2.5 }, itemStyle: { color: chartColors.propertyEquity }, name: 'Property Equity' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.remainingMortgage[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.remainingMortgage, width: 2 }, itemStyle: { color: chartColors.remainingMortgage }, name: 'Remaining Mortgage' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.cumulativeInterest[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.cumulativeInterest, width: 2 }, itemStyle: { color: chartColors.cumulativeInterest }, name: 'Cumulative Interest' },
            { type: 'line', data: p.yearLabels.map((y, i) => [y, p.cumulativeCashFlow[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.cumulativeCashFlow, width: 2 }, itemStyle: { color: chartColors.cumulativeCashFlow }, name: 'Cumulative Cash Flow' }
        ]
    }
})

// ── Case study management ───────────────────────────────────────────
const TOOL_TYPE = 'real-estate-simulator'
const cases = ref<Array<{ id: number; name: string; description: string; expectedAnnualReturn: number; params: RealEstateSimulatorParams }>>([])
const showSaveDialog = ref(false)
const showCasesDialog = ref(false)
const saveName = ref('')
const saveDescription = ref('')
const activeCaseId = ref<number | null>(null)
const activeCaseName = ref('')
const toast = useToast()

async function loadCases() {
    try {
        cases.value = await listCases<RealEstateSimulatorParams>(TOOL_TYPE)
    } catch (e) {
        console.error('Failed to load case studies:', e)
    }
}

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
        cashEquity: cashEquity.value,
        additionalEquity: additionalEquity.value.map(e => ({ ...e })),
        mortgages: mortgages.value.map(m => ({ ...m })),
        grossMonthlyIncome: grossMonthlyIncome.value
    }
}

function openSaveDialog() {
    saveName.value = ''
    saveDescription.value = ''
    showSaveDialog.value = true
}

async function handleSave() {
    const payload = {
        expectedAnnualReturn: leveredYield.value,
        params: getCurrentParams(),
    }
    try {
        if (activeCaseId.value) {
            await updateCase<RealEstateSimulatorParams>(TOOL_TYPE, activeCaseId.value, {
                ...payload,
                name: activeCaseName.value,
            })
            toast.add({ severity: 'success', summary: 'Saved', detail: `"${activeCaseName.value}" updated`, life: 3000 })
        } else {
            const created = await createCase<RealEstateSimulatorParams>(TOOL_TYPE, {
                ...payload,
                name: saveName.value,
                description: saveDescription.value,
            })
            activeCaseId.value = created.id
            activeCaseName.value = created.name
            showSaveDialog.value = false
            toast.add({ severity: 'success', summary: 'Created', detail: `"${created.name}" saved`, life: 3000 })
        }
        await loadCases()
    } catch (e) {
        console.error('Failed to save case study:', e)
    }
}

function loadCase(cs: { id: number; name: string; params: RealEstateSimulatorParams }) {
    const p = cs.params
    purchasePrice.value = p.purchasePrice
    marketValue.value = p.marketValue
    squareMeters.value = p.squareMeters
    monthlyRent.value = p.monthlyRent
    propertyTax.value = p.propertyTax
    insurance.value = p.insurance
    maintenanceCost.value = p.maintenance
    otherCosts.value = p.otherCosts
    cashEquity.value = p.cashEquity
    additionalEquity.value = (p.additionalEquity ?? []).map(e => ({ ...e }))
    mortgages.value = (p.mortgages ?? []).map(m => ({ ...m }))
    grossMonthlyIncome.value = p.grossMonthlyIncome
    firstMortgageManualOverride.value = true // loaded case has explicit principals
    activeCaseId.value = cs.id
    activeCaseName.value = cs.name
}

function clearActiveCase() {
    activeCaseId.value = null
    activeCaseName.value = ''
}

async function removeCaseStudy(id: number) {
    try {
        await deleteCase(TOOL_TYPE, id)
        if (activeCaseId.value === id) {
            clearActiveCase()
        }
        await loadCases()
    } catch (e) {
        console.error('Failed to delete case study:', e)
    }
}

loadCases()

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
                <p>Template placeholder — will be built in Tasks 3-6.</p>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
</style>
```

- [ ] **Step 2: Verify no TypeScript errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -30`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): scaffold view with state and calculation logic"
```

---

## Chunk 2: Template — Header, Input Panel, and Dialogs

### Task 3: Build the header and case study dialogs

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (template section)

- [ ] **Step 1: Replace the template placeholder with the header and grid layout skeleton**

Replace the `<template>` section with:

```html
<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Header -->
                <div class="flex align-items-center justify-content-between mb-3">
                    <div class="flex align-items-center gap-2">
                        <span class="text-xl font-semibold">Real Estate Simulator</span>
                        <span v-if="activeCaseName" class="text-color-secondary">— {{ activeCaseName }}</span>
                        <Button v-if="activeCaseId" icon="pi pi-times" size="small" text rounded severity="secondary" @click="clearActiveCase()" title="Detach from case study" />
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button label="Case Studies" icon="pi pi-list" size="small" outlined @click="showCasesDialog = true" />
                        <Button label="Save" icon="pi pi-save" size="small" @click="activeCaseId ? handleSave() : openSaveDialog()" />
                    </div>
                </div>

                <div class="grid">
                    <!-- Left panel: Inputs -->
                    <div class="col-12 md:col-4">
                        <!-- Task 4 fills this -->
                    </div>

                    <!-- Right panel: Reports -->
                    <div class="col-12 md:col-8">
                        <!-- Task 5+6 fill this -->
                    </div>
                </div>

                <!-- Case Studies Dialog -->
                <Dialog v-model:visible="showCasesDialog" header="Case Studies" :modal="true" :style="{ width: '50rem' }">
                    <DataTable :value="cases" size="small" v-if="cases.length > 0">
                        <Column field="name" header="Name" />
                        <Column field="description" header="Description" />
                        <Column field="expectedAnnualReturn" header="Expected Annual Return">
                            <template #body="{ data }">{{ data.expectedAnnualReturn.toFixed(2) }}%</template>
                        </Column>
                        <Column header="Actions" style="width: 7rem">
                            <template #body="{ data }">
                                <div class="flex gap-1">
                                    <Button icon="pi pi-upload" size="small" text @click="loadCase(data); showCasesDialog = false" title="Load" />
                                    <Button icon="pi pi-trash" size="small" text severity="danger" @click="removeCaseStudy(data.id)" title="Delete" />
                                </div>
                            </template>
                        </Column>
                    </DataTable>
                    <p v-else class="text-color-secondary">No saved case studies yet.</p>
                </Dialog>

                <!-- Save Dialog -->
                <Dialog v-model:visible="showSaveDialog" header="Save as New Case Study" :modal="true" :style="{ width: '30rem' }">
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
                        <Button label="Save" @click="handleSave" :disabled="!saveName" />
                    </template>
                </Dialog>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>
```

- [ ] **Step 2: Verify no errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): add header and case study dialogs"
```

---

### Task 4: Build the input panel (left side)

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (template and style)

- [ ] **Step 1: Replace the left panel placeholder**

Replace `<!-- Task 4 fills this -->` with:

```html
<Card>
    <template #title>Parameters</template>
    <template #content>
        <div class="form-grid">
            <!-- Property Section -->
            <div class="section-header">Property</div>
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

            <!-- Rental Income Section -->
            <div class="section-header">Rental Income</div>
            <div class="field">
                <label>Monthly Rent</label>
                <div class="field-controls">
                    <InputNumber v-model="monthlyRent" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                    <Slider v-model="monthlyRent" :min="0" :max="10000" :step="100" class="field-slider" />
                </div>
            </div>

            <!-- Recurring Costs Section -->
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
                <label>Other Costs</label>
                <div class="field-controls">
                    <InputNumber v-model="otherCosts" :min="0" :max="20000" :step="100" mode="decimal" :maxFractionDigits="0" class="field-input" />
                    <Slider v-model="otherCosts" :min="0" :max="5000" :step="100" class="field-slider" />
                </div>
            </div>

            <!-- Personal Contribution Section -->
            <div class="section-header">Personal Contribution</div>
            <div class="field">
                <label>Cash Equity</label>
                <div class="field-controls">
                    <InputNumber v-model="cashEquity" :min="0" :max="5000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" />
                    <Slider v-model="cashEquity" :min="0" :max="1000000" :step="10000" class="field-slider" />
                </div>
            </div>
            <div v-for="(eq, idx) in additionalEquity" :key="'eq-' + idx" class="field dynamic-item">
                <div class="flex gap-2 align-items-end">
                    <div class="flex-1">
                        <label>Source Name</label>
                        <InputText v-model="eq.name" class="w-full" placeholder="e.g. 2nd Pillar" />
                    </div>
                    <div class="flex-1">
                        <label>Amount</label>
                        <InputNumber v-model="eq.amount" :min="0" :max="5000000" :step="1000" mode="decimal" :maxFractionDigits="0" class="w-full" />
                    </div>
                    <Button icon="pi pi-trash" severity="danger" text size="small" @click="removeEquitySource(idx)" />
                </div>
            </div>
            <Button label="Add Equity Source" icon="pi pi-plus" size="small" text @click="addEquitySource" />
            <div class="field-summary">
                Total Equity: <strong>{{ formatCurrency(totalEquity) }}</strong>
            </div>

            <!-- Mortgages Section -->
            <div class="section-header">Mortgages</div>
            <div v-for="(m, idx) in mortgages" :key="'m-' + idx" class="mortgage-block">
                <div class="flex justify-content-between align-items-center mb-2">
                    <InputText v-model="m.name" class="mortgage-name" />
                    <Button icon="pi pi-trash" severity="danger" text size="small" @click="removeMortgage(idx)" />
                </div>
                <div class="field">
                    <label>Principal</label>
                    <div class="field-controls">
                        <InputNumber v-model="m.principal" :min="0" :max="10000000" :step="10000" mode="decimal" :maxFractionDigits="0" class="field-input" :placeholder="idx === 0 ? String(firstMortgageAutoCalc) : ''" @update:modelValue="if (idx === 0) firstMortgageManualOverride = true" />
                    </div>
                </div>
                <div class="field">
                    <label>Interest Rate (%)</label>
                    <div class="field-controls">
                        <InputNumber v-model="m.interestRate" :min="0" :max="15" :minFractionDigits="1" :maxFractionDigits="2" class="field-input" />
                        <Slider v-model="m.interestRate" :min="0" :max="10" :step="0.1" class="field-slider" />
                    </div>
                </div>
                <div class="field">
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
            <div v-if="Math.abs(financingGap) > 1" class="financing-warning">
                Financing gap: {{ formatCurrency(Math.abs(financingGap)) }}
                ({{ financingGap > 0 ? 'underfunded' : 'overfunded' }})
            </div>

            <!-- Affordability Section -->
            <div class="section-header">Affordability</div>
            <div class="field">
                <label>Gross Monthly Income</label>
                <div class="field-controls">
                    <InputNumber v-model="grossMonthlyIncome" :min="0" :max="100000" :step="500" mode="decimal" :maxFractionDigits="0" class="field-input" />
                    <Slider v-model="grossMonthlyIncome" :min="0" :max="30000" :step="500" class="field-slider" />
                </div>
            </div>
        </div>
    </template>
</Card>
```

- [ ] **Step 2: Add styles for the input panel**

Add the following to the `<style scoped>` section:

```css
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
    color: var(--orange-500);
    font-weight: 600;
    font-size: 0.9rem;
    padding: 0.5rem;
    background: var(--orange-50);
    border-radius: 4px;
}

:deep(.p-card-content) {
    overflow: visible;
}
```

- [ ] **Step 3: Verify no errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): add input panel with all parameter sections"
```

---

## Chunk 3: Report Tabs (Right Panel)

### Task 5: Build the Overview Chart and Affordability tabs

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (template)

- [ ] **Step 1: Replace the right panel placeholder**

Replace `<!-- Task 5+6 fill this -->` with:

```html
<Card>
    <template #title>Analysis</template>
    <template #content>
        <TabView>
            <!-- Tab 1: Overview Chart -->
            <TabPanel header="Overview">
                <div class="chart-wrap">
                    <VChart :option="chartOption" autoresize class="chart" />
                </div>
                <div class="results">
                    <div class="result-row">
                        <span class="result-label">Total Invested (Equity)</span>
                        <span class="result-value">{{ formatCurrency(totalEquity) }}</span>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Property Value</span>
                        <span class="result-value font-bold">{{ formatCurrency(marketValue) }}</span>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Levered Yield (ROI)</span>
                        <span class="result-value font-bold">{{ formatPct(leveredYield) }}</span>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Monthly Cash Flow</span>
                        <span class="result-value" :style="{ color: leveragedCashFlow >= 0 ? 'var(--green-500)' : 'var(--red-500)' }">
                            {{ formatCurrency(leveragedCashFlow / 12) }} / mo
                        </span>
                    </div>
                </div>
            </TabPanel>

            <!-- Tab 2: Affordability -->
            <TabPanel header="Affordability">
                <div class="report-section">
                    <h4>Monthly Mortgage Costs</h4>
                    <div v-for="md in mortgageDetails" :key="md.name" class="result-row">
                        <span class="result-label">{{ md.name }}</span>
                        <span class="result-value">{{ formatCurrency(md.monthlyPayment) }} / mo</span>
                    </div>
                    <div class="result-row font-bold">
                        <span class="result-label">Total Mortgage Payments</span>
                        <span class="result-value">{{ formatCurrency(totalMonthlyMortgagePayments) }} / mo</span>
                    </div>
                </div>

                <div class="report-section">
                    <h4>Total Monthly Housing Cost</h4>
                    <div class="result-row">
                        <span class="result-label">Mortgage Payments</span>
                        <span class="result-value">{{ formatCurrency(totalMonthlyMortgagePayments) }}</span>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Recurring Costs</span>
                        <span class="result-value">{{ formatCurrency(totalRecurringCosts / 12) }}</span>
                    </div>
                    <div class="result-row font-bold">
                        <span class="result-label">Total</span>
                        <span class="result-value">{{ formatCurrency(totalMonthlyHousingCost) }} / mo</span>
                    </div>
                </div>

                <div class="report-section">
                    <h4>Affordability Ratio</h4>
                    <div class="metric-bar">
                        <div class="metric-bar-fill" :style="{ width: Math.min(affordabilityRatio, 100) + '%', backgroundColor: affordabilityColor(affordabilityRatio) }"></div>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Housing Cost / Income</span>
                        <span class="result-value font-bold" :style="{ color: affordabilityColor(affordabilityRatio) }">{{ formatPct(affordabilityRatio) }}</span>
                    </div>
                    <p class="text-color-secondary text-sm">Green &lt; 25% · Orange 25-33% · Red &gt; 33%</p>
                </div>

                <div class="report-section">
                    <h4>Equity Contribution</h4>
                    <div class="metric-bar">
                        <div class="metric-bar-fill" :style="{ width: Math.min(equityContributionPct, 100) + '%', backgroundColor: equityColor(equityContributionPct) }"></div>
                    </div>
                    <div class="result-row">
                        <span class="result-label">Equity / Purchase Price</span>
                        <span class="result-value font-bold" :style="{ color: equityColor(equityContributionPct) }">{{ formatPct(equityContributionPct) }}</span>
                    </div>
                    <p class="text-color-secondary text-sm">Red &lt; 20% · Orange 20-33% · Green &gt; 33%</p>
                </div>
            </TabPanel>

            <!-- Task 6 adds Rentability and Amortization tabs here -->
        </TabView>
    </template>
</Card>
```

- [ ] **Step 2: Add report styles**

Append to the `<style scoped>` section:

```css
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

.report-section {
    margin-bottom: 1.5rem;
}

.report-section h4 {
    margin: 0 0 0.75rem 0;
    font-size: 1rem;
}

.metric-bar {
    height: 8px;
    background: var(--surface-200);
    border-radius: 4px;
    margin-bottom: 0.5rem;
    overflow: hidden;
}

.metric-bar-fill {
    height: 100%;
    border-radius: 4px;
    transition: width 0.3s, background-color 0.3s;
}
```

- [ ] **Step 3: Verify no errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): add overview chart and affordability tabs"
```

---

### Task 6: Build the Rentability and Amortization tabs

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (template)

- [ ] **Step 1: Add the remaining two tabs**

Replace `<!-- Task 6 adds Rentability and Amortization tabs here -->` with:

```html
<!-- Tab 3: Rentability -->
<TabPanel header="Rentability">
    <div class="report-section">
        <h4>Income vs Expenses</h4>
        <div class="result-row">
            <span class="result-label">Annual Rent Income</span>
            <span class="result-value">{{ formatCurrency(annualRent) }} / yr ({{ formatCurrency(monthlyRent) }} / mo)</span>
        </div>
        <div class="result-row">
            <span class="result-label">Recurring Costs</span>
            <span class="result-value">−{{ formatCurrency(totalRecurringCosts) }} / yr ({{ formatCurrency(totalRecurringCosts / 12) }} / mo)</span>
        </div>
        <div class="result-row">
            <span class="result-label">Mortgage Payments</span>
            <span class="result-value">−{{ formatCurrency(totalAnnualMortgagePayments) }} / yr ({{ formatCurrency(totalMonthlyMortgagePayments) }} / mo)</span>
        </div>
        <div class="result-row font-bold" :style="{ color: leveragedCashFlow >= 0 ? 'var(--green-500)' : 'var(--red-500)' }">
            <span class="result-label">Net Cash Flow</span>
            <span class="result-value">{{ formatCurrency(leveragedCashFlow) }} / yr ({{ formatCurrency(leveragedCashFlow / 12) }} / mo)</span>
        </div>
    </div>

    <div class="report-section">
        <h4>Investment Metrics</h4>
        <div class="result-row">
            <span class="result-label">Gross Annual Return</span>
            <span class="result-value">{{ formatPct(grossAnnualReturn) }}</span>
        </div>
        <div class="result-row">
            <span class="result-label">Net Operating Income (NOI)</span>
            <span class="result-value">{{ formatCurrency(noi) }} / yr</span>
        </div>
        <div class="result-row">
            <span class="result-label">Cap Rate</span>
            <span class="result-value">{{ formatPct(capRate) }}</span>
        </div>
        <div class="result-row">
            <span class="result-label">Leveraged Cap Rate</span>
            <span class="result-value">{{ formatPct(leveragedCapRate) }}</span>
        </div>
        <div class="result-row">
            <span class="result-label">Leveraged Cash Flow</span>
            <span class="result-value" :style="{ color: leveragedCashFlow >= 0 ? 'var(--green-500)' : 'var(--red-500)' }">{{ formatCurrency(leveragedCashFlow) }} / yr</span>
        </div>
        <div class="result-row font-bold">
            <span class="result-label">Levered Yield (ROI)</span>
            <span class="result-value">{{ formatPct(leveredYield) }}</span>
        </div>
    </div>
</TabPanel>

<!-- Tab 4: Amortization -->
<TabPanel header="Amortization">
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
                <span class="result-label">Rate</span>
                <span class="result-value">{{ md.interestRate.toFixed(2) }}%</span>
            </div>
            <div class="result-row">
                <span class="result-label">Term</span>
                <span class="result-value">{{ md.termYears }} years</span>
            </div>
            <div class="result-row">
                <span class="result-label">Monthly Payment</span>
                <span class="result-value">{{ formatCurrency(md.monthlyPayment) }}</span>
            </div>
            <div class="result-row">
                <span class="result-label">Total Interest</span>
                <span class="result-value">{{ formatCurrency(md.totalInterest) }}</span>
            </div>
            <div class="result-row">
                <span class="result-label">Interest / Principal</span>
                <span class="result-value">{{ md.interestToPrincipalRatio.toFixed(1) }}%</span>
            </div>
        </div>
    </div>

    <div class="report-section" v-if="amortizationSchedule.length > 0">
        <h4>Yearly Schedule</h4>
        <DataTable :value="amortizationSchedule" size="small" scrollable scrollHeight="400px">
            <Column field="year" header="Year" style="width: 4rem" />
            <Column header="Beginning Balance">
                <template #body="{ data }">{{ formatCurrency(data.totalBeginning) }}</template>
            </Column>
            <Column header="Interest Paid">
                <template #body="{ data }">{{ formatCurrency(data.totalInterest) }}</template>
            </Column>
            <Column header="Principal Paid">
                <template #body="{ data }">{{ formatCurrency(data.totalPrincipal) }}</template>
            </Column>
            <Column header="Ending Balance">
                <template #body="{ data }">{{ formatCurrency(data.totalEnding) }}</template>
            </Column>
        </DataTable>
    </div>
</TabPanel>
```

- [ ] **Step 2: Add amortization styles**

Append to `<style scoped>`:

```css
.mortgage-summary {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border: 1px solid var(--surface-200);
    border-radius: 6px;
}
```

- [ ] **Step 3: Verify no errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): add rentability and amortization tabs"
```

---

## Chunk 4: Smoke Test and Polish

### Task 7: Auto-set first mortgage principal

**Files:**
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue` (script)

- [ ] **Step 1: Add a watcher to auto-set first mortgage principal when it's 0**

Add after the `loadCases()` call in the script section, using `watch`:

Add to imports (update the existing import line):
```typescript
import { ref, computed, watch } from 'vue'
```

Add the watcher:
```typescript
// Auto-set first mortgage principal to cover the gap until user manually overrides it
watch(
    [purchasePrice, cashEquity, additionalEquity, () => mortgages.value.length > 1 ? mortgages.value.slice(1).reduce((s, m) => s + (m.principal ?? 0), 0) : 0],
    () => {
        if (mortgages.value.length > 0 && !firstMortgageManualOverride.value) {
            mortgages.value[0].principal = firstMortgageAutoCalc.value
        }
    },
    { immediate: true, deep: true }
)
```

- [ ] **Step 2: Verify no errors**

Run: `cd webui && npx vue-tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat(real-estate): auto-calculate first mortgage principal"
```

---

### Task 8: Smoke test in browser

**Files:** None (manual verification)

- [ ] **Step 1: Start the dev servers**

Run (in separate terminals or background):
```bash
cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && make run-backend &
cd /home/odo/.datos/edit/programacion/bumbu/etna-finance/webui && npm run dev
```

- [ ] **Step 2: Navigate to the real estate simulator**

Open `http://localhost:5173/tools/real-estate-simulator` in a browser.

Verify:
1. Page loads without console errors
2. Left panel shows all input sections (Property, Rental Income, Recurring Costs, Personal Contribution, Mortgages, Affordability)
3. Sliders and inputs are interactive and reactive
4. First mortgage principal auto-fills based on purchase price - equity
5. "Add Equity Source" and "Add Mortgage" buttons work
6. Right panel shows 4 tabs: Overview, Affordability, Rentability, Amortization
7. Chart renders with 4 series
8. Affordability and equity bars are color-coded
9. Amortization schedule table populates
10. Save/Load case studies work (save, close dialog, reopen, load)
11. Financing gap warning appears when equity + mortgages != purchase price

- [ ] **Step 3: Fix any issues found, commit**

```bash
git add webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "fix(real-estate): polish after smoke test"
```
