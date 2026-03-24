<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useQueryClient } from '@tanstack/vue-query'
import { useEntryMutations } from '@/composables/useEntryMutations'
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
import Textarea from 'primevue/textarea'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { getAccountBalance } from '@/lib/api/report'
import { uploadAttachment, deleteAttachment, getAttachmentUrl } from '@/lib/api/Attachment'
import FileInput from '@/components/common/FileInput.vue'

const queryClient = useQueryClient()
const { createEntry, updateEntry, isCreating, isUpdating } = useEntryMutations()
const backendError = ref('')
const { pickerDateFormat, dateValidation } = useDateFormat()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 }, // stored delta from backend
    balance: { type: Number, default: 0 }, // stored target balance from backend
    date: { type: Date, default: () => new Date() },
    accountId: { type: Number, default: null },
    visible: { type: Boolean, default: false },
    attachmentId: { type: Number, default: null },
    notes: { type: String, default: '' }
})

const selectedFile = ref(null)
const existingAttachmentId = ref(null)
const attachmentPendingDelete = ref(false)

// Current balance fetched from the API
const currentBalance = ref(0)
const balanceLoading = ref(false)
const balanceFetched = ref(false)

function defaultDescription(date) {
    const d = date || new Date()
    const month = d.toLocaleString('en', { month: 'short' })
    return `Revaluation ${month} ${d.getFullYear()}`
}

const formValues = ref({
    description: props.description || (props.isEdit ? '' : defaultDescription(props.date)),
    notes: props.notes,
    targetBalance: 0,
    AccountId: getFormattedAccountId(props.accountId),
    date: getDateOnly(props.date)
})

// Track whether the user has touched the balance field
const balanceTouched = ref(false)

// Balance before this revaluation: when editing, subtract the existing delta
const baseBalance = computed(() => {
    if (!balanceFetched.value) return 0
    if (props.isEdit) return currentBalance.value - (props.amount ?? 0)
    return currentBalance.value
})

// Computed delta shown to the user
const computedDelta = computed(() => {
    if (!balanceFetched.value || !balanceTouched.value) return null
    return (formValues.value.targetBalance ?? 0) - baseBalance.value
})

// Recorded balance: use stored value, or compute from current balance + delta for old entries
const recordedBalance = computed(() => {
    if (!props.isEdit) return null
    if (props.balance) return props.balance
    if (balanceFetched.value && props.amount != null) {
        return currentBalance.value + (props.amount ?? 0)
    }
    return null
})

// Fetch the current account balance for the selected account and date
async function fetchBalance() {
    const accountId = extractAccountId(formValues.value.AccountId)
    const date = formValues.value.date
    if (!accountId || !date) {
        currentBalance.value = 0
        balanceFetched.value = false
        return
    }
    balanceLoading.value = true
    try {
        const bal = await getAccountBalance(accountId, toDateString(date))
        currentBalance.value = bal
        balanceFetched.value = true
    } catch (e) {
        console.error('Failed to fetch account balance:', e)
        currentBalance.value = 0
        balanceFetched.value = false
    } finally {
        balanceLoading.value = false
    }
}

// Reset form when dialog opens
const formKey = ref(0)
watch(() => props.visible, async (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description || (props.isEdit ? '' : defaultDescription(props.date)),
        notes: props.notes,
        targetBalance: 0,
        AccountId: getFormattedAccountId(props.accountId),
        date: getDateOnly(props.date)
    }
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    currentBalance.value = 0
    balanceFetched.value = false
    balanceTouched.value = false
    formKey.value++

    await fetchBalance()
    if (props.isEdit) {
        formValues.value.targetBalance = recordedBalance.value ?? currentBalance.value
        balanceTouched.value = true
    }
})

watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

// Re-fetch balance when account or date changes
watch(() => formValues.value.AccountId, () => fetchBalance())
watch(() => formValues.value.date, () => fetchBalance())

const resolver = computed(() => {
    return zodResolver(z.object({
        date: dateValidation.value,
        targetBalance: z.number({ message: 'Target balance is required' }),
        AccountId: accountValidation
    }))
})

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    return `${action} Revaluation`
})

