<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import Message from 'primevue/message'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import { useQueryClient } from '@tanstack/vue-query'
import AccountSelector from '@/components/AccountSelector.vue'
import { useInstruments } from '@/composables/useInstruments'
import { useMutation } from '@tanstack/vue-query'
import { createStockGrant } from '@/lib/api/Entry'
import { useEntries } from '@/composables/useEntries'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString
} from '@/composables/useEntryDialogForm'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'

const queryClient = useQueryClient()
const backendError = ref('')
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntries({})

const createMutation = useMutation({
    mutationFn: createStockGrant,
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['entries'] })
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
    instrumentId: { type: Number, default: null },
    description: { type: String, default: '' },
    quantity: { type: Number, default: 0 },
    fairMarketValue: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    accountId: { type: Number, default: null }
})

const emit = defineEmits(['update:visible'])
const formKey = ref(0)
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    quantity: props.quantity,
    fairMarketValue: props.fairMarketValue,
    date: getDateOnly(props.date),
    AccountId: getFormattedAccountId(props.accountId)
})

watch(
    () => [props.visible, props.instrumentId, props.description, props.quantity, props.fairMarketValue, props.date, props.accountId],
    () => {
        if (props.visible) {
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                quantity: props.quantity,
                fairMarketValue: props.fairMarketValue,
                date: getDateOnly(props.date),
                AccountId: getFormattedAccountId(props.accountId)
            }
            formKey.value++
        }
    }
)

function getDefaultDescriptionForInstrument(id) {
    const symbol = instrumentById.value[id]?.symbol ?? ''
    return symbol ? `Grant ${symbol}` : ''
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

const resolver = computed(() =>
    zodResolver(
        z.object({
            instrumentId: z.number().min(1, { message: 'Instrument is required' }),
            description: z.string().min(1, { message: 'Description is required' }),
            quantity: z.number().min(0.0001, { message: 'Quantity must be greater than 0' }),
            fairMarketValue: z.number().min(0, { message: 'FMV cannot be negative' }).optional().default(0),
            date: z.date(),
            AccountId: accountValidation
        })
    )
)

const handleSubmit = async (e) => {
    e.preventDefault?.()
    // Build payload from formValues (PrimeVue Form submit event may not include values)
    const v = formValues.value
    const description = (v.description ?? '').toString().trim()
    const instrumentId = Number(v.instrumentId)
    const quantity = Number(v.quantity)
    const date = v.date ? new Date(v.date) : new Date()
    const accountId = extractAccountId(v.AccountId)
    const fairMarketValue = Number(v.fairMarketValue ?? 0)

    if (!description) return
    if (!instrumentId || instrumentId < 1) return
    if (!(quantity > 0)) return
    if (!accountId) return

    backendError.value = ''
    try {
        if (props.isEdit && props.entryId != null) {
            await updateEntry({
                id: String(props.entryId),
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                fairMarketValue,
                accountId
            })
        } else {
            const payload = {
                type: 'stockgrant',
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                fairMarketValue,
                accountId
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        backendError.value = getApiErrorMessage(err)
        console.error(props.isEdit ? 'Failed to update grant:' : 'Failed to create grant:', err)
    }
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="isEdit ? 'Edit grant' : 'Grant instrument'"
        class="entry-dialog"
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
                        placeholder="e.g. RSU vest, gift, award"
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
                    <label for="fairMarketValue" class="form-label">Fair market value per share (optional)</label>
                    <InputNumber
                        id="fairMarketValue"
                        v-model="formValues.fairMarketValue"
                        name="fairMarketValue"
                        :minFractionDigits="2"
                        :maxFractionDigits="4"
                    />
                    <Message v-if="$form.fairMarketValue?.invalid" severity="error" size="small">
                        {{ $form.fairMarketValue?.error?.message }}
                    </Message>
                </div>
                <div>
                    <label for="AccountId" class="form-label">Account (receives instruments)</label>
                    <AccountSelector
                        v-model="formValues.AccountId"
                        name="AccountId"
                        placeholder="Select investment or unvested account"
                        :accountTypes="['investment', 'unvested']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId?.error?.message }}
                    </Message>
                </div>

                <div class="flex justify-content-end gap-3 pt-3">
                    <Button
                        type="button"
                        label="Save"
                        icon="pi pi-check"
                        :loading="isSaving"
                        @click="handleSubmit({ preventDefault: () => {} })"
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
