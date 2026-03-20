<script setup>
import { ref, watch, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useAccounts } from '@/composables/useAccounts'
import { getApiErrorMessage } from '@/utils/apiError'
import IconSelect from '@/components/common/IconSelect.vue'
import { ACCOUNT_TYPES, ACCOUNT_TYPE_LABELS, getAccountTypeLabel } from '@/types/account'
import { useSettingsStore } from '@/store/settingsStore'
import { getProfiles } from '@/lib/api/CsvImport'

const { createAccount, updateAccount, isCreating, isUpdating } = useAccounts()
const settingsStore = useSettingsStore()
const backendError = ref('')

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    accountId: { type: Number, default: null },
    name: { type: String, default: '' },
    currency: { type: String, default: 'CHF' },
    type: { type: String, default: 'cash' },
    icon: { type: String, default: 'wallet' },
    importProfileId: { type: Number, default: 0 },
    visible: { type: Boolean, default: false },
    providerId: { type: Number, required: true }
})

const emit = defineEmits(['update:visible'])
watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const currencies = computed(() => settingsStore.currencies.length > 0 ? settingsStore.currencies : ['CHF'])

const allAccountTypeOptions = [
    { value: ACCOUNT_TYPES.CASH, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.CASH] },
    { value: ACCOUNT_TYPES.CHECKING, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.CHECKING] },
    { value: ACCOUNT_TYPES.SAVINGS, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.SAVINGS] },
    { value: ACCOUNT_TYPES.INVESTMENT, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.INVESTMENT] },
    { value: ACCOUNT_TYPES.UNVESTED, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.UNVESTED] },
    { value: ACCOUNT_TYPES.LENT, label: ACCOUNT_TYPE_LABELS[ACCOUNT_TYPES.LENT] },
]

const instrumentAccountTypes = [ACCOUNT_TYPES.INVESTMENT, ACCOUNT_TYPES.UNVESTED]

const accountTypeOptions = computed(() => {
    if (settingsStore.instruments) return allAccountTypeOptions
    return allAccountTypeOptions.filter(opt => !instrumentAccountTypes.includes(opt.value))
})

// Separate ref for icon since it's not managed by the Form
const selectedIcon = ref(props.icon)

// Import profiles
const profiles = ref([])
const selectedProfileId = ref(props.importProfileId)

const profileOptions = computed(() => [
    { label: 'None', value: 0 },
    ...profiles.value.map(p => ({ label: p.name, value: p.id }))
])

onMounted(async () => {
    try {
        profiles.value = await getProfiles()
    } catch (e) {
        /* ignore - profiles are optional */
    }
})

const formValues = ref({
    name: props.name,
    currency: props.currency,
    type: props.type
})

// Watch props to update form values when editing
watch(props, (newProps) => {
    formValues.value = { name: newProps.name, currency: newProps.currency, type: newProps.type }
    selectedIcon.value = newProps.icon || 'wallet'
    selectedProfileId.value = newProps.importProfileId || 0
})

const resolver = computed(() =>
    zodResolver(
        z.object({
            name: z.string().min(1, { message: 'Name is required' }),
            type: z.string().min(1, { message: 'Type is required' }),
            currency: z.string().min(1, { message: 'Currency is required' })
        })
    )
)

const onFormSubmit = async (e) => {
    if (e.valid) {
        const formData = {
            ...e.values,
            icon: selectedIcon.value,
            accountId: props.accountId,
            providerId: props.providerId,
            importProfileId: selectedProfileId.value
        }

        backendError.value = ''
        if (props.isEdit) {
            try {
                await updateAccount({
                    id: formData.accountId,
                    name: formData.name,
                    currency: formData.currency,
                    type: formData.type,
                    icon: formData.icon,
                    providerId: formData.providerId,
                    importProfileId: formData.importProfileId
                })
                emit('update:visible', false)
            } catch (error) {
                backendError.value = getApiErrorMessage(error)
                console.error('Failed to update account:', error)
            }
        } else {
            try {
                await createAccount({
                    name: formData.name,
                    currency: formData.currency,
                    type: formData.type,
                    icon: formData.icon,
                    providerId: formData.providerId,
                    importProfileId: formData.importProfileId
                })
                emit('update:visible', false)
            } catch (error) {
                backendError.value = getApiErrorMessage(error)
                console.error('Failed to create account:', error)
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
                        <InputText name="name" placeholder="Account Name" />
                        <Message v-if="$form.name?.invalid" severity="error" size="small">{{
                            $form.name.error?.message
                        }}</Message>
                    </div>
                    <div>
                        <label for="icon" class="form-label">Icon</label>
                        <IconSelect v-model="selectedIcon" />
                    </div>
                    <div>
                        <label for="accountTypes" class="form-label">Type</label>
                        <Select
                            v-model="formValues.type"
                            :options="accountTypeOptions"
                            optionLabel="label"
                            optionValue="value"
                            name="type"
                            placeholder="Select Account Type"
                            scrollHeight="22rem"
                        >
                            <template #value="{ value, placeholder }">
                                <span v-if="value != null && value !== ''">{{ getAccountTypeLabel(value) }}</span>
                                <span v-else class="text-color-secondary">{{ placeholder || 'Select Account Type' }}</span>
                            </template>
                            <template #option="slotProps">
                                {{ slotProps.option?.label ?? getAccountTypeLabel(slotProps.option) }}
                            </template>
                        </Select>
                        <Message v-if="$form.type?.invalid" severity="error" size="small">{{
                            $form.type.error?.message
                        }}</Message>
                    </div>
                    <div>
                        <label for="currency" class="form-label">Currency</label>
                        <Select
                            :options="currencies"
                            name="currency"
                            placeholder="Select Currency"
                            scrollHeight="22rem"
                        />
                        <Message v-if="$form.currency?.invalid" severity="error" size="small">{{
                            $form.currency.error?.message
                        }}</Message>
                    </div>
                    <div>
                        <label class="form-label">Import Profile</label>
                        <Select
                            v-model="selectedProfileId"
                            :options="profileOptions"
                            optionLabel="label"
                            optionValue="value"
                            placeholder="No import profile"
                        />
                    </div>

                    <div class="flex justify-content-end gap-3">
                        <Button
                            type="submit"
                            label="Save"
                            icon="ti ti-check"
                            :loading="isCreating || isUpdating"
                        />
                        <Button
                            type="button"
                            label="Cancel"
                            icon="ti ti-x"
                            severity="secondary"
                            @click="$emit('update:visible', false)"
                        />
                    </div>
                </div>
            </Form>
        </Dialog>
    </div>
</template>
