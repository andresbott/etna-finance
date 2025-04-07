<script setup>
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Calendar from 'primevue/calendar'
import InputNumber from 'primevue/inputnumber'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.js'
import { useAccounts } from '@/composables/useAccounts.js'
// import { useCategories } from '@/composables/useCategories.js'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries()
const { accounts } = useAccounts()
// const { categories } = useCategories()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 },
    stockAmount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    type: { type: String, default: 'expense' },
    targetAccountId: { type: Number, default: null },
    originAccountId: { type: Number, default: null },
    categoryId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])

const entryTypes = ref([
    { label: 'Income', value: 'income' },
    { label: 'Expense', value: 'expense' },
    { label: 'Transfer', value: 'transfer' },
    { label: 'Buy Stock', value: 'buystock' },
    { label: 'Sell Stock', value: 'sellstock' }
])

const formValues = ref({
    entryId: props.entryId,
    description: props.description,
    amount: props.amount,
    stockAmount: props.stockAmount,
    date: props.date,
    type: props.type,
    targetAccountId: props.targetAccountId,
    originAccountId: props.originAccountId,
    categoryId: props.categoryId
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
            type: z.string().min(1, { message: 'Type is required' }),
            targetAccountId: z.number().min(1, { message: 'Target account is required' }),
            categoryId: z.number().min(1, { message: 'Category is required' })
        })
    )
)

const onFormSubmit = async (e) => {
    if (e.valid) {
        const formData = {
            ...e.values,
            entryId: props.entryId
        }

        if (props.isEdit) {
            try {
                await updateEntry({
                    id: formData.entryId,
                    description: formData.description,
                    amount: formData.amount,
                    stockAmount: formData.stockAmount,
                    date: formData.date,
                    type: formData.type,
                    targetAccountId: formData.targetAccountId,
                    originAccountId: formData.originAccountId,
                    categoryId: formData.categoryId
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to update entry:', error)
            }
        } else {
            try {
                await createEntry({
                    description: formData.description,
                    amount: formData.amount,
                    stockAmount: formData.stockAmount,
                    date: formData.date,
                    type: formData.type,
                    targetAccountId: formData.targetAccountId,
                    originAccountId: formData.originAccountId,
                    categoryId: formData.categoryId
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to create entry:', error)
            }
        }
    }
}
</script>

<template>
    <div>
        <Dialog
            :visible="visible"
            @update:visible="$emit('update:visible', $event)"
            :draggable="false"
            modal
            :header="isEdit ? 'Edit Entry' : 'Add New Entry'"
        >
            <Form
                v-slot="$form"
                :resolver="resolver"
                :initialValues="formValues"
                :validateOnValueUpdate="false"
                :validateOnBlur="true"
                @submit="onFormSubmit"
            >
                <div v-focustrap class="flex flex-column gap-4">
                    <InputText name="description" placeholder="Description" />
                    <Message v-if="$form.description?.invalid" severity="error" size="small">{{
                        $form.description.error?.message
                    }}</Message>

                    <InputNumber
                        name="amount"
                        placeholder="Amount"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                    />
                    <Message v-if="$form.amount?.invalid" severity="error" size="small">{{
                        $form.amount.error?.message
                    }}</Message>

                    <Calendar name="date" :showIcon="true" />
                    <Message v-if="$form.date?.invalid" severity="error" size="small">{{
                        $form.date.error?.message
                    }}</Message>

                    <Select
                        :options="entryTypes"
                        optionLabel="label"
                        optionValue="value"
                        name="type"
                        placeholder="Select Entry Type"
                    />
                    <Message v-if="$form.type?.invalid" severity="error" size="small">{{
                        $form.type.error?.message
                    }}</Message>

                    <Select
                        :options="accounts"
                        optionLabel="name"
                        optionValue="id"
                        name="targetAccountId"
                        placeholder="Select Target Account"
                    />
                    <Message v-if="$form.targetAccountId?.invalid" severity="error" size="small">{{
                        $form.targetAccountId.error?.message
                    }}</Message>

<!--                    <Select-->
<!--                        :options="categories"-->
<!--                        optionLabel="name"-->
<!--                        optionValue="id"-->
<!--                        name="categoryId"-->
<!--                        placeholder="Select Category"-->
<!--                    />-->
<!--                    <Message v-if="$form.categoryId?.invalid" severity="error" size="small">{{-->
<!--                        $form.categoryId.error?.message-->
<!--                    }}</Message>-->

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
    </div>
</template> 