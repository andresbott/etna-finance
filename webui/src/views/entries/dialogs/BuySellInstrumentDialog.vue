<script setup>
import { ref, watch, computed } from 'vue'
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
import { useInstruments } from '@/composables/useInstruments'
import { createStockTransaction } from '@/lib/api/Entry'
import { useEntryMutations } from '@/composables/useEntryMutations'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString,
    getSubmitValues
} from '@/composables/useEntryDialogForm'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { useLots } from '@/composables/useLots'
import { getLatestPrice } from '@/lib/api/MarketData'

const queryClient = useQueryClient()
const backendError = ref('')
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntryMutations()

const createMutation = useMutation({
    mutationFn: createStockTransaction,
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['entries'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
    }
})

const isSaving = computed(() => createMutation.isPending.value)
const { pickerDateFormat, dateValidation } = useDateFormat()

const instruments = computed(() => instrumentsData.value ?? [])

const instrumentOptions = computed(() =>
    instruments.value.map((s) => ({ label: `${s.symbol} – ${s.name}`, value: s.id }))
)

const instrumentById = computed(() =>
    Object.fromEntries((instruments.value ?? []).map((i) => [i.id, i]))
)

const selectedInstrumentCurrency = computed(() => {
    const id = formValues.value.instrumentId
    if (!id) return null
    return instrumentById.value[id]?.currency ?? null
})

const props = defineProps({
    visible: { type: Boolean, default: false },
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    operationType: { type: String, default: 'buy' },
    instrumentId: { type: Number, default: null },
    description: { type: String, default: '' },
    quantity: { type: Number, default: 0 },
    pricePerShare: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    investmentAccountId: { type: Number, default: null },
    cashAccountId: { type: Number, default: null },
    cashAmount: { type: Number, default: null },
    fees: { type: Number, default: 0 },
    autofocusAmount: { type: Boolean, default: false },
    notes: { type: String, default: '' }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)

// Multi-step state (sell only)
const step = ref(1)
const step1Errors = ref({})

watch(() => props.visible, (v) => {
    if (!v) {
        backendError.value = ''
    } else {
        step.value = 1
        step1Errors.value = {}
    }
})

function validateStep1() {
    const errors = {}
    const v = formValues.value
    if (!v.instrumentId || v.instrumentId < 1) errors.instrumentId = 'Instrument is required'
    if (!v.description?.toString().trim()) errors.description = 'Description is required'
    if (!v.date) errors.date = 'Date is required'
    if (!v.quantity || v.quantity <= 0) errors.quantity = 'Quantity must be greater than 0'
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
}

const initialOriginAmount = () => {
    if (props.cashAmount != null && props.cashAmount > 0) return props.cashAmount
    const q = props.quantity
    const p = props.pricePerShare
    if (q != null && p != null && !Number.isNaN(q) && !Number.isNaN(p) && p > 0) return q * p
    return 0
}

const initialTargetAmount = () => {
    if (props.operationType === 'sell' && props.cashAmount != null && props.cashAmount >= 0) return props.cashAmount
    const q = props.quantity
    const p = props.pricePerShare
    if (q != null && p != null && !Number.isNaN(q) && !Number.isNaN(p) && p > 0) return q * p
    return 0
}

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    notes: props.notes,
    quantity: props.quantity,
    pricePerShare: props.pricePerShare,
    date: getDateOnly(props.date),
    originAmount: initialOriginAmount(),
    targetAmount: initialTargetAmount(),
    fees: 0,
    InvestmentAccountId: getFormattedAccountId(props.investmentAccountId),
    CashAccountId: getFormattedAccountId(props.cashAccountId)
})

