<script setup>
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useAccounts } from '@/composables/useAccounts.js'

const { createAccount, updateAccount, isCreating, isUpdating } = useAccounts()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    accountId: { type: Number, default: null },
    name: { type: String, default: '' },
    currency: { type: String, default: 'CHF' },
    type: { type: String, default: 'cash' },
    visible: { type: Boolean, default: false },
    providerId: { type: Number, required: true }
})

const emit = defineEmits(['update:visible'])

const currencies = ref(['CHF', 'USD', 'EUR'])
const accountTypes = ref(['cash', 'checkin','savings','investment'])

const formValues = ref({
    name: props.name,
    currency: props.currency,
    type: props.type
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = { ...newProps }
})

const resolver = ref(
    zodResolver(
        z.object({
            name: z.string().min(1, { message: 'Name is required' }),
            currency: z.string().min(1, { message: 'Currency is required' }),
            type: z.string().min(1, { message: 'Type is required' })
        })
    )
)

const onFormSubmit = async (e) => {
    if (e.valid) {
        const formData = {
            ...e.values,
            accountId: props.accountId,
            providerId: props.providerId
        }

        if (props.isEdit) {
            try {
                await updateAccount({
                    id: formData.accountId,
                    name: formData.name,
                    currency: formData.currency,
                    type: formData.type,
                    providerId: formData.providerId
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to update account:', error)
                // Handle error (maybe show a toast notification)
            }
        } else {
            try {
                await createAccount({
                    name: formData.name,
                    currency: formData.currency,
                    type: formData.type,
                    providerId: formData.providerId
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to create account:', error)
                // Handle error (maybe show a toast notification)
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
            :header="isEdit ? 'Edit Account' : 'Add New Account'"
        >
            <Form
                v-slot="$form"
                :resolver="resolver"
                :initialValues="formValues"
                :validateOnValueUpdate="false"
                :validateOnBlur="true"
                @submit="onFormSubmit"
            >
                <div v-focustrap class="flex flex-column gap-3">
                    <!-- Hidden Provider ID Field -->
                    <div>
                        <label for="name" class="form-label">Name</label>
                        <InputText name="name" placeholder="Account Name" />
                        <Message v-if="$form.name?.invalid" severity="error" size="small">{{
                            $form.name.error?.message
                        }}</Message>
                    </div>
                    <div>
                        <label for="currency" class="form-label">Currency</label>
                        <Select
                            :options="currencies"
                            name="currency"
                            placeholder="Select Currency"
                        />
                        <Message v-if="$form.currency?.invalid" severity="error" size="small">{{
                            $form.currency.error?.message
                        }}</Message>
                    </div>
                    <div>
                        <label for="accountTypes" class="form-label">Type</label>
                        <Select
                            :options="accountTypes"
                            name="type"
                            placeholder="Select Account Type"
                        />
                        <Message v-if="$form.type?.invalid" severity="error" size="small">{{
                            $form.type.error?.message
                        }}</Message>
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
    </div>
</template>
<style>
.form-label {
    display: block;
    font-weight: 500;
}
</style>
