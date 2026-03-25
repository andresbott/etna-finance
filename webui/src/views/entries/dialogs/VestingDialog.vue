<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import Message from 'primevue/message'
import Divider from 'primevue/divider'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import { useQueryClient, useMutation } from '@tanstack/vue-query'
import AccountSelector from '@/components/AccountSelector.vue'
import CategorySelect from '@/components/common/CategorySelect.vue'
import { useInstruments } from '@/composables/useInstruments'
import { createStockVest } from '@/lib/api/Entry'
import { useEntryMutations } from '@/composables/useEntryMutations'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString
} from '@/composables/useEntryDialogForm'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { useLots } from '@/composables/useLots'
import { getLatestPrice } from '@/lib/api/MarketData'

const queryClient = useQueryClient()
const backendError = ref('')
const priceFetchStatus = ref('idle')
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntryMutations()

const createMutation = useMutation({
    mutationFn: createStockVest,
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['entries'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-lots'] })
    }
})

const isSaving = computed(() => createMutation.isPending.value)
const { pickerDateFormat, dateValidation, formatDate } = useDateFormat()

const instruments = computed(() => instrumentsData.value ?? [])

const instrumentOptions = computed(() =>
    instruments.value.map((s) => ({ label: `${s.symbol} – ${s.name}`, value: s.id }))
)

const instrumentById = computed(() =>
    Object.fromEntries((instruments.value ?? []).map((i) => [i.id, i]))
)

const props = defineProps({
    visible: { type: Boolean, default: false },
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    instrumentId: { type: Number, default: null },
    description: { type: String, default: '' },
    vestingPrice: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    originAccountId: { type: Number, default: null },
    targetAccountId: { type: Number, default: null },
    categoryId: { type: Number, default: 0 },
    notes: { type: String, default: '' },
    initialLotAllocations: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)
const categoryId = ref(props.categoryId)

// Multi-step state
const step = ref(1)
const step1Errors = ref({})
const step2Errors = ref({})

watch(() => props.visible, (v) => {
    if (!v) {
        backendError.value = ''
    } else {
        step.value = 1
        step1Errors.value = {}
        step2Errors.value = {}
    }
})

function validateStep1() {
    const errors = {}
    const v = formValues.value
    if (!v.instrumentId || v.instrumentId < 1) errors.instrumentId = 'Instrument is required'
    if (!v.description?.toString().trim()) errors.description = 'Description is required'
    if (!v.date) errors.date = 'Date is required'
    if (!v.vestingPrice || v.vestingPrice <= 0) errors.vestingPrice = 'Vesting price must be greater than 0'
    const originId = extractAccountId(v.OriginAccountId)
    if (originId == null) errors.OriginAccountId = 'Source account is required'
    const targetId = extractAccountId(v.TargetAccountId)
    if (targetId == null) errors.TargetAccountId = 'Target account is required'
    if (!categoryId.value || categoryId.value < 1) errors.categoryId = 'Income category is required'
    return errors
}

function goToStep2() {
    step1Errors.value = validateStep1()
    if (Object.keys(step1Errors.value).length === 0) {
        step.value = 2
        step1Errors.value = {}
    }
}

function goToStep1() {
    step.value = 1
    step2Errors.value = {}
}

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    notes: props.notes,
    vestingPrice: props.vestingPrice,
    date: getDateOnly(props.date),
    OriginAccountId: getFormattedAccountId(props.originAccountId),
    TargetAccountId: getFormattedAccountId(props.targetAccountId)
})

// Reset form when dialog opens
watch(
    () => [props.visible, props.instrumentId, props.description, props.vestingPrice, props.date, props.originAccountId, props.targetAccountId],
    async () => {
        if (props.visible) {
            await nextTick()
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                notes: props.notes,
                vestingPrice: props.vestingPrice,
                date: getDateOnly(props.date),
                OriginAccountId: getFormattedAccountId(props.originAccountId),
                TargetAccountId: getFormattedAccountId(props.targetAccountId)
            }
            categoryId.value = props.categoryId
            formKey.value++
        }
    }
)

function getDefaultDescriptionForInstrument(id) {
    const symbol = instrumentById.value[id]?.symbol ?? ''
    return symbol ? `Vest ${symbol}` : ''
}

watch(
    () => formValues.value.instrumentId,
    async (instrumentId) => {
        if (instrumentId != null && instrumentId >= 1) {
            const keepExisting = props.isEdit && instrumentId === props.instrumentId
            const currentDesc = (formValues.value.description ?? '').toString().trim()
            if (!keepExisting || !currentDesc) {
                formValues.value = { ...formValues.value, description: getDefaultDescriptionForInstrument(instrumentId) }
                formKey.value++
            }
            // Prefill vesting price from latest market price unless editing with same instrument
            if (!keepExisting) {
                const symbol = instrumentById.value[instrumentId]?.symbol
                if (symbol) {
                    priceFetchStatus.value = 'loading'
                    try {
                        const latest = await getLatestPrice(symbol)
                        if (latest?.price && formValues.value.instrumentId === instrumentId) {
                            formValues.value = { ...formValues.value, vestingPrice: latest.price }
                            priceFetchStatus.value = 'success'
                        } else {
                            priceFetchStatus.value = 'failed'
                        }
                    } catch {
                        priceFetchStatus.value = 'failed'
                    }
                }
            }
        }
    }
)

