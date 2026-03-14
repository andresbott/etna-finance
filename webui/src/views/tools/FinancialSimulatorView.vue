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
import { listCases, createCase, deleteCase } from '@/lib/api/ToolsData'
import type { CaseStudy } from '@/lib/api/ToolsData'
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
            listCases('portfolio-simulator'),
            listCases('real-estate-simulator'),
            listCases('buy-vs-rent-simulator'),
        ])
        portfolioCases.value = p
        realEstateCases.value = [...r, ...b]
    } catch (e) {
        console.error('Failed to load simulations:', e)
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
const comparisonAmount = ref(100000)
const comparisonYears = ref(20)
const showChartSettingsDialog = ref(false)
const settingsAmount = ref(100000)
const settingsYears = ref(20)

function openChartSettings() {
    settingsAmount.value = comparisonAmount.value
    settingsYears.value = comparisonYears.value
    showChartSettingsDialog.value = true
}

function applyChartSettings() {
    comparisonAmount.value = settingsAmount.value
    comparisonYears.value = settingsYears.value
    showChartSettingsDialog.value = false
}

const chartOption = computed(() => {
    const cases = chartCases.value
    if (cases.length === 0) return null

    const years = Array.from({ length: comparisonYears.value + 1 }, (_, i) => i)

    const amount = comparisonAmount.value
    const seriesList = cases.map((cs) => {
        const rate = cs.expectedAnnualReturn / 100
        return {
            type: 'line' as const,
            name: cs.name,
            data: years.map((y) => [y, amount + amount * rate * y]),
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
                    lines.push(
                        `<span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${p.color};margin-right:4px"></span>${p.seriesName}: ${formatCurrencyShort(p.data[1])}`
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
            name: 'Net Worth',
            axisLabel: { formatter: (v: number) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } },
        },
        series: seriesList,
    }
})

function formatCurrencyShort(value: number): string {
    if (value >= 1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value >= 1_000) return (value / 1_000).toFixed(0) + 'k'
    if (value <= -1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value <= -1_000) return (value / 1_000).toFixed(0) + 'k'
    return value.toFixed(0)
}

function typeLabel(toolType: string): string {
    if (toolType === 'portfolio-simulator') return 'Portfolio'
    if (toolType === 'buy-vs-rent-simulator') return 'Buy vs Rent'
    return 'Real Estate'
}

function typeIcon(toolType: string): string {
    if (toolType === 'portfolio-simulator') return 'pi pi-chart-pie'
    if (toolType === 'buy-vs-rent-simulator') return 'pi pi-arrows-h'
    return 'pi pi-home'
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
                            <span>{{ comparisonYears }}-Year Net Worth Comparison ({{ formatCurrencyShort(comparisonAmount) }})</span>
                            <Button icon="pi pi-cog" text rounded @click="openChartSettings" v-tooltip.bottom="'Chart settings'" />
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
                            <Button label="Add Simulation" icon="pi pi-plus" size="small" @click="openAddDialog" />
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
                                    <i v-if="data.attachmentId" class="pi pi-paperclip" title="Has attachment"></i>
                                </template>
                            </Column>
                            <Column header="Actions" style="width: 120px">
                                <template #body="{ data }">
                                    <div class="flex gap-2 justify-content-start">
                                        <Button icon="pi pi-pencil" text rounded class="p-1" @click="handleEdit(data)" v-tooltip.bottom="'Edit'" />
                                        <Button icon="pi pi-copy" text rounded class="p-1" @click="handleDuplicate(data)" v-tooltip.bottom="'Duplicate'" />
                                        <Button icon="pi pi-trash" text rounded severity="danger" class="p-1" @click="openDeleteDialog(data)" v-tooltip.bottom="'Delete'" />
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
                        <Button label="Cancel" text @click="showAddDialog = false" />
                        <Button label="Create" @click="handleCreate" :disabled="!newName" />
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
                        <Button label="Cancel" text @click="showDuplicateDialog = false" />
                        <Button label="Duplicate" @click="handleConfirmDuplicate" :disabled="!duplicateName" />
                    </template>
                </Dialog>

                <!-- Chart Settings Dialog -->
                <Dialog v-model:visible="showChartSettingsDialog" header="Chart Settings" :modal="true" :style="{ width: '26rem' }">
                    <div class="flex flex-column gap-3">
                        <div class="field">
                            <label for="settingsAmount">Investment Amount</label>
                            <InputNumber id="settingsAmount" v-model="settingsAmount" :min="0" :max="100000000" mode="decimal" :minFractionDigits="0" :maxFractionDigits="0" class="w-full" />
                        </div>
                        <div class="field">
                            <label for="settingsYears">Years: {{ settingsYears }}</label>
                            <Slider id="settingsYears" v-model="settingsYears" :min="1" :max="50" class="w-full" />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Cancel" text @click="showChartSettingsDialog = false" />
                        <Button label="Apply" @click="applyChartSettings" />
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
</style>
