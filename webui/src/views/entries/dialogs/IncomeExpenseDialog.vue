<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useQueryClient } from '@tanstack/vue-query'
import { useEntryMutations } from '@/composables/useEntryMutations'
import { useAccounts } from '@/composables/useAccounts'
import { findAccountById } from '@/utils/accountUtils'
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
import CategorySelect from '@/components/common/CategorySelect.vue'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { uploadAttachment, deleteAttachment, getAttachmentUrl } from '@/lib/api/Attachment'
import FileInput from '@/components/common/FileInput.vue'
import { useSettingsStore } from '@/store/settingsStore'

const queryClient = useQueryClient()
const { createEntry, updateEntry, isCreating, isUpdating } = useEntryMutations()
const backendError = ref('')
const { accounts } = useAccounts()
const { pickerDateFormat, dateValidation } = useDateFormat()
const settingsStore = useSettingsStore()
const maxAttachmentBytes = computed(() => settingsStore.maxAttachmentSizeMB * 1024 * 1024)

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryType: { type: String, required: true }, // 'expense' or 'income'
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 },
    stockAmount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },

    accountId: { type: Number, default: null },
    visible: { type: Boolean, default: false },
    categoryId: { type: Number, default: 0 },
    autofocusAmount: { type: Boolean, default: false },
    attachmentId: { type: Number, default: null },
    notes: { type: String, default: '' }
})
const categoryId = ref(props.categoryId)

const selectedFile = ref(null)
const existingAttachmentId = ref(null)
const attachmentPendingDelete = ref(false)
const fileError = ref('')

const amountInputRef = ref(null)

// Watch for visibility and autofocusAmount to focus the amount field
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })
watch(() => [props.visible, props.autofocusAmount], ([visible, autofocus]) => {
    if (visible && autofocus) {
        // Use multiple delays to ensure dialog is fully ready
        setTimeout(() => {
            const inputElement = amountInputRef.value?.$el?.querySelector('input')
            if (inputElement) {
                inputElement.focus()
                inputElement.select()
            }
        }, 180)
    }
})

const formValues = ref({
    description: props.description,
    notes: props.notes,
    amount: props.amount,
    AccountId: getFormattedAccountId(props.accountId),
    stockAmount: props.stockAmount,
    date: getDateOnly(props.date)
})

// Reset form when dialog opens
const formKey = ref(0)
watch(() => props.visible, (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description,
        notes: props.notes,
        amount: props.amount,
        AccountId: getFormattedAccountId(props.accountId),
        stockAmount: props.stockAmount,
        date: getDateOnly(props.date)
    }
    categoryId.value = props.categoryId
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    fileError.value = ''
    formKey.value++
})

// Track if selected account is of type stocks
const selectedAccount = ref(null)

// Direct handler for account selection changes
const handleAccountSelection = (accountObject) => {
    updateSelectedAccount(accountObject)
}

// Function to update selected account
const updateSelectedAccount = (accountObject) => {
    if (!accountObject || !accounts.value) {
        selectedAccount.value = null
        return
    }

    const accountId = extractAccountId(accountObject)

    if (isNaN(accountId) || accountId === null) {
        selectedAccount.value = null
        return
    }

    selectedAccount.value = findAccountById(accounts.value, accountId)
}

// Also keep the watch for reactive updates, with immediate flag
watch(
    () => formValues.value.AccountId,
    (newValue) => {
        updateSelectedAccount(newValue)
    },
    { immediate: true }
)

// Check if selected account is of type "stocks"
const isStocksAccount = computed(() => {
    return selectedAccount.value?.type === 'stocks'
})