// ---------------------------------------------------------------------------
// Lot selection (step 2)
// ---------------------------------------------------------------------------
const lotAllocations = ref([])

const originAccountIdRef = computed(() => extractAccountId(formValues.value.OriginAccountId))
const instrumentIdRef = computed(() => formValues.value.instrumentId ?? null)
const beforeDateRef = computed(() => formValues.value.date ? toDateString(formValues.value.date) : null)

const { lots } = useLots(originAccountIdRef, instrumentIdRef, beforeDateRef)

const visibleLots = computed(() =>
    lots.value.filter(l => getLotAvailable(l) > 0)
)

const totalAvailable = computed(() =>
    visibleLots.value.reduce((sum, l) => sum + getLotAvailable(l), 0)
)

// When editing, each lot's "available" quantity is its current DB quantity plus
// whatever was already allocated to this vest (which was subtracted when the vest was created).
const initialAllocMap = computed(() => {
    const map = {}
    for (const a of (props.initialLotAllocations ?? [])) {
        map[a.lotId] = (map[a.lotId] || 0) + a.quantity
    }
    return map
})

function getLotAvailable(lot) {
    return lot.quantity + (initialAllocMap.value[lot.id] || 0)
}

function getLotQty(lotId) {
    const entry = lotAllocations.value.find(a => a.lotId === lotId)
    return entry ? entry.quantity : 0
}

function setLotQty(lotId, qty) {
    const idx = lotAllocations.value.findIndex(a => a.lotId === lotId)
    if (idx >= 0) {
        if (!qty || qty <= 0) {
            lotAllocations.value.splice(idx, 1)
        } else {
            lotAllocations.value[idx].quantity = qty
        }
    } else if (qty > 0) {
        lotAllocations.value.push({ lotId, quantity: qty })
    }
}

const allocatedTotal = computed(() =>
    lotAllocations.value.reduce((sum, a) => sum + (a.quantity || 0), 0)
)

const totalIncome = computed(() => {
    const price = formValues.value.vestingPrice ?? 0
    return allocatedTotal.value * price
})

const totalIncomeDisplay = computed(() => {
    const t = totalIncome.value
    return t != null && !Number.isNaN(t) && t > 0 ? t.toFixed(2) : ''
})

const lotSelectionError = computed(() => {
    if (allocatedTotal.value === 0) return 'Please allocate shares to at least one lot'
    return null
})

const hasLotError = computed(() => lotSelectionError.value !== null)

watch(
    () => [props.visible, formValues.value.instrumentId, formValues.value.OriginAccountId],
    ([visible]) => {
        if (visible && props.isEdit && props.initialLotAllocations?.length > 0) {
            lotAllocations.value = props.initialLotAllocations.map(a => ({ lotId: a.lotId, quantity: a.quantity }))
        } else {
            lotAllocations.value = []
        }
    }
)

function applyAll() {
    const newAllocs = []
    for (const lot of visibleLots.value) {
        const avail = getLotAvailable(lot)
        if (avail > 0) {
            newAllocs.push({ lotId: lot.id, quantity: avail })
        }
    }
    lotAllocations.value = newAllocs
}

function clearAllocations() {
    lotAllocations.value = []
}

