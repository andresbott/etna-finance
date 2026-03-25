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
import Divider from 'primevue/divider'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'
import { uploadAttachment, deleteAttachment, getAttachmentUrl } from '@/lib/api/Attachment'
import FileInput from '@/components/common/FileInput.vue'
import { useSettingsStore } from '@/store/settingsStore'

const queryClient = useQueryClient()
const { createEntry, updateEntry, deleteEntry, isCreating, isUpdating } = useEntryMutations()
const backendError = ref('')
const { accounts } = useAccounts()
const { pickerDateFormat, dateValidation } = useDateFormat()
const settingsStore = useSettingsStore()
const maxAttachmentBytes = computed(() => settingsStore.maxAttachmentSizeMB * 1024 * 1024)

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    targetAmount: { type: Number, default: 0 },
    originAmount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    targetAccountId: { type: Number, default: null },
    originAccountId: { type: Number, default: null },
    visible: { type: Boolean, default: false },
    autofocusAmount: { type: Boolean, default: false },
    attachmentId: { type: Number, default: null },
    notes: { type: String, default: '' },
    deleteAfterCreateId: { type: Number, default: null }
})

const selectedFile = ref(null)
const existingAttachmentId = ref(null)
const attachmentPendingDelete = ref(false)
const fileError = ref('')

const originAmountInputRef = ref(null)

// Watch for visibility and autofocusAmount to focus the origin amount field
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })
watch(() => [props.visible, props.autofocusAmount], ([visible, autofocus]) => {
    if (visible && autofocus) {
        // Use a longer delay to ensure dialog is fully ready
        setTimeout(() => {
            const inputElement = originAmountInputRef.value?.$el?.querySelector('input')
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
    date: getDateOnly(props.date),
    targetAmount: props.targetAmount,
    originAmount: props.originAmount,
    targetAccountId: getFormattedAccountId(props.targetAccountId),
    originAccountId: getFormattedAccountId(props.originAccountId)
})

// Reset form when dialog opens
const formKey = ref(0)
watch(() => props.visible, (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description,
        notes: props.notes,
        date: getDateOnly(props.date),
        targetAmount: props.targetAmount,
        originAmount: props.originAmount,
        targetAccountId: getFormattedAccountId(props.targetAccountId),
        originAccountId: getFormattedAccountId(props.originAccountId)
    }
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    fileError.value = ''
    formKey.value++
})

// Track selected accounts
const selectedTargetAccount = ref(null)
const selectedOriginAccount = ref(null)

// Update selected target account when it changes
const updateSelectedTargetAccount = (accountObject) => {
    if (!accountObject || !accounts.value) {
        selectedTargetAccount.value = null
        return
    }

    const accountId = extractAccountId(accountObject)

    if (isNaN(accountId) || accountId === null) {
        selectedTargetAccount.value = null
        return
    }

    selectedTargetAccount.value = findAccountById(accounts.value, accountId)
}

// Update selected origin account when it changes
const updateSelectedOriginAccount = (accountObject) => {
    if (!accountObject || !accounts.value) {
        selectedOriginAccount.value = null
        return
    }

    const accountId = extractAccountId(accountObject)

    if (isNaN(accountId) || accountId === null) {
        selectedOriginAccount.value = null
        return
    }

    selectedOriginAccount.value = findAccountById(accounts.value, accountId)
}

// Handle account selection changes
const handleTargetAccountSelection = (accountObject) => {
    updateSelectedTargetAccount(accountObject)
}

const handleOriginAccountSelection = (accountObject) => {
    updateSelectedOriginAccount(accountObject)
}

// Watch for account changes from v-model
watch(
    () => formValues.value.targetAccountId,
    (newValue) => {
        updateSelectedTargetAccount(newValue)
    },
    { immediate: true }
)

watch(
    () => formValues.value.originAccountId,
    (newValue) => {
        updateSelectedOriginAccount(newValue)
    },
    { immediate: true }
)

// Build the resolver for transfer entries
const resolver = computed(() => {
    // Schema for transfers between cash/bank accounts
    const baseSchema = {
        description: z.string().min(1, { message: 'Description is required' }),
        date: dateValidation.value,
        targetAmount: z.number().min(0.01, { message: 'Target amount must be greater than 0' }),
        targetAccountId: accountValidation,
        originAmount: z.number().min(0.01, { message: 'Origin amount must be greater than 0' }),
        originAccountId: accountValidation
    }

    return zodResolver(z.object(baseSchema))
})

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    return `${action} Transfer`
})

