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
import CategorySelect from '@/components/common/categorySelect.vue'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries({})
const { accounts } = useAccounts()
const { pickerDateFormat } = useDateFormat()

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
    autofocusAmount: { type: Boolean, default: false }
})
const categoryId = ref(props.categoryId)

const amountInputRef = ref(null)

// Watch for visibility and autofocusAmount to focus the amount field
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
    amount: props.amount,
    AccountId: getFormattedAccountId(props.accountId),
    stockAmount: props.stockAmount,
    date: getDateOnly(props.date)
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = {
        description: newProps.description,
        amount: newProps.amount,
        AccountId: getFormattedAccountId(newProps.accountId),
        stockAmount: newProps.stockAmount,
        date: getDateOnly(newProps.date)
    }
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

    // Find the account in the accounts structure
    const accountId = extractAccountId(accountObject)

    if (isNaN(accountId) || accountId === null) {
        selectedAccount.value = null
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

    selectedAccount.value = found
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
        date: z.date(),
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
    const values = getSubmitValues(e, formValues)
    const accountId = extractAccountId(values.AccountId)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const amount = Number(values.amount ?? formValues.value.amount ?? 0)
    const stockAmount = values.stockAmount != null ? Number(values.stockAmount) : formValues.value.stockAmount
    const date = values.date ?? formValues.value.date

    if (!description || !(amount > 0) || accountId == null) return

    const entryData = {
        description,
        amount,
        date: toDateString(date),
        accountId,
        type: props.entryType,
        categoryId: categoryId.value
    }
    if (stockAmount != null && Number(stockAmount) > 0) {
        entryData.stockAmount = Number(stockAmount)
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
        class="entry-dialog"
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
                        :accountTypes="['cash', 'checkin', 'bank', 'savings']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId.error?.message }}
                    </Message>
                </div>

                <CategorySelect v-model="categoryId" :type="entryType" />

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
