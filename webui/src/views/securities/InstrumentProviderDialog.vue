<script setup>
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useInstruments } from '@/composables/useInstruments'
import { getApiErrorMessage } from '@/utils/apiError'
import IconSelect from '@/components/common/IconSelect.vue'

const { createInstrumentProvider, updateInstrumentProvider, isCreatingProvider, isUpdatingProvider } =
    useInstruments()
const backendError = ref('')

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    providerId: { type: Number, default: null },
    name: { type: String, default: '' },
    description: { type: String, default: '' },
    icon: { type: String, default: 'pi-chart-bar' },
    visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible'])
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const selectedIcon = ref(props.icon)

const formValues = ref({
    providerId: props.providerId,
    name: props.name,
    description: props.description
})

watch(props, (newProps) => {
    formValues.value = { ...newProps }
    selectedIcon.value = newProps.icon || 'pi-chart-bar'
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
            icon: selectedIcon.value,
            providerId: props.providerId
        }

        backendError.value = ''
        if (props.isEdit) {
            try {
                await updateInstrumentProvider({
                    id: formData.providerId,
                    payload: {
                        name: formData.name,
                        description: formData.description,
                        icon: formData.icon
                    }
                })
                emit('update:visible', false)
            } catch (error) {
                backendError.value = getApiErrorMessage(error)
                console.error('Failed to update instrument provider:', error)
            }
        } else {
            try {
                await createInstrumentProvider({
                    name: formData.name,
                    description: formData.description,
                    icon: formData.icon
                })
                emit('update:visible', false)
            } catch (error) {
                backendError.value = getApiErrorMessage(error)
                console.error('Failed to create instrument provider:', error)
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
        :header="isEdit ? 'Edit Instrument Provider' : 'Add New Instrument Provider'"
        class="entry-dialog"
    >
        <Message v-if="backendError" severity="error" :closable="false" class="mb-2 mx-3 mt-2">{{ backendError }}</Message>
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
                    <label for="name" class="form-label">Name</label>
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
                <div>
                    <label for="icon" class="form-label">Icon</label>
                    <IconSelect v-model="selectedIcon" />
                </div>
                <div class="flex justify-content-end gap-2">
                    <Button
                        type="submit"
                        :label="isEdit ? 'Update' : 'Create'"
                        :loading="isCreatingProvider || isUpdatingProvider"
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
