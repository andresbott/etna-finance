<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import Select from 'primevue/select'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { listCases, createCase, deleteCase, getCaseAttachmentUrl } from '@/lib/api/ToolsData'
import type { CaseStudy, RealEstateSimulatorParams } from '@/lib/api/ToolsData'
import { computeRealEstateAnnualYield } from '@/lib/simulators/realEstate'
import DeleteDialog from '@/components/common/ConfirmDialog.vue'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const router = useRouter()
const leftSidebarCollapsed = ref(true)

const portfolioCases = ref<CaseStudy[]>([])
const realEstateCases = ref<CaseStudy[]>([])
const loading = ref(false)

const allCases = computed(() => {
    const merged = [...portfolioCases.value, ...realEstateCases.value]
    merged.sort((a, b) => a.id - b.id)
    return merged
})

const chartCases = computed(() => allCases.value.filter(cs => cs.toolType !== 'buy-vs-rent-simulator'))

// --- Add Dialog ---
const showAddDialog = ref(false)
const newName = ref('')
const newDescription = ref('')
const newType = ref('portfolio-simulator')
const typeOptions = [
    { label: 'Portfolio Simulator', value: 'portfolio-simulator' },
    { label: 'Real Estate Simulator', value: 'real-estate-simulator' },
    { label: 'Buy vs Rent', value: 'buy-vs-rent-simulator' },
]

function openAddDialog() {
    newName.value = ''
    newDescription.value = ''
    newType.value = 'portfolio-simulator'
    showAddDialog.value = true
}

async function handleCreate() {
    if (!newName.value) return
    try {
        const cs = await createCase(newType.value, {
            name: newName.value,
            description: newDescription.value,
            expectedAnnualReturn: 0,
            params: {},
        })
        showAddDialog.value = false
        router.push(`/financial-simulator/${cs.toolType}/${cs.id}`)
    } catch (e) {
        console.error('Failed to create simulation:', e)
    }
}

// --- Load ---
async function loadAll() {
    loading.value = true
    try {
        const [p, r, b] = await Promise.all([
            listCases('portfolio-simulator').catch((e) => { console.error('Failed to load portfolio simulations:', e); return [] }),
            listCases('real-estate-simulator').catch((e) => { console.error('Failed to load real-estate simulations:', e); return [] }),
            listCases('buy-vs-rent-simulator').catch((e) => { console.error('Failed to load buy-vs-rent simulations:', e); return [] }),
        ])
        portfolioCases.value = p
        realEstateCases.value = [...r, ...b]
    } finally {
        loading.value = false
    }
}

loadAll()

// --- Delete ---
const deleteDialogVisible = ref(false)
const entryToDelete = ref<CaseStudy | null>(null)

function openDeleteDialog(cs: CaseStudy) {
    entryToDelete.value = cs
    deleteDialogVisible.value = true
}

async function handleConfirmDelete() {
    if (!entryToDelete.value) return
    try {
        await deleteCase(entryToDelete.value.toolType, entryToDelete.value.id)
        deleteDialogVisible.value = false
        entryToDelete.value = null
        await loadAll()
    } catch (e) {
        console.error('Failed to delete simulation:', e)
    }
}

// --- Duplicate ---
const showDuplicateDialog = ref(false)
const duplicateSource = ref<CaseStudy | null>(null)
const duplicateName = ref('')
const duplicateDescription = ref('')

function handleDuplicate(cs: CaseStudy) {
    duplicateSource.value = cs
    const baseName = cs.name.replace(/\s*\(copy(?:\s+\d+)?\)$/, '')
    const existingNames = new Set(allCases.value.map((c) => c.name))

    let copyName = `${baseName} (copy)`
    if (existingNames.has(copyName)) {
        let i = 2
        while (existingNames.has(`${baseName} (copy ${i})`)) i++
        copyName = `${baseName} (copy ${i})`
    }

    duplicateName.value = copyName
    duplicateDescription.value = cs.description
    showDuplicateDialog.value = true
}

async function handleConfirmDuplicate() {
    if (!duplicateSource.value || !duplicateName.value) return
    const cs = duplicateSource.value
    try {
        await createCase(cs.toolType, {
            name: duplicateName.value,
            description: duplicateDescription.value,
            expectedAnnualReturn: cs.expectedAnnualReturn,
            params: cs.params,
        })
        showDuplicateDialog.value = false
        duplicateSource.value = null
        await loadAll()
    } catch (e) {
        console.error('Failed to duplicate simulation:', e)
    }
}

// --- Edit ---
function handleEdit(cs: CaseStudy) {
    router.push(`/financial-simulator/${cs.toolType}/${cs.id}`)
}

// --- Chart ---
const comparisonYears = ref(20)
const baseInvestment = ref(0)
const inflationRate = ref(0)
const showChartSettingsDialog = ref(false)
const settingsYears = ref(20)
const settingsBaseInvestment = ref(0)
const settingsInflation = ref(0)

