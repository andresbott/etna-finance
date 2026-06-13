<script setup>
import { ResponsiveHorizontal } from '@/components/layout'
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import DatePicker from 'primevue/datepicker'
import VChart from 'vue-echarts'
import Button from 'primevue/button'
import SelectButton from 'primevue/selectbutton'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Dialog from 'primevue/dialog'
import { getCase, createCase, updateCase, uploadCaseAttachment, getCaseAttachmentUrl, deleteCaseAttachment } from '@/lib/api/ToolsData'
import { computeBondsProjection, computeBondsExpectedReturn } from '@/lib/simulators/bonds'
import FileInput from '@/components/common/FileInput.vue'
import { useToast } from 'primevue/usetoast'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const props = defineProps({ caseId: { type: Number, required: true } })

const router = useRouter()
const leftSidebarCollapsed = ref(true)

// --- Maturity date helpers ---
function startOfMonth(d) {
    return new Date(d.getFullYear(), d.getMonth(), 1)
}
function addMonths(d, months) {
    return new Date(d.getFullYear(), d.getMonth() + months, 1)
}
function monthsBetween(from, to) {
    return (to.getFullYear() - from.getFullYear()) * 12 + (to.getMonth() - from.getMonth())
}

const today = startOfMonth(new Date())
const minMaturity = addMonths(today, 1)

// Form inputs (defaults)
const faceValue = ref(1000)
const purchasePrice = ref(950)
const couponRatePct = ref(4)
const couponFrequency = ref(2)
const maturityDate = ref(addMonths(today, 120)) // 10 years out
const maturityYears = ref(10)                    // whole-year slider, kept in sync
const taxesPct = ref(25)

// Whole months until maturity (clamped to at least one month).
const durationMonths = computed(() => Math.max(1, monthsBetween(today, startOfMonth(maturityDate.value))))
const durationYears = computed(() => durationMonths.value / 12)

const durationText = computed(() => {
    const y = Math.floor(durationMonths.value / 12)
    const m = durationMonths.value % 12
    const parts = []
    if (y > 0) parts.push(`${y} year${y === 1 ? '' : 's'}`)
    if (m > 0) parts.push(`${m} month${m === 1 ? '' : 's'}`)
    return `Maturity in ${parts.join(', ')}`
})

// Keep the year slider in sync when the date changes; guard against feedback loop.
let syncingMaturity = false
watch(maturityDate, () => {
    syncingMaturity = true
    maturityYears.value = Math.min(50, Math.max(1, Math.round(durationYears.value)))
    syncingMaturity = false
})
watch(maturityYears, (n) => {
    if (syncingMaturity) return
    maturityDate.value = addMonths(today, n * 12)
})

const TOOL_TYPE = 'bonds-simulator'

const activeCaseName = ref('')
const activeCaseDescription = ref('')
const activeCaseAttachmentId = ref(null)
const selectedAttachmentFile = ref(null)
const toast = useToast()

function toISODate(d) {
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    return `${y}-${m}-01`
}

