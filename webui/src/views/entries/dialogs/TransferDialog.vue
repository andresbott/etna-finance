<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.ts'
import { useAccounts } from '@/composables/useAccounts'
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
import Divider from 'primevue/divider'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries({})
const backendError = ref('')
const { accounts } = useAccounts()
const { pickerDateFormat } = useDateFormat()

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
    autofocusAmount: { type: Boolean, default: false }
})

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
    date: getDateOnly(props.date),
    targetAmount: props.targetAmount,
    originAmount: props.originAmount,
    targetAccountId: getFormattedAccountId(props.targetAccountId),
    originAccountId: getFormattedAccountId(props.originAccountId)
})

// Watch props to update form values when editing
const formKey = ref(0)
watch(props, (newProps) => {
    formValues.value = {
        description: newProps.description,
        date: getDateOnly(newProps.date),
        targetAmount: newProps.targetAmount,
        originAmount: newProps.originAmount,
        targetAccountId: getFormattedAccountId(newProps.targetAccountId),
        originAccountId: getFormattedAccountId(newProps.originAccountId)
    }
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

    // Search through all providers and their accounts
    let found = null
    if (accounts.value) {
        for (const provider of accounts.value) {
            found = provider.accounts.find((acc) => acc.id === accountId)
            if (found) {
                break
            }
        }
    }

    selectedTargetAccount.value = found
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

    // Search through all providers and their accounts
    let found = null
    if (accounts.value) {
        for (const provider of accounts.value) {
            found = provider.accounts.find((acc) => acc.id === accountId)
            if (found) {
                break
            }
        }
    }

    selectedOriginAccount.value = found
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
        date: z.date(),
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
        date: toDateString(date),
        targetAmount,
        originAmount,
        targetAccountId,
        originAccountId,
        type: 'transfer'
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
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} transfer:`, error)
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
                                :account-types="['cash', 'checkin', 'bank', 'savings']"
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
                        <i class="pi pi-arrow-right text-2xl"></i>
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
                                :account-types="['cash', 'checkin', 'bank', 'savings']"
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

                <!-- Action buttons -->
                <div class="flex justify-content-end gap-3 pt-3">
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