// ---------------------------------------------------------------------------
// Save logic
// ---------------------------------------------------------------------------
const doSave = async () => {
    const v = formValues.value
    const description = (v.description ?? '').toString().trim()
    const notes = (v.notes ?? '').toString()
    const instrumentId = Number(v.instrumentId)
    const vestingPrice = Number(v.vestingPrice ?? 0)
    const date = v.date
    const originId = extractAccountId(v.OriginAccountId)
    const targetId = extractAccountId(v.TargetAccountId)

    if (!description || !(instrumentId >= 1) || !(vestingPrice > 0) || originId == null || targetId == null) return
    if (hasLotError.value) return

    const activeAllocations = lotAllocations.value.filter(a => a.quantity > 0)
    if (activeAllocations.length === 0) return

    backendError.value = ''
    try {
        if (props.isEdit && props.entryId != null) {
            await updateEntry({
                id: String(props.entryId),
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                vestingPrice,
                categoryId: categoryId.value,
                originAccountId: originId,
                targetAccountId: targetId,
                lotAllocations: activeAllocations
            })
            queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
            queryClient.invalidateQueries({ queryKey: ['portfolio-lots'] })
        } else {
            const payload = {
                type: 'stockvest',
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                vestingPrice,
                categoryId: categoryId.value,
                originAccountId: originId,
                targetAccountId: targetId,
                lotAllocations: activeAllocations
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error(props.isEdit ? 'Failed to update stock vest:' : 'Failed to create stock vest:', err)
    }
}

const handleVestSave = async () => {
    step2Errors.value = {}
    if (hasLotError.value) {
        step2Errors.value.lots = lotSelectionError.value
        return
    }
    await doSave()
}

const dialogTitle = computed(() =>
    props.isEdit ? 'Edit vest' : 'Vest shares'
)
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="dialogTitle"
        class="entry-dialog entry-dialog--wide"
    >
        <Form
            :key="formKey"
            v-slot="$form"
            :resolver="zodResolver(z.object({}))"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
        >
            <Message v-if="backendError" severity="error" :closable="false" class="mb-2">{{ backendError }}</Message>
            <div v-focustrap class="flex flex-column gap-3">

                <!-- Step indicator -->
                <div class="step-indicator">
                    <div :class="['step-dot', { 'step-dot--active': step === 1 }]">1</div>
                    <div class="step-line"></div>
                    <div :class="['step-dot', { 'step-dot--active': step === 2 }]">2</div>
                </div>

                <!-- Step 1: basics -->
                <div v-if="step === 1" class="flex flex-column gap-3">
                    <div>
                        <label for="instrumentId" class="form-label">Investment instrument</label>
                        <Select
                            id="instrumentId"
                            v-model="formValues.instrumentId"
                            name="instrumentId"
                            :options="instrumentOptions"
                            optionLabel="label"
                            optionValue="value"
                            placeholder="Select instrument"
                        />
                        <Message v-if="step1Errors.instrumentId" severity="error" size="small">
                            {{ step1Errors.instrumentId }}
                        </Message>
                    </div>
                    <div>
                        <label for="description" class="form-label">Description</label>
                        <InputText
                            id="description"
                            v-model="formValues.description"
                            name="description"
                            placeholder="e.g. Vest AAPL"
                        />
                        <Message v-if="step1Errors.description" severity="error" size="small">
                            {{ step1Errors.description }}
                        </Message>
                    </div>
                    <div>
                        <label for="date" class="form-label">Date</label>
                        <DatePicker
                            id="date"
                            name="date"
                            v-model="formValues.date"
                            :showIcon="true"
                            iconDisplay="input"
                            :dateFormat="pickerDateFormat"
                            :showButtonBar="true"
                        />
                        <Message v-if="step1Errors.date" severity="error" size="small">
                            {{ step1Errors.date }}
                        </Message>
                    </div>
                    <div>
                        <label for="vestingPrice" class="form-label">Vesting price per share</label>
                        <InputNumber
                            id="vestingPrice"
                            v-model="formValues.vestingPrice"
                            name="vestingPrice"
                            :minFractionDigits="2"
                            :maxFractionDigits="4"
                        />
                        <Message v-if="step1Errors.vestingPrice" severity="error" size="small">
                            {{ step1Errors.vestingPrice }}
                        </Message>
                        <small v-if="priceFetchStatus === 'loading'" class="text-color-secondary">Fetching latest price...</small>
                        <small v-else-if="priceFetchStatus === 'failed'" style="color: var(--p-orange-500, #f59e0b)">Could not fetch price. Enter manually.</small>
                    </div>

                    <div>
                        <CategorySelect v-model="categoryId" type="income" label="Income category" />
                        <Message v-if="step1Errors.categoryId" severity="error" size="small">
                            {{ step1Errors.categoryId }}
                        </Message>
                    </div>

                    <div class="flex flex-row gap-3 align-items-start">
                        <div class="flex flex-column gap-2 flex-1" style="min-width: 0">
                            <label for="OriginAccountId" class="form-label">Source (restricted stock) account</label>
                            <AccountSelector
                                v-model="formValues.OriginAccountId"
                                name="OriginAccountId"
                                placeholder="Select restricted stock account"
                                :accountTypes="['restrictedstock']"
                            />
                            <Message v-if="step1Errors.OriginAccountId" severity="error" size="small">
                                {{ step1Errors.OriginAccountId }}
                            </Message>
                        </div>
                        <div class="flex align-items-center pt-4">
                            <i class="ti ti-arrow-right text-xl text-color-secondary"></i>
                        </div>
                        <div class="flex flex-column gap-2 flex-1" style="min-width: 0">
                            <label for="TargetAccountId" class="form-label">Target (investment) account</label>
                            <AccountSelector
                                v-model="formValues.TargetAccountId"
                                name="TargetAccountId"
                                placeholder="Select investment account"
                                :accountTypes="['investment']"
                            />
                            <Message v-if="step1Errors.TargetAccountId" severity="error" size="small">
                                {{ step1Errors.TargetAccountId }}
                            </Message>
                        </div>
                    </div>

                    <!-- Notes Field -->
                    <div>
                        <label for="notes" class="form-label">Notes</label>
                        <Textarea
                            id="notes"
                            v-model="formValues.notes"
                            name="notes"
                            rows="3"
                            autoResize
                            fluid
                        />
                    </div>
                </div>

                <!-- Step 2: lot selection -->
                <template v-if="step === 2">
                    <p class="section-heading">Select lots to vest</p>
                    <div class="flex flex-column gap-3">
                        <div v-if="visibleLots.length > 0" class="lot-selection">
                            <div class="flex align-items-center justify-content-between mb-2">
                                <span class="font-semibold text-sm">Lot selection</span>
                                <div class="flex gap-2">
                                    <Button type="button" label="All" size="small" severity="secondary" @click="applyAll" />
                                    <Button type="button" label="Clear" size="small" severity="secondary" @click="clearAllocations" />
                                </div>
                            </div>
                            <table class="lot-table w-full">
                                <thead>
                                    <tr>
                                        <th class="text-left">Open date</th>
                                        <th class="text-right">Available</th>
                                        <th class="text-right">Grant FMV</th>
                                        <th class="text-right">Vest qty</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="lot in visibleLots" :key="lot.id">
                                        <td>{{ formatDate(lot.openDate) }}</td>
                                        <td class="text-right">{{ getLotAvailable(lot).toFixed(3) }}</td>
                                        <td class="text-right">{{ lot.costPerShare?.toFixed(3) }}</td>
                                        <td class="text-right">
                                            <InputNumber
                                                :modelValue="getLotQty(lot.id)"
                                                @update:modelValue="setLotQty(lot.id, $event)"
                                                :min="0"
                                                :max="getLotAvailable(lot)"
                                                :minFractionDigits="0"
                                                :maxFractionDigits="4"
                                                size="small"
                                                class="lot-qty-input"
                                            />
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                            <div class="flex justify-content-between mt-2">
                                <span class="lot-summary">
                                    Total quantity: {{ allocatedTotal.toFixed(4).replace(/\.?0+$/, '') }}
                                </span>
                                <span v-if="totalIncomeDisplay" class="lot-summary">
                                    Total income: {{ totalIncomeDisplay }}
                                </span>
                            </div>
                            <Message v-if="step2Errors.lots" severity="error" size="small" :closable="false">
                                {{ step2Errors.lots }}
                            </Message>
                        </div>
                        <div v-else>
                            <Message severity="warn" :closable="false">
                                No open lots found for the selected instrument and source account.
                            </Message>
                        </div>
                    </div>
                </template>

                <!-- Action buttons -->
                <div class="flex justify-content-end gap-3 pt-3">
                    <template v-if="step === 1">
                        <Button type="button" label="Next" icon="ti ti-arrow-right" iconPos="right" @click="goToStep2" />
                    </template>
                    <template v-else-if="step === 2">
                        <Button type="button" label="Save" icon="ti ti-check" :loading="isSaving" @click="handleVestSave" />
                        <Button type="button" label="Back" icon="ti ti-arrow-left" severity="secondary" @click="goToStep1" />
                    </template>
                    <Button
                        type="button"
                        label="Cancel"
                        icon="ti ti-x"
                        severity="secondary"
                        @click="$emit('update:visible', false)"
                    />
                </div>
            </div>
        </Form>
    </Dialog>
</template>

<style scoped>
.step-indicator {
    display: flex;
    align-items: center;
    gap: 0;
    margin-bottom: 0.25rem;
}

.step-dot {
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    font-weight: 600;
    flex-shrink: 0;
    background: var(--p-surface-200, #e5e7eb);
    color: var(--p-text-muted-color, #6b7280);
    transition: background 0.2s, color 0.2s;
}

.step-dot--active {
    background: var(--p-primary-color, #6366f1);
    color: #fff;
}

.step-line {
    flex: 1;
    height: 2px;
    background: var(--p-surface-200, #e5e7eb);
}

.section-heading {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--p-text-muted-color, #6b7280);
    margin: 0 0 0.5rem 0;
}

.lot-table {
    border-collapse: collapse;
    font-size: 0.875rem;
}

.lot-table th,
.lot-table td {
    padding: 0.3rem 0.5rem;
    border-bottom: 1px solid var(--p-surface-border, #e2e8f0);
    white-space: nowrap;
}

.lot-table th {
    font-weight: 600;
    color: var(--p-text-muted-color, #6b7280);
}

.lot-qty-input {
    width: 7rem;
}

.lot-summary {
    font-size: 0.875rem;
    color: var(--p-text-muted-color, #6b7280);
}
</style>
