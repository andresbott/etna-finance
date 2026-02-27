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
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import { useQueryClient } from '@tanstack/vue-query'
import AccountSelector from '@/components/AccountSelector.vue'
import { useInstruments } from '@/composables/useInstruments'
import { useMutation } from '@tanstack/vue-query'
import { createStockTransaction } from '@/lib/api/Entry'
import { useEntries } from '@/composables/useEntries'
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

const queryClient = useQueryClient()
const backendError = ref('')
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntries({})

const createMutation = useMutation({
    mutationFn: createStockTransaction,
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['entries'] })
        queryClient.invalidateQueries({ queryKey: ['portfolio-positions'] })
    }
})

const isSaving = computed(() => createMutation.isPending.value)
const { pickerDateFormat } = useDateFormat()

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
    autofocusAmount: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

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
    (instrumentId) => {
        if (instrumentId != null && instrumentId >= 1) {
            const currentDesc = (formValues.value.description ?? '').toString().trim()
            if (!currentDesc) {
                formValues.value = { ...formValues.value, description: getDefaultDescriptionForInstrument(instrumentId) }
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
        date: z.date(),
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

const handleSubmit = async (e) => {
    e.preventDefault?.()
    const values = getSubmitValues(e, formValues)
    const v = formValues.value
    const description = (values.description ?? v.description ?? '').toString().trim()
    const instrumentId = Number(values.instrumentId ?? v.instrumentId)
    const quantity = Number(values.quantity ?? v.quantity)
    const date = values.date ?? v.date
    const invId = extractAccountId(values.InvestmentAccountId ?? v.InvestmentAccountId)
    const cashId = extractAccountId(values.CashAccountId ?? v.CashAccountId)
    const netAmount = props.operationType === 'sell' ? Number(values.targetAmount ?? v.targetAmount ?? 0) : 0
    const fees = props.operationType === 'sell' ? Number(values.fees ?? v.fees ?? 0) : 0
    const total =
        props.operationType === 'buy'
            ? Number(values.originAmount ?? v.originAmount ?? 0)
            : netAmount + fees
    const pricePerShare = Number(values.pricePerShare ?? v.pricePerShare ?? 0)
    const stockAmount = quantity * pricePerShare

    if (!description || !(instrumentId >= 1) || !(quantity > 0) || invId == null || cashId == null || !(total > 0)) return
    if (props.operationType === 'buy' && !(stockAmount > 0)) return

    backendError.value = ''
    try {
        if (props.isEdit && props.entryId != null) {
            const updatePayload = {
                id: String(props.entryId),
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                totalAmount: total,
                investmentAccountId: invId,
                cashAccountId: cashId,
                ...(props.operationType === 'buy' ? { StockAmount: stockAmount } : { fees })
            }
            await updateEntry(updatePayload)
        } else {
            const payload = {
                type: props.operationType === 'sell' ? 'stocksell' : 'stockbuy',
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                totalAmount: total,
                investmentAccountId: invId,
                cashAccountId: cashId,
                ...(props.operationType === 'buy' ? { StockAmount: stockAmount } : { fees })
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error(props.isEdit ? 'Failed to update stock operation:' : 'Failed to create stock operation:', err)
    }
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
                <!-- Top section: instrument, description, date, quantity -->
                <div class="flex flex-column gap-3">
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
                        <Message v-if="$form.instrumentId?.invalid" severity="error" size="small">
                            {{ $form.instrumentId?.error?.message }}
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
                        <Message v-if="$form.description?.invalid" severity="error" size="small">
                            {{ $form.description?.error?.message }}
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
                        <Message v-if="$form.date?.invalid" severity="error" size="small">
                            {{ $form.date?.error?.message }}
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
                        <Message v-if="$form.quantity?.invalid" severity="error" size="small">
                            {{ $form.quantity?.error?.message }}
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
                </div>

                <Divider />

                <!-- Two columns: buy = cash → investment, sell = investment → cash -->
                <div class="flex flex-row">
                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <template v-if="operationType === 'buy'">
                            <div>
                                <label for="CashAccountId" class="form-label">Cash account</label>
                                <AccountSelector
                                    v-model="formValues.CashAccountId"
                                    name="CashAccountId"
                                    placeholder="Select cash account"
                                    :accountTypes="['cash', 'checkin', 'savings']"
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
                        </template>
                        <template v-else>
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
                        </template>
                    </div>

                    <div class="flex align-items-center justify-content-center px-2">
                        <i class="pi pi-arrow-right text-2xl"></i>
                    </div>

                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <template v-if="operationType === 'buy'">
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
                        </template>
                        <template v-else>
                            <div>
                                <label for="CashAccountId" class="form-label">Cash account</label>
                                <AccountSelector
                                    v-model="formValues.CashAccountId"
                                    name="CashAccountId"
                                    placeholder="Select cash account"
                                    :accountTypes="['cash', 'checkin', 'savings']"
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
                        </template>
                    </div>
                </div>

                <div class="flex justify-content-end gap-3 pt-3">
                    <Button
                        type="submit"
                        label="Save"
                        icon="pi pi-check"
                        :loading="isSaving"
                    />
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
</style>
