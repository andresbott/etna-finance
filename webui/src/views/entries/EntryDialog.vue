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

/**
 * EntryDialog Component
 * 
 * Provides a modal dialog for creating or editing financial entries (income, expense, transfer)
 * Uses PrimeVue Form with Zod validation
 */

// -----------------------------------------------------------------------------
// Composable State & API
// -----------------------------------------------------------------------------
const { createEntry, updateEntry, isCreating, isUpdating } = useEntries()
const { accounts } = useAccounts()

// -----------------------------------------------------------------------------
// Props & Emits
// -----------------------------------------------------------------------------
const props = defineProps({
  isEdit: { type: Boolean, default: false },
  entryType: { type: String, required: true }, // 'expense', 'income', or 'transfer'
  entryId: { type: Number, default: null },
  description: { type: String, default: '' },
  amount: { type: Number, default: 0 },
  date: { type: Date, default: () => new Date() },
  targetAccountId: { type: Number, default: null },
  originAccountId: { type: Number, default: null }, // For transfers only
  visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])

// -----------------------------------------------------------------------------
// Form State
// -----------------------------------------------------------------------------
const formValues = ref({
  targetAccountId: props.targetAccountId
})

// Watch props to update form values when editing
watch(props, (newProps) => {
  formValues.value = { ...newProps }
})

// -----------------------------------------------------------------------------
// Computed Properties
// -----------------------------------------------------------------------------
// Dynamic form title based on entry type and edit mode
const dialogTitle = computed(() => {
  const action = props.isEdit ? 'Edit' : 'Add New'
  
  const typeMap = {
    'income': 'Income',
    'expense': 'Expense',
    'transfer': 'Transfer'
  }
  
  const type = typeMap[props.entryType] || 'Entry'
  return `${action} ${type}`
})

// Dynamic Zod validator based on entry type
const resolver = computed(() => {
  // Account validation - handles {id: true} format from AccountSelector
  const accountValidation = z
    .union([z.null(), z.record(z.boolean())])
    .refine(
      obj => obj !== null,
      { message: "Account must be selected" }
    )

  // Base schema for all entry types
  const baseSchema = {
    description: z.string().min(1, { message: 'Description is required' }),
    amount: z.number().min(0.01, { message: 'Amount must be greater than 0' }),
    date: z.date(),
    targetAccountId: accountValidation,
  }
  
  // TODO: Add originAccountId validation for transfers

  return zodResolver(z.object(baseSchema))
})

// -----------------------------------------------------------------------------
// Event Handlers
// -----------------------------------------------------------------------------
const handleSubmit = async (e) => {
console.log(e)
  if (!e.valid) return

  const entryData = {
    ...e.values,
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

const closeDialog = () => {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="closeDialog"
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
        <div class="form-field">
          <label for="description" class="form-label">Description</label>
          <InputText id="description" name="description" v-focus />
          <Message v-if="$form.description?.invalid" severity="error" size="small">
            {{ $form.description.error?.message }}
          </Message>
        </div>

        <!-- Amount Field -->
        <div class="form-field">
          <label for="amount" class="form-label">Amount</label>
          <InputNumber
            id="amount"
            name="amount"
            :minFractionDigits="2"
            :maxFractionDigits="2"
          />
          <Message v-if="$form.amount?.invalid" severity="error" size="small">
            {{ $form.amount.error?.message }}
          </Message>
        </div>

        <!-- Date Field -->
        <div class="form-field">
          <label for="date" class="form-label">Date</label>
          <DatePicker 
            id="date" 
            name="date" 
            :showIcon="true" 
            dateFormat="dd/mm/yy" 
          />
          <Message v-if="$form.date?.invalid" severity="error" size="small">
            {{ $form.date.error?.message }}
          </Message>
        </div>

        <!-- Origin Account (transfers only) -->
        <div class="form-field" v-if="props.entryType === 'transfer'">
          <label for="originAccountId" class="form-label">Origin Account</label>
          <AccountSelector 
            v-model="formValues.originAccountId" 
            name="originAccountId" 
          />
          <Message v-if="$form.originAccountId?.invalid" severity="error" size="small">
            {{ $form.originAccountId.error?.message }}
          </Message>
        </div>

        <!-- Target Account (all entry types) -->
        <div class="form-field">
          <label for="targetAccountId" class="form-label">
            {{ props.entryType === 'transfer' ? 'Target Account' : 'Account' }}
          </label>
          <AccountSelector 
            v-model="formValues.targetAccountId" 
            name="targetAccountId" 
          />
          <Message v-if="$form.targetAccountId?.invalid" severity="error" size="small">
            {{ $form.targetAccountId.error?.message }}
          </Message>
        </div>

        <!-- Form Actions -->
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
            @click="closeDialog"
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

.form-field {
  margin-bottom: 0.5rem;
}
</style>
