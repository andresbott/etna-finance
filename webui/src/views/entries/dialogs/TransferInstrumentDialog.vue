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
import { createStockTransfer } from '@/lib/api/Entry'
import { useEntries } from '@/composables/useEntries'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString,
    getSubmitValues
} from '@/composables/useEntryDialogForm'

const queryClient = useQueryClient()
const { instruments: instrumentsData } = useInstruments()
const { updateEntry } = useEntries({})

const createMutation = useMutation({
    mutationFn: createStockTransfer,
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
    date: { type: Date, default: () => new Date() },
    originAccountId: { type: Number, default: null },
    targetAccountId: { type: Number, default: null }
})

const emit = defineEmits(['update:visible'])

const formValues = ref({
    instrumentId: props.instrumentId,
    description: props.description,
    quantity: props.quantity,
    date: getDateOnly(props.date),
    OriginAccountId: getFormattedAccountId(props.originAccountId),
    TargetAccountId: getFormattedAccountId(props.targetAccountId)
})

watch(
    () => [
        props.visible,
        props.instrumentId,
        props.description,
        props.quantity,
        props.date,
        props.originAccountId,
        props.targetAccountId
    ],
    () => {
        if (props.visible) {
            formValues.value = {
                instrumentId: props.instrumentId,
                description: props.description,
                quantity: props.quantity,
                date: getDateOnly(props.date),
                OriginAccountId: getFormattedAccountId(props.originAccountId),
                TargetAccountId: getFormattedAccountId(props.targetAccountId)
            }
        }
    }
)

function getDefaultDescriptionForInstrument(id) {
    const symbol = instrumentById.value[id]?.symbol ?? ''
    return symbol ? `Transfer ${symbol}` : ''
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

const accountValidation = z
    .union([z.null(), z.record(z.boolean())])
    .refine((obj) => obj != null, { message: 'Account must be selected' })

const resolver = computed(() =>
    zodResolver(
        z.object({
            instrumentId: z.number().min(1, { message: 'Instrument is required' }),
            description: z.string().min(1, { message: 'Description is required' }),
            quantity: z.number().min(0.0001, { message: 'Quantity must be greater than 0' }),
            date: z.date(),
            OriginAccountId: accountValidation,
            TargetAccountId: accountValidation
        })
    )
)

const handleSubmit = async (e) => {
    e.preventDefault?.()
    const values = getSubmitValues(e, formValues)
    const v = formValues.value
    const description = (values.description ?? v.description ?? '').toString().trim()
    const instrumentId = Number(values.instrumentId ?? v.instrumentId)
    const quantity = Number(values.quantity ?? v.quantity)
    const date = values.date ?? v.date
    const originId = extractAccountId(values.OriginAccountId ?? v.OriginAccountId)
    const targetId = extractAccountId(values.TargetAccountId ?? v.TargetAccountId)

    if (!description || !(instrumentId >= 1) || !(quantity > 0) || originId == null || targetId == null) return

    try {
        if (props.isEdit && props.entryId != null) {
            await updateEntry({
                id: String(props.entryId),
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                originAccountId: originId,
                targetAccountId: targetId
            })
        } else {
            const payload = {
                type: 'stocktransfer',
                description,
                date: toDateString(date),
                instrumentId,
                quantity,
                originAccountId: originId,
                targetAccountId: targetId
            }
            await createMutation.mutateAsync(payload)
        }
        emit('update:visible', false)
    } catch (err) {
        console.error(props.isEdit ? 'Failed to update instrument transfer:' : 'Failed to create instrument transfer:', err)
    }
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="isEdit ? 'Edit transfer instrument' : 'Transfer instrument'"
        class="entry-dialog entry-dialog--wide"
    >
        <Form
            v-slot="$form"
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
            @submit="handleSubmit"
        >
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
                        placeholder="e.g. RSU vest transfer to brokerage"
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

                <div class="transfer-instrument-accounts flex flex-row gap-3 align-items-start">
                    <div class="flex flex-column gap-2 flex-1 transfer-instrument-accounts__cell">
                        <label for="OriginAccountId" class="form-label">From (source account)</label>
                        <AccountSelector
                            v-model="formValues.OriginAccountId"
                            name="OriginAccountId"
                            placeholder="Select investment or unvested account"
                            :accountTypes="['investment', 'unvested']"
                        />
                        <Message v-if="$form.OriginAccountId?.invalid" severity="error" size="small">
                            {{ $form.OriginAccountId?.error?.message }}
                        </Message>
                    </div>
                    <div class="flex align-items-center pt-4">
                        <i class="pi pi-arrow-right text-xl text-color-secondary"></i>
                    </div>
                    <div class="flex flex-column gap-2 flex-1 transfer-instrument-accounts__cell">
                        <label for="TargetAccountId" class="form-label">To (target account)</label>
                        <AccountSelector
                            v-model="formValues.TargetAccountId"
                            name="TargetAccountId"
                            placeholder="Select investment or unvested account"
                            :accountTypes="['investment', 'unvested']"
                        />
                        <Message v-if="$form.TargetAccountId?.invalid" severity="error" size="small">
                            {{ $form.TargetAccountId?.error?.message }}
                        </Message>
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
/* Allow flex cells to shrink so the row doesn't force horizontal scroll */
.transfer-instrument-accounts__cell {
    min-width: 0;
}
</style>