// When dialog opens, always sync form from props so edit shows the same values you entered (e.g. net + fees for sell).
watch(
    () => [props.visible, props.instrumentId, props.description, props.quantity, props.pricePerShare, props.date, props.investmentAccountId, props.cashAccountId, props.cashAmount, props.fees],
    async () => {
        if (props.visible) {
            await nextTick()
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                notes: props.notes,
                quantity: props.quantity,
                pricePerShare: props.pricePerShare,
                date: getDateOnly(props.date),
                originAmount: initialOriginAmount(),
                targetAmount: initialTargetAmount(),
                fees: props.operationType === 'sell' ? (props.fees ?? 0) : 0,
                InvestmentAccountId: getFormattedAccountId(props.investmentAccountId),
                CashAccountId: getFormattedAccountId(props.cashAccountId)
            }
            formKey.value++
        }
    }
)

function getDefaultDescriptionForInstrument(id) {
    const symbol = instrumentById.value[id]?.symbol ?? ''
    if (!symbol) return ''
    const action = props.operationType === 'sell' ? 'Sell' : 'Buy'
    return `${action} ${symbol}`
}

watch(
    () => formValues.value.instrumentId,
    async (instrumentId) => {
        if (instrumentId != null && instrumentId >= 1) {
            const keepExisting = props.isEdit && instrumentId === props.instrumentId
            const currentDesc = (formValues.value.description ?? '').toString().trim()
            if (!keepExisting || !currentDesc) {
                formValues.value = { ...formValues.value, description: getDefaultDescriptionForInstrument(instrumentId) }
            }
            // Prefill price unless we're editing an existing entry and the instrument hasn't changed
            const keepExistingPrice = keepExisting
            if (!keepExistingPrice) {
                const symbol = instrumentById.value[instrumentId]?.symbol
                if (symbol) {
                    const latest = await getLatestPrice(symbol)
                    if (latest?.price && formValues.value.instrumentId === instrumentId) {
                        formValues.value = { ...formValues.value, pricePerShare: latest.price }
                    }
                }
            }
        }
    }
)

const totalAmount = computed(() => {
    const q = formValues.value.quantity
    const p = formValues.value.pricePerShare
    if (q == null || p == null || Number.isNaN(q) || Number.isNaN(p)) return null
    return q * p
})

const totalAmountDisplay = computed(() => {
    const t = totalAmount.value
    return t != null && !Number.isNaN(t) ? t.toFixed(2) : ''
})

const resolver = computed(() => {
    const base = {
        instrumentId: z.number().min(1, { message: 'Instrument is required' }),
        description: z.string().min(1, { message: 'Description is required' }),
        quantity: z.number().min(0.0001, { message: 'Quantity must be greater than 0' }),
        pricePerShare: z.number().min(0, { message: 'Price must be 0 or greater' }),
        date: dateValidation.value,
        InvestmentAccountId: accountValidation,
        CashAccountId: accountValidation
    }
    if (props.operationType === 'buy') {
        base.originAmount = z.number().min(0.01, { message: 'Amount must be greater than 0' })
    }
    if (props.operationType === 'sell') {
        base.targetAmount = z.number().min(0.01, { message: 'Amount must be greater than 0' })
        base.fees = z.number().min(0, { message: 'Fees cannot be negative' }).optional().default(0)
    }
    return zodResolver(z.object(base))
})

const dialogTitle = computed(() => {
    const op = props.operationType === 'sell' ? 'Sell' : 'Buy'
    return props.isEdit ? `Edit ${op} instrument` : `${op} instrument`
})

// ---------------------------------------------------------------------------
// Lot selection (only for sell operations)
// ---------------------------------------------------------------------------
const lotAllocations = ref([])

const investmentAccountIdRef = computed(() => extractAccountId(formValues.value.InvestmentAccountId))
const instrumentIdRef = computed(() => formValues.value.instrumentId ?? null)

const { lots } = useLots(investmentAccountIdRef, instrumentIdRef)

const showLotTable = computed(() =>
    props.operationType === 'sell' &&
    lots.value.length > 0 &&
    investmentAccountIdRef.value != null &&
    instrumentIdRef.value != null
)

