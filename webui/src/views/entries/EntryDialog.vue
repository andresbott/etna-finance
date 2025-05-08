<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Calendar from 'primevue/calendar'
import InputNumber from 'primevue/inputnumber'
import Select from 'primevue/select'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.js'
import { useAccounts } from '@/composables/useAccounts.js'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries()
const { accounts } = useAccounts()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryType: { type: String, required: true }, // 'expense' or 'income'
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    targetAccountId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

const formValues = ref({
    entryId: props.entryId,
    description: props.description,
    amount: props.amount,
    date: props.date,
    targetAccountId: props.targetAccountId
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = { ...newProps }
})

const resolver = ref(
    zodResolver(
        z.object({
            description: z.string().min(1, { message: 'Description is required' }),
            amount: z.number().min(0.01, { message: 'Amount must be greater than 0' }),
            date: z.date(),
            targetAccountId: z.number().min(1, { message: 'Target account is required' })
        })
    )
)

const dialogTitle = computed(() => {
    const action = props.isEdit ? 'Edit' : 'Add New'
    const type = props.entryType === 'income' ? 'Income' : 'Expense'
    return `${action} ${type}`
})

const handleSubmit = async (e) => {
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

const hideDialog = () => {
    emit('update:visible', false)
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
            <div v-focustrap class="flex flex-column gap-4">
                <InputText name="description" placeholder="Description" />
                <Message v-if="$form.description?.invalid" severity="error" size="small">
                    {{ $form.description.error?.message }}
                </Message>

                <InputNumber
                    name="amount"
                    placeholder="Amount"
                    :minFractionDigits="2"
                    :maxFractionDigits="2"
                />
                <Message v-if="$form.amount?.invalid" severity="error" size="small">
                    {{ $form.amount.error?.message }}
                </Message>

                <Calendar name="date" :showIcon="true" dateFormat="dd/mm/yy" />
                <Message v-if="$form.date?.invalid" severity="error" size="small">
                    {{ $form.date.error?.message }}
                </Message>

                <Select
                    :options="accounts"
                    optionLabel="name"
                    optionValue="id"
                    name="targetAccountId"
                    placeholder="Select Target Account"
                />
                <Message v-if="$form.targetAccountId?.invalid" severity="error" size="small">
                    {{ $form.targetAccountId.error?.message }}
                </Message>

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
