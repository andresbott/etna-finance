<script setup>
import { ref, watch, computed, nextTick, onMounted } from 'vue'
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

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries()
const { accounts } = useAccounts()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryType: { type: String, required: true }, // 'expense' or 'income'
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    targetAmount: { type: Number, default: 0 },
    stockAmount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    targetAccountId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

// Convert numeric targetAccountId to {id: true} format for form validation
const getFormattedAccountId = (accountId) => {
    if (accountId === null || accountId === undefined) return null;
    return { [accountId]: true };
}

const formValues = ref({
    targetAccountId: getFormattedAccountId(props.targetAccountId),
    stockAmount: props.stockAmount
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = {
        ...newProps,
        targetAccountId: getFormattedAccountId(newProps.targetAccountId)
    }
})

// Helper function to extract numeric ID from {id: true} object
const extractAccountId = (formValue) => {
    if (!formValue) return null;
    
    // Handle numeric ID (for backwards compatibility)
    if (typeof formValue === 'number') return formValue;
    
    // Handle {id: true} format
    if (typeof formValue === 'object') {
        const keys = Object.keys(formValue);
        if (keys.length > 0) {
            return parseInt(keys[0], 10);
        }
    }
    
    return null;
}

// Track if selected account is of type stocks
const selectedAccount = ref(null)

// Direct handler for account selection changes
const handleAccountSelection = (accountObject) => {
    updateSelectedAccount(accountObject);
}

// Function to update selected account
const updateSelectedAccount = (accountObject) => {
    if (!accountObject || !accounts.value) {
        selectedAccount.value = null
        return
    }

    // Find the account in the accounts structure
    const accountId = extractAccountId(accountObject);
    
    if (isNaN(accountId) || accountId === null) {
        selectedAccount.value = null
        return
    }
    
    // Search through all providers and their accounts
    let found = null
    if (accounts.value) {
        for (const provider of accounts.value) {
            found = provider.accounts.find(acc => acc.id === accountId)
            if (found) {
                break
            }
        }
    }
    
    selectedAccount.value = found
}

// Also keep the watch for reactive updates, with immediate flag
watch(() => formValues.value.targetAccountId, (newValue) => {
    updateSelectedAccount(newValue);
}, { immediate: true })

// Check if selected account is of type "stocks"
const isStocksAccount = computed(() => {
    return selectedAccount.value?.type === 'stocks'
})

// Build the resolver for income/expense entries
const resolver = computed(() => {
    // Account validation - handles {id: true} format from AccountSelector
    const accountValidation = z
        .union([z.null(), z.record(z.boolean())])
        .refine(
            (obj) => obj !== null,
            { message: 'Account must be selected' }
        )

    // Base schema for income and expense entry types
    const baseSchema = {
        description: z.string().min(1, { message: 'Description is required' }),
        date: z.date(),
        targetAmount: z.number().min(0.01, { message: 'Amount must be greater than 0' }),
        targetAccountId: accountValidation
    }

    // Add stockAmount to schema if selected account is of type stocks
    if (isStocksAccount.value) {
        baseSchema.stockAmount = z.number().min(0.01, { message: 'Stock amount must be greater than 0' })
    }

    return zodResolver(z.object(baseSchema))
})

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    const type = props.entryType === 'income' ? 'Income' : 'Expense'
    return `${action} ${type}`
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

    const entryData = {
        ...formData,
        type: props.entryType
    }

    try {
        if (props.isEdit) {
            await updateEntry({ id: props.entryId, ...entryData })
        } else {
            await createEntry(entryData)
        }
        emit('update:visible', false)
    } catch (error) {
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} ${props.entryType}:`, error)
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
                <!-- Description Field -->
                <div>
                    <label for="description" class="form-label">Description</label>
                    <InputText id="description" name="description" v-focus />
                    <Message v-if="$form.description?.invalid" severity="error" size="small">
                        {{ $form.description.error?.message }}
                    </Message>
                </div>

                <!-- Account field -->
                <div>
                    <label for="targetAccountId" class="form-label">Account</label>
                    <AccountSelector 
                        v-model="formValues.targetAccountId" 
                        name="targetAccountId"
                        @update:modelValue="handleAccountSelection" 
                        :accountTypes="['cash']"
                    />
                    <Message v-if="$form.targetAccountId?.invalid" severity="error" size="small">
                        {{ $form.targetAccountId.error?.message }}
                    </Message>
                </div>

                <!-- Amount Field -->
                <div>
                    <label for="targetAmount" class="form-label">Amount</label>
                    <InputNumber
                        id="targetAmount"
                        name="targetAmount"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                    />
                    <Message v-if="$form.targetAmount?.invalid" severity="error" size="small">
                        {{ $form.targetAmount.error?.message }}
                    </Message>
                </div>

                <!-- Stock Amount Field - only shown for stock accounts -->
                <div v-if="isStocksAccount">
                    <label for="stockAmount" class="form-label">Stock Amount</label>
                    <InputNumber
                        id="stockAmount"
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
                        :showIcon="true" 
                        dateFormat="dd/mm/yy" 
                        :showButtonBar="true"
                    />
                    <Message v-if="$form.date?.invalid" severity="error" size="small">
                        {{ $form.date.error?.message }}
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
<style>
.form-label {
    display: block;
    font-weight: 500;
}
</style>
