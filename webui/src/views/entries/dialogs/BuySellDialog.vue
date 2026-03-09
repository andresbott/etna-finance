<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { accountValidation } from '@/utils/entryValidation'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId
} from '@/composables/useEntryDialogForm'
import Message from 'primevue/message'
import Divider from 'primevue/divider'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import { useQueryClient } from '@tanstack/vue-query'
import AccountSelector from '@/components/AccountSelector.vue'
import { useInstruments } from '@/composables/useInstruments'
import { useDateFormat } from '@/composables/useDateFormat'
import { useMutation } from '@tanstack/vue-query'
import { createStockTransaction } from '@/lib/api/Entry'
import { getApiErrorMessage } from '@/utils/apiError'

const queryClient = useQueryClient()
const backendError = ref('')
const { instruments: instrumentsData } = useInstruments()

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

const props = defineProps({
    visible: { type: Boolean, default: false },
    isEdit: { type: Boolean, default: false },
    operationType: { type: String, default: 'buy' },
    instrumentId: { type: Number, default: null },
    description: { type: String, default: '' },
    quantity: { type: Number, default: 0 },
    pricePerShare: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    investmentAccountId: { type: Number, default: null },
    cashAccountId: { type: Number, default: null },
    autofocusAmount: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    quantity: props.quantity,
    pricePerShare: props.pricePerShare,
    date: getDateOnly(props.date),
    InvestmentAccountId: getFormattedAccountId(props.investmentAccountId),
    CashAccountId: getFormattedAccountId(props.cashAccountId)
})

watch(
    () => [props.visible, props.instrumentId, props.description, props.quantity, props.pricePerShare, props.date, props.investmentAccountId, props.cashAccountId],
    () => {
        if (props.visible) {
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                quantity: props.quantity,
                pricePerShare: props.pricePerShare,
                date: getDateOnly(props.date),
                InvestmentAccountId: getFormattedAccountId(props.investmentAccountId),
                CashAccountId: getFormattedAccountId(props.cashAccountId)
            }
            formKey.value++
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

const resolver = computed(() =>
    zodResolver(
        z.object({
            instrumentId: z.number().min(1, { message: 'Instrument is required' }),
            description: z.string().min(1, { message: 'Description is required' }),
            quantity: z.number().min(0.0001, { message: 'Quantity must be greater than 0' }),
            pricePerShare: z.number().min(0, { message: 'Price must be 0 or greater' }),
            date: dateValidation.value,
            InvestmentAccountId: accountValidation,
            CashAccountId: accountValidation
        })
    )
)

const dialogTitle = computed(() => {
    const op = props.operationType === 'sell' ? 'Sell' : 'Buy'
    return props.isEdit ? `Edit ${op} instrument` : `${op} instrument`
})

const handleSubmit = async (e) => {
    if (!e.valid) return

    const invId = extractAccountId(e.values.InvestmentAccountId)
    const cashId = extractAccountId(e.values.CashAccountId)
    const q = Number(e.values.quantity)
    const p = Number(e.values.pricePerShare)
    const total = q * p

    const payload = {
        type: props.operationType === 'sell' ? 'stocksell' : 'stockbuy',
        description: e.values.description,
        date: new Date(e.values.date).toISOString().slice(0, 10),
        instrumentId: e.values.instrumentId,
        quantity: q,
        totalAmount: total,
        investmentAccountId: invId,
        cashAccountId: cashId
    }

    backendError.value = ''
    try {
        await createMutation.mutateAsync(payload)
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error('Failed to create stock operation:', err)
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
                <!-- Top section: description and date (same as transfer) -->
                <div class="flex flex-column gap-3">
                    <div>
                        <label for="description" class="form-label">Description</label>
                        <InputText
                            id="description"
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
                </div>

                <Divider />

                <!-- From / To layout: buy = cash → investment, sell = investment → cash -->
                <div class="flex flex-row">
                    <!-- From (left): cash account + amount for buy; investment + instrument + qty/price for sell -->
                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <h3 class="m-0 text-lg font-medium">From</h3>
                        <template v-if="operationType === 'buy'">
                            <div>
                                <label for="CashAccountId" class="form-label">Cash account</label>
                                <AccountSelector
                                    v-model="formValues.CashAccountId"
                                    name="CashAccountId"
                                    placeholder="Select cash account"
                                    :accountTypes="['cash', 'checkin', 'bank', 'savings', 'lent']"
                                />
                                <Message v-if="$form.CashAccountId?.invalid" severity="error" size="small">
                                    {{ $form.CashAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <label class="form-label">Amount</label>
                                <p class="amount-display mt-1 mb-0">{{ totalAmountDisplay || '—' }}</p>
                            </div>
                        </template>
                        <template v-else>
                            <div>
                                <label for="InvestmentAccountId" class="form-label">Investment account</label>
                                <AccountSelector
                                    v-model="formValues.InvestmentAccountId"
                                    name="InvestmentAccountId"
                                    placeholder="Select investment account"
                                    :accountTypes="['investment']"
                                />
                                <Message v-if="$form.InvestmentAccountId?.invalid" severity="error" size="small">
                                    {{ $form.InvestmentAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <label for="instrumentId" class="form-label">Investment instrument</label>
                                <Select
                                    id="instrumentId"
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
                            <div>
                                <span class="form-label">Total</span>
                                <p class="amount-display mt-1 mb-0">{{ totalAmountDisplay || '—' }}</p>
                            </div>
                        </template>
                    </div>

                    <div class="flex align-items-center justify-content-center px-2">
                        <i class="pi pi-arrow-right text-2xl"></i>
                    </div>

                    <!-- To (right): investment + instrument + qty/price for buy; cash account + amount for sell -->
                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <h3 class="m-0 text-lg font-medium">To</h3>
                        <template v-if="operationType === 'buy'">
                            <div>
                                <label for="InvestmentAccountId" class="form-label">Investment account</label>
                                <AccountSelector
                                    v-model="formValues.InvestmentAccountId"
                                    name="InvestmentAccountId"
                                    placeholder="Select investment account"
                                    :accountTypes="['investment']"
                                />
                                <Message v-if="$form.InvestmentAccountId?.invalid" severity="error" size="small">
                                    {{ $form.InvestmentAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <label for="instrumentId" class="form-label">Investment instrument</label>
                                <Select
                                    id="instrumentId"
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
                                    :accountTypes="['cash', 'checkin', 'bank', 'savings', 'lent']"
                                />
                                <Message v-if="$form.CashAccountId?.invalid" severity="error" size="small">
                                    {{ $form.CashAccountId?.error?.message }}
                                </Message>
                            </div>
                            <div>
                                <label class="form-label">Amount</label>
                                <p class="amount-display mt-1 mb-0">{{ totalAmountDisplay || '—' }}</p>
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
