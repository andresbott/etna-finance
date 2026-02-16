<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useSettingsStore } from '@/store/settingsStore'

const props = defineProps({
    visible: { type: Boolean, default: false },
    isEdit: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    instrument: {
        type: Object,
        default: () => null
    }
})

const emit = defineEmits(['update:visible', 'save'])

const settingsStore = useSettingsStore()
const currencies = computed(() => {
    const list = settingsStore.currencies.length > 0 ? settingsStore.currencies : ['CHF']
    return list.map(c => ({ label: c, value: c }))
})

const formValues = ref({
    symbol: '',
    name: '',
    currency: 'CHF'
})

watch(
    () => [props.visible, props.instrument],
    ([visible, instrument]) => {
        if (visible) {
            if (instrument) {
                formValues.value = {
                    symbol: instrument.symbol ?? '',
                    name: instrument.name ?? '',
                    currency: instrument.currency ?? 'CHF'
                }
            } else {
                formValues.value = {
                    symbol: '',
                    name: '',
                    currency: 'CHF'
                }
            }
        }
    },
    { immediate: true }
)

const resolver = ref(
    zodResolver(
        z.object({
            symbol: z.string().min(1, { message: 'Symbol is required' }),
            name: z.string().min(1, { message: 'Name is required' }),
            currency: z.string().min(1, { message: 'Currency is required' })
        })
    )
)

const onFormSubmit = (e) => {
    if (e.valid) {
        emit('save', {
            id: props.instrument?.id,
            ...e.values
        })
        emit('update:visible', false)
    }
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="isEdit ? 'Edit investment instrument' : 'Add investment instrument'"
        class="entry-dialog"
    >
        <Form
            :key="`instrument-form-${visible}-${instrument?.id ?? 'new'}`"
            v-slot="$form"
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="true"
            @submit="onFormSubmit"
        >
            <div v-focustrap class="flex flex-column gap-3">
                <div>
                    <label for="symbol" class="form-label">Symbol</label>
                    <InputText
                        id="symbol"
                        name="symbol"
                        placeholder="e.g. AAPL"
                    />
                    <Message v-if="$form.symbol?.invalid" severity="error" size="small">
                        {{ $form.symbol.error?.message }}
                    </Message>
                </div>
                <div>
                    <label for="name" class="form-label">Name</label>
                    <InputText
                        id="name"
                        name="name"
                        placeholder="e.g. Apple Inc."
                    />
                    <Message v-if="$form.name?.invalid" severity="error" size="small">
                        {{ $form.name.error?.message }}
                    </Message>
                </div>
                <div>
                    <label for="currency" class="form-label">Currency</label>
                    <Select
                        id="currency"
                        name="currency"
                        :options="currencies"
                        optionLabel="label"
                        optionValue="value"
                        placeholder="Select currency"
                    />
                    <Message v-if="$form.currency?.invalid" severity="error" size="small">
                        {{ $form.currency.error?.message }}
                    </Message>
                </div>
                <div class="flex justify-content-end gap-3">
                    <Button type="submit" label="Save" icon="pi pi-check" :loading="loading" />
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