const formatCurrency = (val) => {
    return val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const canSubmit = computed(() => balanceFetched.value && !balanceLoading.value)

const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    if (!canSubmit.value) return
    const values = getSubmitValues(e, formValues)
    const accountId = extractAccountId(values.AccountId)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const targetBalance = Number(formValues.value.targetBalance ?? 0)
    const date = values.date ?? formValues.value.date

    if (accountId == null) return

    // Compute delta: target - base balance (excluding existing revaluation effect)
    const delta = targetBalance - baseBalance.value

    const entryData = {
        description,
        notes: (formValues.value.notes ?? '').toString(),
        Amount: delta,
        balance: targetBalance,
        date: toDateString(date),
        accountId,
        type: 'revaluation'
    }

    backendError.value = ''

    const shouldDeleteAttachment = attachmentPendingDelete.value && !!existingAttachmentId.value
    const fileToUpload = selectedFile.value

    try {
        let result
        if (props.isEdit) {
            result = await updateEntry({ id: props.entryId, ...entryData })
        } else {
            result = await createEntry(entryData)
        }

        const savedId = props.isEdit ? props.entryId : result.id

        let attachmentChanged = false
        if (shouldDeleteAttachment) {
            try { await deleteAttachment(savedId); attachmentChanged = true } catch (e) { console.error('Failed to delete attachment:', e) }
        }
        if (fileToUpload) {
            try { await uploadAttachment(savedId, fileToUpload); attachmentChanged = true } catch (e) { console.error('Failed to upload attachment:', e) }
        }
        if (attachmentChanged) {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }

        emit('update:visible', false)
    } catch (error) {
        backendError.value = getApiErrorMessage(error)
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} revaluation:`, error)
    }
}

const viewAttachment = () => {
    window.open(getAttachmentUrl(props.entryId), '_blank')
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
                        placeholder="e.g. Q1 pension revaluation"
                    />
                </div>


                <!-- Target Balance Field -->
                <div>
                    <label for="targetBalance" class="form-label">Balance</label>
                    <InputNumber
                        id="targetBalance"
                        v-model="formValues.targetBalance"
                        name="targetBalance"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                        v-focus
                        @blur="balanceTouched = true"
                    />
                    <Message v-if="$form.targetBalance?.invalid" severity="error" size="small">
                        {{ $form.targetBalance.error?.message }}
                    </Message>
                </div>

                <!-- Computed Delta (info) -->
                <div v-if="computedDelta !== null" class="delta-info">
                    <label class="form-label">Change</label>
                    <div :class="['text-lg font-semibold', computedDelta >= 0 ? 'text-green-500' : 'text-red-500']">
                        {{ computedDelta >= 0 ? '+' : '' }}{{ formatCurrency(computedDelta) }}
                    </div>
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
                        :accountTypes="['pension', 'savings']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId.error?.message }}
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

                <div>
                    <label class="form-label">Attachment</label>
                    <div v-if="existingAttachmentId && !attachmentPendingDelete && !selectedFile" class="flex align-items-center gap-2">
                        <Button
                            icon="ti ti-paperclip"
                            label="View attachment"
                            text
                            size="small"
                            @click="viewAttachment"
                        />
                        <Button
                            icon="ti ti-trash"
                            text
                            rounded
                            severity="danger"
                            size="small"
                            @click="attachmentPendingDelete = true"
                            v-tooltip.bottom="'Remove attachment'"
                        />
                    </div>
                    <div v-else>
                        <FileInput
                            v-model="selectedFile"
                            accept=".jpg,.jpeg,.png,.webp,.pdf"
                            label="Choose file"
                            icon="ti ti-paperclip"
                        />
                    </div>
                </div>

                <div class="flex justify-content-end gap-3">
                    <Button
                        type="submit"
                        label="Save"
                        icon="ti ti-check"
                        :loading="isCreating || isUpdating"
                        :disabled="!canSubmit"
                    />
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
