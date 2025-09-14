<script setup>
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useAccounts } from '@/composables/useAccounts.js'

const { createAccountProvider, updateAccountProvider, isCreating, isUpdating } = useAccounts()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    providerId: { type: Number, default: null },
    name: { type: String, default: '' },
    description: { type: String, default: '' },
    visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])

const formValues = ref({
    providerId: props.providerId,
    name: props.name,
    description: props.description
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = { ...newProps }
})

const resolver = ref(
    zodResolver(
        z.object({
            name: z.string().min(1, { message: 'Name is required' }),
            description: z.string().optional()
        })
    )
)

const onFormSubmit = async (e) => {
    if (e.valid) {
        const formData = {
            ...e.values,
            providerId: props.providerId
        }

        if (props.isEdit) {
            try {
                await updateAccountProvider({
                    id: formData.providerId,
                    name: formData.name,
                    description: formData.description
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to update account provider:', error)
            }
        } else {
            try {
                await createAccountProvider({
                    name: formData.name,
                    description: formData.description
                })
                emit('update:visible', false)
            } catch (error) {
                console.error('Failed to create account provider:', error)
            }
        }
    }
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="isEdit ? 'Edit Account Provider' : 'Add New Account Provider'"
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
                <div>
                    <label for="name" class="form-label">Description</label>
                    <InputText name="name" placeholder="Provider Name" />
                    <Message v-if="$form.name?.invalid" severity="error" size="small">{{
                        $form.name.error?.message
                    }}</Message>
                </div>
                <div>
                    <label for="description" class="form-label">Description</label>
                    <InputText name="description" placeholder="Description" />
                    <Message v-if="$form.description?.invalid" severity="error" size="small">{{
                        $form.description.error?.message
                    }}</Message>
                </div>
                <div class="flex justify-content-end gap-2">
                    <Button
                        type="submit"
                        :label="isEdit ? 'Update' : 'Create'"
                        :loading="isCreating || isUpdating"
                    />
                    <Button
                        label="Cancel"
                        severity="secondary"
                        text
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