const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    if (fileError.value) return
    const values = getSubmitValues(e, formValues)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const targetAmount = Number(values.targetAmount ?? formValues.value.targetAmount ?? 0)
    const originAmount = Number(values.originAmount ?? formValues.value.originAmount ?? 0)
    const targetAccountId = extractAccountId(values.targetAccountId ?? formValues.value.targetAccountId)
    const originAccountId = extractAccountId(values.originAccountId ?? formValues.value.originAccountId)
    const date = values.date ?? formValues.value.date

    if (!description || !(targetAmount > 0) || !(originAmount > 0) || targetAccountId == null || originAccountId == null) return

    const entryData = {
        description,
        notes: (formValues.value.notes ?? '').toString(),
        date: toDateString(date),
        targetAmount,
        originAmount,
        targetAccountId,
        originAccountId,
        type: 'transfer'
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

        // If this was a transform from income/expense, delete the original entry
        if (props.deleteAfterCreateId != null) {
            try {
                await deleteEntry(String(props.deleteAfterCreateId))
            } catch (e) {
                console.error('Transfer created but failed to delete original entry:', e)
                backendError.value = 'Transfer created, but the original entry could not be deleted. Please delete it manually.'
                return // Keep dialog open so user sees the warning
            }
        }

        emit('update:visible', false)
    } catch (error) {
        backendError.value = getApiErrorMessage(error)
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} transfer:`, error)
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
            <div class="flex flex-column gap-3">
                <!-- Top section with common fields -->
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

                <!-- Separator -->
                <Divider />

                <!-- Bottom section with side-by-side fields -->
                <div class="flex flex-row">
                    <!-- Origin section (left side) -->
                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <h3 class="m-0 text-lg font-medium">From</h3>

                        <!-- Origin Account field -->
                        <div>
                            <label for="originAccountId" class="form-label">Origin Account</label>
                            <AccountSelector
                                v-model="formValues.originAccountId"
                                name="originAccountId"
                                @update:modelValue="handleOriginAccountSelection"
                                :account-types="['cash', 'checkin', 'bank', 'savings', 'lent', 'pension', 'prepaidexpense']"
                            />
                            <Message
                                v-if="$form.originAccountId?.invalid"
                                severity="error"
                                size="small"
                            >
                                {{ $form.originAccountId.error?.message }}
                            </Message>
                        </div>

                        <!-- Origin Amount Field -->
                        <div>
                            <label for="originAmount" class="form-label">Origin Amount</label>
                            <InputNumber
                                ref="originAmountInputRef"
                                id="originAmount"
                                v-model="formValues.originAmount"
                                name="originAmount"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                            />
                            <Message
                                v-if="$form.originAmount?.invalid"
                                severity="error"
                                size="small"
                            >
                                {{ $form.originAmount.error?.message }}
                            </Message>
                        </div>
                    </div>

                    <!-- Arrow between sections -->
                    <div class="flex align-items-center justify-content-center px-2">
                        <i class="ti ti-arrow-right text-2xl"></i>
                    </div>

                    <!-- Target section (right side) -->
                    <div class="flex flex-column gap-3 flex-1 p-2">
                        <h3 class="m-0 text-lg font-medium">To</h3>

                        <!-- Target Account field -->
                        <div>
                            <label for="targetAccountId" class="form-label">Target Account</label>
                            <AccountSelector
                                v-model="formValues.targetAccountId"
                                name="targetAccountId"
                                @update:modelValue="handleTargetAccountSelection"
                                :account-types="['cash', 'checkin', 'bank', 'savings', 'lent', 'pension', 'prepaidexpense']"
                            />
                            <Message
                                v-if="$form.targetAccountId?.invalid"
                                severity="error"
                                size="small"
                            >
                                {{ $form.targetAccountId.error?.message }}
                            </Message>
                        </div>

                        <!-- Target Amount Field -->
                        <div>
                            <label for="targetAmount" class="form-label">Target Amount</label>
                            <InputNumber
                                id="targetAmount"
                                v-model="formValues.targetAmount"
                                name="targetAmount"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                            />
                            <Message
                                v-if="$form.targetAmount?.invalid"
                                severity="error"
                                size="small"
                            >
                                {{ $form.targetAmount.error?.message }}
                            </Message>
                        </div>
                    </div>
                </div>

                <!-- Attachment -->
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

                <!-- Action buttons -->
                <div class="flex justify-content-end gap-3 pt-3">
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
        </Form>
    </Dialog>
</template>