watch(
    () => [props.visible, formValues.value.instrumentId, formValues.value.InvestmentAccountId],
    () => { lotAllocations.value = [] }
)

const allocatedTotal = computed(() =>
    lotAllocations.value.reduce((sum, a) => sum + (a.quantity || 0), 0)
)

const sellQty = computed(() => formValues.value.quantity ?? 0)

const lotSelectionError = computed(() => {
    if (!showLotTable.value) return null
    if (allocatedTotal.value === 0) return 'Please allocate shares to at least one lot'
    if (Math.abs(allocatedTotal.value - sellQty.value) > 0.0001) return 'Allocated quantity must equal sell quantity'
    return null
})

const allocationMismatch = computed(() => lotSelectionError.value !== null)

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

function applyFIFO() {
    const newAllocs = []
    let remaining = sellQty.value
    for (const lot of lots.value) {
        if (remaining <= 0) break
        const take = Math.min(lot.quantity, remaining)
        if (take > 0) {
            newAllocs.push({ lotId: lot.id, quantity: take })
            remaining -= take
        }
    }
    lotAllocations.value = newAllocs
}

function applyLIFO() {
    const newAllocs = []
    let remaining = sellQty.value
    const reversed = [...lots.value].reverse()
    for (const lot of reversed) {
        if (remaining <= 0) break
        const take = Math.min(lot.quantity, remaining)
        if (take > 0) {
            newAllocs.push({ lotId: lot.id, quantity: take })
            remaining -= take
        }
    }
    lotAllocations.value = newAllocs
}

function clearAllocations() {
    lotAllocations.value = []
}

function getLotGainLoss(lot) {
    const qty = getLotQty(lot.id)
    if (!qty || qty <= 0) return null
    const sellPrice = formValues.value.pricePerShare || 0
    return (sellPrice - lot.costPerShare) * qty
}

