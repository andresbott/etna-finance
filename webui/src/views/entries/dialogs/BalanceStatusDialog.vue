<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.ts'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString,
    getSubmitValues
} from '@/composables/useEntryDialogForm'

import AccountSelector from '@/components/AccountSelector.vue'
import Message from 'primevue/message'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries({})
const backendError = ref('')
const { pickerDateFormat, dateValidation } = useDateFormat()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    accountId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

const formValues = ref({
    description: props.description,
    amount: props.amount,
    AccountId: getFormattedAccountId(props.accountId),
    date: getDateOnly(props.date)
})

// Watch props to update form values when editing
const formKey = ref(0)
watch(props, (newProps) => {
    formValues.value = {
        description: newProps.description,
        amount: newProps.amount,
        AccountId: getFormattedAccountId(newProps.accountId),
        date: getDateOnly(newProps.date)
    }
    formKey.value++
})

watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const resolver = computed(() => {
    return zodResolver(z.object({
        date: dateValidation.value,
        amount: z.number({ message: 'Stated balance is required' }),
        AccountId: accountValidation
    }))
})

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    return `${action} Balance Status`
})

const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    const values = getSubmitValues(e, formValues)
    const accountId = extractAccountId(values.AccountId)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const amount = Number(values.amount ?? formValues.value.amount ?? 0)
    const date = values.date ?? formValues.value.date

    if (accountId == null) return

    const entryData = {
        description,
        Amount: amount,
        date: toDateString(date),
        accountId,
        type: 'balancestatus'
    }

    backendError.value = ''
    try {
        if (props.isEdit) {
            await updateEntry({ id: props.entryId, ...entryData })
        } else {
            await createEntry(entryData)
        }
        emit('update:visible', false)
    } catch (error) {
        backendError.value = getApiErrorMessage(error)
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} balance status:`, error)
    }
}

// Define the emit for updating visibility
const emit = defineEmits(['update:visible'])
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="dialogTitle"
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
            <div class="flex flex-column gap-3">
                <!-- Description Field -->
                <div>
                    <label for="description" class="form-label">Description</label>
                    <InputText
                        id="description"
                        v-model="formValues.description"
                        name="description"
                        placeholder="e.g. March bank statement"
                        v-focus
                    />
                </div>

                <!-- Stated Balance Field -->
                <div>
                    <label for="amount" class="form-label">Stated Balance</label>
                    <InputNumber
                        id="amount"
                        v-model="formValues.amount"
                        name="amount"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                    />
                    <Message v-if="$form.amount?.invalid" severity="error" size="small">
                        {{ $form.amount.error?.message }}
                    </Message>
                </div>

                <!-- Date Field -->
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
                        {{ $form.date.error?.message }}
                    </Message>
                </div>

                <!-- Account field -->
                <div>
                    <label for="AccountId" class="form-label">Account</label>
                    <AccountSelector
                        v-model="formValues.AccountId"
                        name="AccountId"
                        :accountTypes="['cash', 'checkin', 'bank', 'savings', 'lent']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId.error?.message }}
                    </Message>
                </div>

                <div class="flex justify-content-end gap-3">
                    <Button
                        type="submit"
                        label="Save"
                        icon="pi pi-check"
                        :loading="isCreating || isUpdating"
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
