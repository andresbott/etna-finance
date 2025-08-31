<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.js'
import { useAccounts } from '@/composables/useAccounts.js'

import AccountSelector from '@/components/AccountSelector.vue'
import Message from 'primevue/message'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Divider from 'primevue/divider'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries()
const { accounts } = useAccounts()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    targetAmount: { type: Number, default: 0 },
    originAmount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    targetAccountId: { type: Number, default: null },
    originAccountId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

// Convert numeric accountId to {id: true} format for form validation
const getFormattedAccountId = (accountId) => {
    if (accountId === null || accountId === undefined) return null
    return { [accountId]: true }
}

const formValues = ref({
    targetAccountId: getFormattedAccountId(props.targetAccountId),
    originAccountId: getFormattedAccountId(props.originAccountId)
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = {
        ...newProps,
        targetAccountId: getFormattedAccountId(newProps.targetAccountId),
        originAccountId: getFormattedAccountId(newProps.originAccountId)
    }
})

// Helper function to extract numeric ID from {id: true} object
const extractAccountId = (formValue) => {
    if (!formValue) return null

    // Handle numeric ID (for backwards compatibility)
    if (typeof formValue === 'number') return formValue

    // Handle {id: true} format
    if (typeof formValue === 'object') {
        const keys = Object.keys(formValue)
        if (keys.length > 0) {
            return parseInt(keys[0], 10)
        }
    }

    return null
}

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
    // Account validation - handles {id: true} format from AccountSelector
    const accountValidation = z
        .union([z.null(), z.record(z.boolean())])
        .refine((obj) => obj !== null, { message: 'Account must be selected' })

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
    if (!e.valid) return

    // Extract account IDs from the form values
    const formData = { ...e.values }

    // Convert targetAccountId from {id: true} to numeric id
    if (formData.targetAccountId && typeof formData.targetAccountId === 'object') {
        const targetKeys = Object.keys(formData.targetAccountId)
        formData.targetAccountId = targetKeys.length > 0 ? parseInt(targetKeys[0], 10) : null
    }

    // Convert originAccountId from {id: true} to numeric id
    if (formData.originAccountId && typeof formData.originAccountId === 'object') {
        const originKeys = Object.keys(formData.originAccountId)
        formData.originAccountId = originKeys.length > 0 ? parseInt(originKeys[0], 10) : null
    }

    const entryData = {
        ...formData,
        type: 'transfer'
    }

    try {
        if (props.isEdit) {
            await updateEntry({ id: props.entryId, ...entryData })
        } else {
            await createEntry(entryData)
        }
        emit('update:visible', false)
    } catch (error) {
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
        :style="{ width: '700px' }"
    >
        <Form
            v-slot="$form"
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
            @submit="handleSubmit"
        >
            <div class="flex flex-column gap-3">
                <!-- Top section with common fields -->
                <div class="flex flex-column gap-3">
                    <!-- Description Field -->
                    <div>
                        <label for="description" class="form-label">Description</label>
                        <InputText id="description" name="description" v-focus class="w-full" />
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
                            :showIcon="true"
                            dateFormat="dd/mm/yy"
                            class="w-full"
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
                                :account-types="['cash', 'bank']"
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
                                id="originAmount"
                                name="originAmount"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                                class="w-full"
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
                                :account-types="['cash', 'bank']"
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
                                name="targetAmount"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                                class="w-full"
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
<style>
.form-label {
    display: block;
    font-weight: 500;
}
</style>