// Core save logic shared by buy (form submit) and sell (direct click).
const doSave = async () => {
    const v = formValues.value
    const description = (v.description ?? '').toString().trim()
    const notes = (v.notes ?? '').toString()
    const instrumentId = Number(v.instrumentId)
    const quantity = Number(v.quantity)
    const date = v.date
    const invId = extractAccountId(v.InvestmentAccountId)
    const cashId = extractAccountId(v.CashAccountId)
    const netAmount = props.operationType === 'sell' ? Number(v.targetAmount ?? 0) : 0
    const fees = props.operationType === 'sell' ? Number(v.fees ?? 0) : 0
    const total =
        props.operationType === 'buy'
            ? Number(v.originAmount ?? 0)
            : netAmount + fees
    const pricePerShare = Number(v.pricePerShare ?? 0)
    const stockAmount = quantity * pricePerShare

    if (!description || !(instrumentId >= 1) || !(quantity > 0) || invId == null || cashId == null || !(total > 0)) return
    if (props.operationType === 'buy' && !(stockAmount > 0)) return
    if (allocationMismatch.value) return

    const activeAllocations = lotAllocations.value.filter(a => a.quantity > 0)
    const lotAllocationsPayload =
        props.operationType === 'sell' && activeAllocations.length > 0
            ? activeAllocations
            : undefined

    backendError.value = ''
    try {
        if (props.isEdit && props.entryId != null) {
            const updatePayload = {
                id: String(props.entryId),
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                quantity,
                totalAmount: total,
                investmentAccountId: invId,
                cashAccountId: cashId,
                ...(props.operationType === 'buy' ? { StockAmount: stockAmount } : { fees }),
                ...(lotAllocationsPayload ? { lotAllocations: lotAllocationsPayload } : {})
            }
            await updateEntry(updatePayload)
        } else {
            const payload = {
                type: props.operationType === 'sell' ? 'stocksell' : 'stockbuy',
                description,
                notes,
                date: toDateString(date),
                instrumentId,
                quantity,
                totalAmount: total,
                investmentAccountId: invId,
                cashAccountId: cashId,
                ...(props.operationType === 'buy' ? { StockAmount: stockAmount } : { fees }),
                ...(lotAllocationsPayload ? { lotAllocations: lotAllocationsPayload } : {})
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error(props.isEdit ? 'Failed to update stock operation:' : 'Failed to create stock operation:', err)
    }
}

// Buy flow: triggered by PrimeVue Form @submit (type="submit" button).
const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    await doSave()
}

// Sell step 2 flow: triggered by direct @click to bypass PrimeVue Form validation
// (step 1 fields are unmounted via v-if in step 2, causing the resolver to fail).
const handleSellSave = async () => {
    await doSave()
}
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
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
            @submit="handleSubmit"
        >
            <Message v-if="backendError" severity="error" :closable="false" class="mb-2">{{ backendError }}</Message>
            <div v-focustrap class="flex flex-column gap-3">

                <!-- Step indicator (sell only) -->
                <div v-if="operationType === 'sell'" class="step-indicator">
                    <div :class="['step-dot', { 'step-dot--active': step === 1 }]">1</div>
                    <div class="step-line"></div>
                    <div :class="['step-dot', { 'step-dot--active': step === 2 }]">2</div>
                </div>

                <!-- Step 1 (buy always, sell step 1): instrument, description, date, quantity, price -->
                <div v-if="operationType === 'buy' || step === 1" class="flex flex-column gap-3">
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
                        <Message v-if="step1Errors.instrumentId || $form.instrumentId?.invalid" severity="error" size="small">
                            {{ step1Errors.instrumentId ?? $form.instrumentId?.error?.message }}
                        </Message>
                    </div>
                    <div>
                        <label for="description" class="form-label">Description</label>
                        <InputText
                            id="description"
                            v-model="formValues.description"
                            name="description"
                            placeholder="e.g. Buy 10 AAPL"
                        />
                        <Message v-if="step1Errors.description || $form.description?.invalid" severity="error" size="small">
                            {{ step1Errors.description ?? $form.description?.error?.message }}
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
                        <Message v-if="step1Errors.date || $form.date?.invalid" severity="error" size="small">
                            {{ step1Errors.date ?? $form.date?.error?.message }}
                        </Message>
                    </div>
                    <div>
                        <label for="quantity" class="form-label">Quantity</label>
                        <InputNumber
                            id="quantity"
                            v-model="formValues.quantity"
                            name="quantity"
                            :minFractionDigits="0"
                            :maxFractionDigits="4"
                        />
                        <Message v-if="step1Errors.quantity || $form.quantity?.invalid" severity="error" size="small">
                            {{ step1Errors.quantity ?? $form.quantity?.error?.message }}
                        </Message>
                    </div>
                    <div>
                        <label for="pricePerShare" class="form-label">Price per share</label>
                        <InputNumber
                            id="pricePerShare"
                            v-model="formValues.pricePerShare"
                            name="pricePerShare"
                            :minFractionDigits="2"
                            :maxFractionDigits="4"
                        />
                        <Message v-if="$form.pricePerShare?.invalid" severity="error" size="small">
                            {{ $form.pricePerShare?.error?.message }}
                        </Message>
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

                <!-- Buy: divider + horizontal two-column layout (cash → investment) -->
                <template v-if="operationType === 'buy'">
                    <Divider />
                    <div class="flex flex-row">
                        <div class="flex flex-column gap-3 flex-1 p-2">
                            <div>
                                <label for="CashAccountId" class="form-label">Cash account</label>
                                <AccountSelector
                                    v-model="formValues.CashAccountId"
                                    name="CashAccountId"
                                    placeholder="Select cash account"
                                    :accountTypes="['cash', 'checkin', 'savings', 'lent']"
                                />
                                <Message v-if="$form.CashAccountId?.invalid" severity="error" size="small">
                                    {{ $form.CashAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <label for="originAmount" class="form-label">Amount</label>
                                <InputNumber
                                    id="originAmount"
                                    v-model="formValues.originAmount"
                                    name="originAmount"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                />
                                <Message v-if="$form.originAmount?.invalid" severity="error" size="small">
                                    {{ $form.originAmount?.error?.message }}
                                </Message>
                            </div>
                        </div>
                        <div class="flex align-items-center justify-content-center px-2">
                            <i class="pi pi-arrow-right text-2xl"></i>
                        </div>
                        <div class="flex flex-column gap-3 flex-1 p-2">
                            <div>
                                <label for="InvestmentAccountId" class="form-label">Investment account</label>
                                <AccountSelector
                                    v-model="formValues.InvestmentAccountId"
                                    name="InvestmentAccountId"
                                    placeholder="Select investment or unvested account"
                                    :accountTypes="['investment', 'unvested']"
                                />
                                <Message v-if="$form.InvestmentAccountId?.invalid" severity="error" size="small">
                                    {{ $form.InvestmentAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <span class="form-label">Total</span>
                                <p class="amount-display mt-1 mb-0">{{ totalAmountDisplay || '—' }}</p>
                            </div>
                        </div>
                    </div>
                </template>

                <!-- Sell step 2: origin (investment + lots) → destination (cash + amounts) -->
                <template v-if="operationType === 'sell' && step === 2">
                    <!-- Origin -->
                    <p class="section-heading">From</p>
                    <div class="flex flex-column gap-3">
                        <div>
                            <label for="InvestmentAccountId" class="form-label">Investment account</label>
                            <AccountSelector
                                v-model="formValues.InvestmentAccountId"
                                name="InvestmentAccountId"
                                placeholder="Select investment or unvested account"
                                :accountTypes="['investment', 'unvested']"
                            />
                            <Message v-if="$form.InvestmentAccountId?.invalid" severity="error" size="small">
                                {{ $form.InvestmentAccountId?.error?.message }}
                            </Message>
                        </div>
                        <div>
                            <span class="form-label">Total</span>
                            <p class="amount-display mt-1 mb-0">{{ totalAmountDisplay || '—' }}</p>
                        </div>
                        <!-- Lot selection -->
                        <div v-if="showLotTable" class="lot-selection">
                            <div class="flex align-items-center justify-content-between mb-2">
                                <span class="font-semibold text-sm">Lot selection</span>
                                <div class="flex gap-2">
                                    <Button type="button" label="FIFO" size="small" severity="secondary" @click="applyFIFO" />
                                    <Button type="button" label="LIFO" size="small" severity="secondary" @click="applyLIFO" />
                                    <Button type="button" label="Clear" size="small" severity="secondary" @click="clearAllocations" />
                                </div>
                            </div>
                            <table class="lot-table w-full">
                                <thead>
                                    <tr>
                                        <th class="text-left">Open date</th>
                                        <th class="text-right">Available</th>
                                        <th class="text-right">Cost/share</th>
                                        <th class="text-right">Sell qty</th>
                                        <th class="text-right">Gain / Loss</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="lot in lots" :key="lot.id">
                                        <td>{{ lot.openDate?.split('T')[0] ?? lot.openDate }}</td>
                                        <td class="text-right">{{ lot.quantity }}</td>
                                        <td class="text-right">{{ lot.costPerShare?.toFixed(4) }}</td>
                                        <td class="text-right">
                                            <InputNumber
                                                :modelValue="getLotQty(lot.id)"
                                                @update:modelValue="setLotQty(lot.id, $event)"
                                                :min="0"
                                                :max="lot.quantity"
                                                :minFractionDigits="0"
                                                :maxFractionDigits="4"
                                                size="small"
                                                class="lot-qty-input"
                                            />
                                        </td>
                                        <td class="text-right">
                                            <span
                                                v-if="getLotGainLoss(lot) != null"
                                                :class="['lot-gl', getLotGainLoss(lot) >= 0 ? 'lot-gl--gain' : 'lot-gl--loss']"
                                            >
                                                {{ getLotGainLoss(lot) >= 0 ? '+' : '' }}{{ getLotGainLoss(lot).toFixed(2) }}
                                            </span>
                                            <span v-else class="lot-gl--empty">—</span>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                            <div class="flex justify-content-end mt-2">
                                <span :class="['lot-summary', { 'lot-summary--mismatch': allocationMismatch }]">
                                    Allocated {{ allocatedTotal.toFixed(4) }} of {{ sellQty.toFixed(4) }}
                                </span>
                            </div>
                            <Message v-if="lotSelectionError" severity="error" size="small" :closable="false">
                                {{ lotSelectionError }}
                            </Message>
                        </div>
                    </div>

                    <Divider />

                    <!-- Destination -->
                    <p class="section-heading">To</p>
                    <div class="flex flex-column gap-3">
                        <div>
                            <label for="CashAccountId" class="form-label">Cash account</label>
                            <AccountSelector
                                v-model="formValues.CashAccountId"
                                name="CashAccountId"
                                placeholder="Select cash account"
                                :accountTypes="['cash', 'checkin', 'savings', 'lent']"
                                :currency="selectedInstrumentCurrency"
                            />
                            <Message v-if="$form.CashAccountId?.invalid" severity="error" size="small">
                                {{ $form.CashAccountId?.error?.message }}
                            </Message>
                        </div>
                        <div>
                            <label for="targetAmount" class="form-label">Net amount (after fees)</label>
                            <InputNumber
                                id="targetAmount"
                                v-model="formValues.targetAmount"
                                name="targetAmount"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                            />
                            <Message v-if="$form.targetAmount?.invalid" severity="error" size="small">
                                {{ $form.targetAmount?.error?.message }}
                            </Message>
                        </div>
                        <div>
                            <label for="fees" class="form-label">Fees (optional)</label>
                            <InputNumber
                                id="fees"
                                v-model="formValues.fees"
                                name="fees"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                            />
                            <Message v-if="$form.fees?.invalid" severity="error" size="small">
                                {{ $form.fees?.error?.message }}
                            </Message>
                        </div>
                    </div>
                </template>

                <!-- Action buttons -->
                <div class="flex justify-content-end gap-3 pt-3">
                    <!-- Sell step 1: Next -->
                    <template v-if="operationType === 'sell' && step === 1">
                        <Button type="button" label="Next" icon="pi pi-arrow-right" iconPos="right" @click="goToStep2" />
                    </template>
                    <!-- Sell step 2: Back + Save (direct click, bypasses PrimeVue Form submit) -->
                    <template v-else-if="operationType === 'sell' && step === 2">
                        <Button type="button" label="Save" icon="pi pi-check" :loading="isSaving" @click="handleSellSave" />
                        <Button type="button" label="Back" icon="pi pi-arrow-left" severity="secondary" @click="goToStep1" />
                    </template>
                    <!-- Buy: Save -->
                    <template v-else>
                        <Button type="submit" label="Save" icon="pi pi-check" :loading="isSaving" />
                    </template>
                    <Button
                        type="button"
                        label="Cancel"
                        icon="pi pi-times"
                        severity="secondary"
                        @click="$emit('update:visible', false)"
                    />
                </div>
            </div>
        </Form>
    </Dialog>
</template>

<style scoped>
.amount-display {
    font-weight: 500;
}

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

.lot-gl {
    font-variant-numeric: tabular-nums;
    font-weight: 500;
}

.lot-gl--gain {
    color: var(--p-green-500, #22c55e);
}

.lot-gl--loss {
    color: var(--p-red-500, #ef4444);
}

.lot-gl--empty {
    color: var(--p-text-muted-color, #9ca3af);
}

.lot-summary {
    font-size: 0.875rem;
    color: var(--p-text-muted-color, #6b7280);
}

.lot-summary--mismatch {
    color: var(--p-red-500, #ef4444);
    font-weight: 600;
}
</style>