const isMoneyMode = computed(() => (baseInvestment.value ?? 0) > 0)

function openChartSettings() {
    settingsYears.value = comparisonYears.value
    settingsBaseInvestment.value = baseInvestment.value
    settingsInflation.value = inflationRate.value
    showChartSettingsDialog.value = true
}

function applyChartSettings() {
    comparisonYears.value = settingsYears.value
    baseInvestment.value = settingsBaseInvestment.value ?? 0
    inflationRate.value = settingsInflation.value ?? 0
    showChartSettingsDialog.value = false
}

function computeCumulativeGrowth(cs: CaseStudy, years: number, inflation: number): number[] {
    let annualYields: number[]
    if (cs.toolType === 'real-estate-simulator') {
        annualYields = computeRealEstateAnnualYield(cs.params as unknown as RealEstateSimulatorParams, years)
    } else {
        // Portfolio: flat annual return
        annualYields = Array(years).fill(cs.expectedAnnualReturn)
    }
    // Compound into cumulative growth %, adjusted for inflation
    const result: number[] = []
    let cumulative = 1
    const inflationFactor = 1 + inflation / 100
    for (let i = 0; i < years; i++) {
        cumulative *= 1 + (annualYields[i] ?? 0) / 100
        const realValue = cumulative / Math.pow(inflationFactor, i + 1)
        result.push((realValue - 1) * 100)
    }
    return result
}

const chartOption = computed(() => {
    const cases = chartCases.value
    if (cases.length === 0) return undefined

    const years = Array.from({ length: comparisonYears.value }, (_, i) => i + 1)

    const money = isMoneyMode.value
    const base = baseInvestment.value
    const inflation = inflationRate.value ?? 0

    const seriesList = cases.map((cs) => {
        const growthData = computeCumulativeGrowth(cs, comparisonYears.value, inflation)
        const startVal = money ? base : 0
        return {
            type: 'line' as const,
            name: cs.name,
            data: [[0, startVal], ...years.map((y, i) => [y, money ? base * (1 + growthData[i] / 100) : growthData[i]])],
            smooth: 0.2,
            showSymbol: false,
            lineStyle: { width: 2 },
        }
    })

    return {
        animation: true,
        legend: {
            type: 'scroll',
            bottom: 0,
            data: cases.map((cs) => cs.name),
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params: Array<{ seriesName: string; data: [number, number]; color: string }>) => {
                if (!params || params.length === 0) return ''
                const year = params[0].data[0]
                const lines = [`<strong>Year ${year}</strong>`]
                for (const p of params) {
                    const val = p.data[1]
                    const formatted = money
                        ? val.toLocaleString(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 0 })
                        : (Math.abs(val) >= 100 ? Math.round(val) + '%' : val.toFixed(2) + '%')
                    lines.push(
                        `<span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${p.color};margin-right:4px"></span>${p.seriesName}: ${formatted}`
                    )
                }
                return lines.join('<br/>')
            },
        },
        xAxis: {
            type: 'value',
            name: 'Year',
            nameLocation: 'middle',
            nameGap: 25,
            axisLabel: { formatter: (v: number) => v + 'y' },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } },
        },
        yAxis: {
            type: 'value',
            name: money ? 'Value' : 'Cumulative Growth (%)',
            axisLabel: {
                formatter: money
                    ? (v: number) => v.toLocaleString()
                    : (v: number) => (Math.abs(v) >= 100 ? Math.round(v) + '%' : v.toFixed(1) + '%'),
            },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } },
        },
        series: seriesList,
    }
})

function typeLabel(toolType: string): string {
    if (toolType === 'portfolio-simulator') return 'Portfolio'
    if (toolType === 'buy-vs-rent-simulator') return 'Buy vs Rent'
    return 'Real Estate'
}

function openAttachment(cs: CaseStudy) {
    window.open(getCaseAttachmentUrl(cs.toolType, cs.id), '_blank')
}