function getCurrentParams() {
    return {
        faceValue: faceValue.value,
        purchasePrice: purchasePrice.value,
        couponRatePct: couponRatePct.value,
        couponFrequency: couponFrequency.value,
        maturityDate: toISODate(startOfMonth(maturityDate.value)),
        taxesPct: taxesPct.value,
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

function computeExpectedAnnualReturn() {
    return computeBondsExpectedReturn(getCurrentParams(), durationYears.value)
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
        faceValue.value = p.faceValue ?? faceValue.value
        purchasePrice.value = p.purchasePrice ?? purchasePrice.value
        couponRatePct.value = p.couponRatePct ?? couponRatePct.value
        couponFrequency.value = p.couponFrequency ?? couponFrequency.value
        if (p.maturityDate) {
            const [yy, mm] = p.maturityDate.split('-').map(Number)
            maturityDate.value = new Date(yy, (mm || 1) - 1, 1)
        } else if (p.yearsToMaturity) {
            // Legacy cases stored a whole-year duration instead of a date.
            maturityDate.value = addMonths(today, Math.round(p.yearsToMaturity) * 12)
        }
        taxesPct.value = p.taxesPct ?? taxesPct.value
    }
    activeCaseName.value = cs.name
    activeCaseDescription.value = cs.description ?? ''
    activeCaseAttachmentId.value = cs.attachmentId ?? null
}

onMounted(async () => {
    try {
        const cs = await getCase(TOOL_TYPE, props.caseId)
        loadCaseData(cs)
    } catch (e) {
        console.error('Failed to load case:', e)
    }
})

watch(selectedAttachmentFile, async (file) => {
    if (!file) return
    try {
        const result = await uploadCaseAttachment(TOOL_TYPE, props.caseId, file)
        activeCaseAttachmentId.value = result.id
        toast.add({ severity: 'success', summary: 'Uploaded', detail: `"${result.originalName}" attached`, life: 3000 })
    } catch (e) {
        console.error('Failed to upload attachment:', e)
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to upload attachment', life: 3000 })
    }
    selectedAttachmentFile.value = null
})

async function handleAttachmentDelete() {
    try {
        await deleteCaseAttachment(TOOL_TYPE, props.caseId)
        activeCaseAttachmentId.value = null
        toast.add({ severity: 'success', summary: 'Removed', detail: 'Attachment removed', life: 3000 })
    } catch (e) {
        console.error('Failed to delete attachment:', e)
    }
}

function getAttachmentUrl() {
    return getCaseAttachmentUrl(TOOL_TYPE, props.caseId)
}

function viewAttachment() {
    window.open(getAttachmentUrl(), '_blank')
}

const projection = computed(() => computeBondsProjection(getCurrentParams(), durationYears.value))

function formatYear(v) {
    return (Number.isInteger(v) ? v : v.toFixed(1)) + 'y'
}

const chartColors = {
    invested: '#64748b',
    totalValue: '#22c55e',
    cumulativeCoupons: '#3b82f6',
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
            data: ['Invested', 'Total Value', 'Cumulative Coupons']
        },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '6%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                const idx = params[0].dataIndex
                const y = years[idx]
                const lines = [
                    `Year <strong>${formatYear(y)}</strong>`,
                    `Invested: ${formatCurrency(s.invested[idx])}`,
                    `Total Value: ${formatCurrency(s.totalValue[idx])}`,
                    `Cumulative Coupons: ${formatCurrency(s.cumulativeCoupons[idx])}`
                ]
                return lines.join('<br/>')
            }
        },
        xAxis: {
            type: 'value',
            name: 'Year',
            nameLocation: 'middle',
            nameGap: 25,
            axisLabel: { formatter: formatYear },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        yAxis: {
            type: 'value',
            name: 'Value',
            axisLabel: { formatter: (v) => formatCurrencyShort(v) },
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: [
            { type: 'line', data: years.map((y, i) => [y, s.invested[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.invested, width: 2 }, itemStyle: { color: chartColors.invested }, name: 'Invested' },
            { type: 'line', data: years.map((y, i) => [y, s.totalValue[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.totalValue, width: 2.5 }, itemStyle: { color: chartColors.totalValue }, name: 'Total Value' },
            { type: 'line', data: years.map((y, i) => [y, s.cumulativeCoupons[i]]), smooth: 0.2, showSymbol: false, lineStyle: { color: chartColors.cumulativeCoupons, width: 2 }, itemStyle: { color: chartColors.cumulativeCoupons }, name: 'Cumulative Coupons' }
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
                        <Button icon="ti ti-arrow-left" label="Back" text @click="router.push('/financial-simulator')" />
                        <span class="text-xl font-bold">Bonds Simulator : {{ activeCaseName }}</span>
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button label="Edit" icon="ti ti-pencil" size="small" outlined @click="showEditDialog = true" />
                        <Button label="Save" icon="ti ti-device-floppy" size="small" @click="handleSave()" />
                        <Button label="Save As" icon="ti ti-copy" size="small" outlined @click="openSaveAsDialog()" />
                    </div>
                </div>

                <div class="grid">
                    <div class="col-12 md:col-4">
                        <Card>
                            <template #title>Parameters</template>
                            <template #content>
                                <div class="form-grid">
                                    <div class="field">
                                        <label for="face">Face value</label>
                                        <div class="field-controls">
                                            <InputNumber id="face" v-model="faceValue" :min="0" :max="1000000" mode="decimal" :minFractionDigits="0" :maxFractionDigits="2" class="field-input" />
                                            <Slider v-model="faceValue" :min="0" :max="10000" :step="100" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="price">Purchase price</label>
                                        <div class="field-controls">
                                            <InputNumber id="price" v-model="purchasePrice" :min="0" :max="1000000" mode="decimal" :minFractionDigits="0" :maxFractionDigits="2" class="field-input" />
                                            <Slider v-model="purchasePrice" :min="0" :max="10000" :step="100" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="coupon">Coupon rate (%)</label>
                                        <div class="field-controls">
                                            <InputNumber id="coupon" v-model="couponRatePct" :min="0" :max="100" :minFractionDigits="1" :maxFractionDigits="2" class="field-input" />
                                            <Slider v-model="couponRatePct" :min="0" :max="20" :step="0.25" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label>Coupon frequency</label>
                                        <SelectButton
                                            v-model="couponFrequency"
                                            :options="[
                                                { label: 'Annual', value: 1 },
                                                { label: 'Semi-annual', value: 2 },
                                            ]"
                                            optionLabel="label"
                                            optionValue="value"
                                        />
                                    </div>
                                    <div class="field">
                                        <label for="maturity">Maturity (month / year)</label>
                                        <div class="field-controls">
                                            <DatePicker
                                                id="maturity"
                                                v-model="maturityDate"
                                                view="month"
                                                dateFormat="mm/yy"
                                                :minDate="minMaturity"
                                                showIcon
                                                iconDisplay="input"
                                                class="field-input"
                                            />
                                            <span class="text-color-secondary text-sm">{{ durationText }}</span>
                                            <Slider v-model="maturityYears" :min="1" :max="50" :step="1" class="field-slider" />
                                        </div>
                                    </div>
                                    <div class="field">
                                        <label for="taxes">Taxes (%)</label>
                                        <div class="field-controls">
                                            <InputNumber id="taxes" v-model="taxesPct" :min="0" :max="100" :minFractionDigits="1" :maxFractionDigits="2" class="field-input" />
                                            <Slider v-model="taxesPct" :min="0" :max="50" :step="1" class="field-slider" />
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
                                        <span class="result-label">Net YTM (ROI)</span>
                                        <span class="result-value font-bold">{{ projection.netYTM.toFixed(2) }}%</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Total Return</span>
                                        <span class="result-value">{{ formatCurrency(projection.totalReturn) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Total Coupons (gross)</span>
                                        <span class="result-value">{{ formatCurrency(projection.totalCoupons) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Tax Impact</span>
                                        <span class="result-value">−{{ formatCurrency(projection.couponTaxPaid + projection.capitalGainTaxPaid) }}</span>
                                    </div>
                                    <div class="result-row">
                                        <span class="result-label">Capital Gain / Loss</span>
                                        <span class="result-value">{{ formatCurrency(projection.capitalGain) }}</span>
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
                        <div class="field">
                            <label>Attachment</label>
                            <div v-if="activeCaseAttachmentId" class="flex align-items-center gap-2">
                                <Button icon="ti ti-paperclip" label="View attachment" text size="small" @click="viewAttachment" />
                                <Button icon="ti ti-trash" text rounded severity="danger" size="small" @click="handleAttachmentDelete" v-tooltip.bottom="'Remove attachment'" />
                            </div>
                            <FileInput v-else v-model="selectedAttachmentFile" accept=".jpg,.jpeg,.png,.webp,.pdf" label="Upload file" icon="ti ti-paperclip" />
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
</style>