// Build the resolver for income/expense entries
const resolver = computed(() => {
    // Base schema for income and expense entry types
    const baseSchema = {
        description: z.string().min(1, { message: 'Description is required' }),
        date: dateValidation.value,
        amount: z.number().min(0.01, { message: 'Amount must be greater than 0' }),
        AccountId: accountValidation
    }

    // Add stockAmount to schema if selected account is of type stocks
    if (isStocksAccount.value) {
        baseSchema.stockAmount = z
            .number()
            .min(0.01, { message: 'Stock amount must be greater than 0' })
    }

    return zodResolver(z.object(baseSchema))
})

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    const type = props.entryType === 'income' ? 'Income' : 'Expense'
    return `${action} ${type}`
})

const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    if (fileError.value) return
    const values = getSubmitValues(e, formValues)
    const accountId = extractAccountId(values.AccountId)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const amount = Number(values.amount ?? formValues.value.amount ?? 0)
    const stockAmount = values.stockAmount != null ? Number(values.stockAmount) : formValues.value.stockAmount
    const date = values.date ?? formValues.value.date

    if (!description || !(amount > 0) || accountId == null) return

    const entryData = {
        description,
        notes: (formValues.value.notes ?? '').toString(),
        amount,
        date: toDateString(date),
        accountId,
        type: props.entryType,
        categoryId: categoryId.value
    }
    if (stockAmount != null && Number(stockAmount) > 0) {
        entryData.stockAmount = Number(stockAmount)
    }

    backendError.value = ''

    // Capture attachment state before async calls – the props watcher may reset
    // these reactive refs when the query refetches after updateEntry.
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
            await uploadAttachment(savedId, fileToUpload)
            attachmentChanged = true
        }
        if (attachmentChanged) {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }

        emit('update:visible', false)
    } catch (error) {
        backendError.value = getApiErrorMessage(error)
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} ${props.entryType}:`, error)
    }
}

const viewAttachment = () => {
    window.open(getAttachmentUrl(props.entryId), '_blank')
}

// Define the emit for updating visibility
const emit = defineEmits(['update:visible', 'transformToTransfer'])
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
                        v-if="autofocusAmount"
                    />
                    <InputText
                        id="description"
                        v-model="formValues.description"
                        name="description"
                        v-focus
                        v-else
                    />
                    <Message v-if="$form.description?.invalid" severity="error" size="small">
                        {{ $form.description.error?.message }}
                    </Message>
                </div>

                <!-- Amount Field -->
                <div>
                    <label for="amount" class="form-label">Amount</label>
                    <InputNumber
                        ref="amountInputRef"
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

                <!-- Stock Amount Field - only shown for stock accounts -->
                <div v-if="isStocksAccount">
                    <label for="stockAmount" class="form-label">Stock Amount</label>
                    <InputNumber
                        id="stockAmount"
                        v-model="formValues.stockAmount"
                        name="stockAmount"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                    />
                    <Message v-if="$form.stockAmount?.invalid" severity="error" size="small">
                        {{ $form.stockAmount.error?.message }}
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
                        @update:modelValue="handleAccountSelection"
                        :accountTypes="['cash', 'checkin', 'bank', 'savings', 'lent']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId.error?.message }}
                    </Message>
                </div>

                <CategorySelect v-model="categoryId" :type="entryType" />

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
                            :maxSizeBytes="maxAttachmentBytes"
                            @error="fileError = $event"
                        />
                    </div>
                </div>

                <div class="flex justify-content-between gap-3">
                    <Button
                        v-if="isEdit"
                        type="button"
                        label="To Transfer"
                        icon="ti ti-arrows-left-right"
                        severity="secondary"
                        outlined
                        @click="emit('transformToTransfer', {
                            id: entryId,
                            date: formValues.date,
                            description: formValues.description,
                            notes: formValues.notes,
                            amount: formValues.amount,
                            accountId: extractAccountId(formValues.AccountId),
                            entryType: entryType
                        })"
                    />
                    <div v-else></div>
                    <div class="flex gap-3">
                        <Button
                            type="submit"
                            label="Save"
                            icon="ti ti-check"
                            :loading="isCreating || isUpdating"
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
            </div>
        </Form>
    </Dialog>
</template>