function typeIcon(toolType: string): string {
    if (toolType === 'portfolio-simulator') return 'ti ti-chart-pie'
    if (toolType === 'buy-vs-rent-simulator') return 'ti ti-arrows-left-right'
    return 'ti ti-home'
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Comparison Chart -->
                <Card class="mb-3">
                    <template #title>
                        <div class="flex align-items-center justify-content-between">
                            <span>{{ comparisonYears }}-Year {{ isMoneyMode ? 'Investment Value' : 'Cumulative Growth' }} Comparison{{ inflationRate > 0 ? ` (${inflationRate}% inflation adj.)` : '' }}</span>
                            <Button icon="ti ti-settings" text rounded @click="openChartSettings" v-tooltip.bottom="'Chart settings'" />
                        </div>
                    </template>
                    <template #content>
                        <div v-if="chartCases.length > 0" class="chart-wrap">
                            <VChart :option="chartOption" autoresize class="chart" />
                        </div>
                        <p v-else class="text-color-secondary m-0">
                            No simulations yet. Create one to see the comparison chart.
                        </p>
                    </template>
                </Card>

                <!-- Simulations List -->
                <Card>
                    <template #title>
                        <div class="flex align-items-center justify-content-between">
                            <span>Simulations</span>
                            <Button label="Add Simulation" icon="ti ti-plus" size="small" @click="openAddDialog" />
                        </div>
                    </template>
                    <template #content>
                        <DataTable :value="allCases" :loading="loading" size="small" v-if="allCases.length > 0">
                            <Column field="name" header="Name" />
                            <Column header="Type">
                                <template #body="{ data }">
                                    <span class="flex align-items-center gap-2">
                                        <i :class="typeIcon(data.toolType)"></i>
                                        {{ typeLabel(data.toolType) }}
                                    </span>
                                </template>
                            </Column>
                            <Column header="ROI">
                                <template #body="{ data }">
                                    {{ data.toolType === 'buy-vs-rent-simulator' ? '-' : data.expectedAnnualReturn.toFixed(2) + '%' }}
                                </template>
                            </Column>
                            <Column header="Attachment" style="width: 6rem">
                                <template #body="{ data }">
                                    <i v-if="data.attachmentId" class="ti ti-paperclip attachment-icon" @click="openAttachment(data)" v-tooltip.bottom="'View Attachment'" />
                                </template>
                            </Column>
                            <Column header="Actions" style="width: 120px">
                                <template #body="{ data }">
                                    <div class="flex gap-2 justify-content-start">
                                        <Button icon="ti ti-pencil" text rounded class="p-1" @click="handleEdit(data)" v-tooltip.bottom="'Edit'" />
                                        <Button icon="ti ti-copy" text rounded class="p-1" @click="handleDuplicate(data)" v-tooltip.bottom="'Duplicate'" />
                                        <Button icon="ti ti-trash" text rounded severity="danger" class="p-1" @click="openDeleteDialog(data)" v-tooltip.bottom="'Delete'" />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                        <p v-else class="text-color-secondary m-0">No simulations yet.</p>
                    </template>
                </Card>

                <!-- Delete Dialog -->
                <DeleteDialog
                    v-model:visible="deleteDialogVisible"
                    :name="entryToDelete?.name"
                    message="Are you sure you want to delete this simulation?"
                    @confirm="handleConfirmDelete"
                />

                <!-- Add Dialog -->
                <Dialog v-model:visible="showAddDialog" header="New Simulation" :modal="true" :style="{ width: '30rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="simName">Name</label>
                            <InputText id="simName" v-model="newName" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="simDesc">Description</label>
                            <InputText id="simDesc" v-model="newDescription" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="simType">Type</label>
                            <Select id="simType" v-model="newType" :options="typeOptions" optionLabel="label" optionValue="value" class="w-full" />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Create" @click="handleCreate" :disabled="!newName" />
                        <Button label="Cancel" text @click="showAddDialog = false" />
                    </template>
                </Dialog>

                <!-- Duplicate Dialog -->
                <Dialog v-model:visible="showDuplicateDialog" header="Duplicate Simulation" :modal="true" :style="{ width: '30rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="dupName">Name</label>
                            <InputText id="dupName" v-model="duplicateName" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="dupDesc">Description</label>
                            <InputText id="dupDesc" v-model="duplicateDescription" class="w-full" />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Duplicate" @click="handleConfirmDuplicate" :disabled="!duplicateName" />
                        <Button label="Cancel" text @click="showDuplicateDialog = false" />
                    </template>
                </Dialog>

                <!-- Chart Settings Dialog -->
                <Dialog v-model:visible="showChartSettingsDialog" header="Chart Settings" :modal="true" :style="{ width: '26rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="settingsYears">Years: {{ settingsYears }}</label>
                            <Slider id="settingsYears" v-model="settingsYears" :min="1" :max="50" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="settingsBase">Base Investment (0 = show %)</label>
                            <InputNumber id="settingsBase" v-model="settingsBaseInvestment" :min="0" mode="decimal" :minFractionDigits="0" :maxFractionDigits="0" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="settingsInflation">Inflation % (0 = nominal)</label>
                            <div class="field-controls">
                                <InputNumber id="settingsInflation" v-model="settingsInflation" :min="0" :max="30" :minFractionDigits="0" :maxFractionDigits="1" suffix="%" class="field-input" />
                                <Slider v-model="settingsInflation" :min="0" :max="15" :step="0.5" class="field-slider" />
                            </div>
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Apply" @click="applyChartSettings" />
                        <Button label="Cancel" text @click="showChartSettingsDialog = false" />
                    </template>
                </Dialog>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.chart-wrap {
    margin-bottom: 0.5rem;
}

.chart {
    height: 380px;
    width: 100%;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
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

.attachment-icon {
    font-size: 0.85rem;
    cursor: pointer;
    opacity: 0.6;
    font-weight: bold;
}
.attachment-icon:hover {
    opacity: 1;
}
</style>
