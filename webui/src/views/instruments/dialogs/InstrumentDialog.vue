<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useSettingsStore } from '@/store/settingsStore'
import { lookupInstrument, LookupRateLimitError } from '@/lib/api/Instrument'

const OTHER = '__other__'

const TYPE_VALUES = ['Stock', 'ETF', 'REIT', 'Bond']
const TYPE_OPTIONS = [
    ...TYPE_VALUES.map(v => ({ label: v, value: v })),
    { label: 'Other', value: OTHER }
]

const EXCHANGE_VALUES = [
    'NYSE', 'NASDAQ', 'LSE', 'TSX', 'Euronext', 'XETRA', 'SIX',
    'JPX (Tokyo)', 'HKEX', 'ASX', 'BME (Madrid)', 'Borsa Italiana', 'Nasdaq Nordic'
]
const EXCHANGE_OPTIONS = [
    ...EXCHANGE_VALUES.map(v => ({ label: v, value: v })),
    { label: 'Other', value: OTHER }
]

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
const defaultCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
const currencies = computed(() => {
    const list = settingsStore.currencies.length > 0 ? settingsStore.currencies : [defaultCurrency.value]
    return list.map(c => ({ label: c, value: c }))
})

const formValues = ref({
    symbol: '',
    name: '',
    currency: '',
    notes: '',
    type: '',
    typeOther: '',
    exchange: '',
    exchangeOther: ''
})

// Bumping this nonce re-keys the Form so it re-initializes from formValues (used by autofill).
const formNonce = ref(0)
const autofilling = ref(false)
const autofillMessage = ref('')

// Split a stored value into a {select, other} pair: known values select directly, unknown
// values select "Other" and go into the free-text field.
const splitKnown = (value, known) => {
    if (!value) return { select: '', other: '' }
    return known.includes(value)
        ? { select: value, other: '' }
        : { select: OTHER, other: value }
}

const buildFormValues = (instrument) => {
    const currencyDefault = defaultCurrency.value
    if (!instrument) {
        return {
            symbol: '', name: '', currency: currencyDefault, notes: '',
            type: '', typeOther: '', exchange: '', exchangeOther: ''
        }
    }
    const t = splitKnown(instrument.type ?? '', TYPE_VALUES)
    const e = splitKnown(instrument.exchange ?? '', EXCHANGE_VALUES)
    return {
        symbol: instrument.symbol ?? '',
        name: instrument.name ?? '',
        currency: instrument.currency ?? currencyDefault,
        notes: instrument.notes ?? '',
        type: t.select,
        typeOther: t.other,
        exchange: e.select,
        exchangeOther: e.other
    }
}

watch(
    () => [props.visible, props.instrument],
    ([visible, instrument]) => {
        if (visible) {
            formValues.value = buildFormValues(instrument)
            formNonce.value++
        }
    },
    { immediate: true }
)

const resolver = zodResolver(
        z
            .object({
                symbol: z.string().min(1, { message: 'Symbol is required' }),
                name: z.string().min(1, { message: 'Name is required' }),
                currency: z.string().min(1, { message: 'Currency is required' }),
                type: z.string().min(1, { message: 'Type is required' }),
                typeOther: z.string().optional(),
                exchange: z.string().min(1, { message: 'Exchange is required' }),
                exchangeOther: z.string().optional(),
                notes: z.string().optional()
            })
            .refine(d => d.type !== OTHER || (d.typeOther?.trim().length ?? 0) > 0, {
                message: 'Please specify the type',
                path: ['typeOther']
            })
            .refine(d => d.exchange !== OTHER || (d.exchangeOther?.trim().length ?? 0) > 0, {
                message: 'Please specify the exchange',
                path: ['exchangeOther']
            })
    )

const onAutofill = async (symbol) => {
    const sym = (symbol ?? '').trim()
    if (!sym || autofilling.value) return
    autofilling.value = true
    autofillMessage.value = ''
    try {
        const data = await lookupInstrument(sym)
        if (!data) return // 204 -> null: silently do nothing
        const t = splitKnown(data.type ?? '', TYPE_VALUES)
        const e = splitKnown(data.exchange ?? '', EXCHANGE_VALUES)
        formValues.value = {
            symbol: sym,
            name: data.name ?? '',
            currency: data.currency || formValues.value.currency,
            notes: data.notes ?? '',
            type: t.select,
            typeOther: t.other,
            exchange: e.select,
            exchangeOther: e.other
        }
        formNonce.value++
    } catch (e) {
        if (e instanceof LookupRateLimitError) {
            autofillMessage.value = e.retryAfterSeconds
                ? `Rate limited by the data provider. Try again in ~${e.retryAfterSeconds}s.`
                : 'Rate limited by the data provider. Try again in a moment.'
        }
        // other errors: silently do nothing
    } finally {
        autofilling.value = false
    }
}

const onFormSubmit = (e) => {
    if (!e.valid) return
    const v = e.values
    const type = v.type === OTHER ? v.typeOther.trim() : v.type
    const exchange = v.exchange === OTHER ? v.exchangeOther.trim() : v.exchange
    emit('save', {
        id: props.instrument?.id,
        symbol: v.symbol,
        name: v.name,
        currency: v.currency,
        notes: v.notes ?? '',
        type,
        exchange
    })
    emit('update:visible', false)
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
            :key="`instrument-form-${visible}-${instrument?.id ?? 'new'}-${formNonce}`"
            v-slot="$form"
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="true"
            @submit="onFormSubmit"
        >
            <div v-focustrap class="flex flex-column gap-3">
                <div>
                    <label for="symbol" class="form-label">
                        Symbol
                        <i
                            v-if="isEdit"
                            class="ti ti-help-circle ml-1"
                            v-tooltip.top="'The symbol cannot be changed after creation.'"
                            aria-label="The symbol cannot be changed after creation."
                        />
                    </label>
                    <div class="flex gap-2">
                        <InputText
                            id="symbol"
                            name="symbol"
                            class="flex-1"
                            placeholder="e.g. AAPL"
                            :disabled="isEdit"
                        />
                        <Button
                            type="button"
                            icon="ti ti-wand"
                            severity="secondary"
                            :loading="autofilling"
                            v-tooltip.bottom="isEdit ? 'Regenerate data from symbol' : 'Autofill from symbol'"
                            :aria-label="isEdit ? 'Regenerate data from symbol' : 'Autofill from symbol'"
                            @click="onAutofill($form.symbol?.value || formValues.symbol)"
                        />
                    </div>
                    <Message v-if="$form.symbol?.invalid" severity="error" size="small">
                        {{ $form.symbol.error?.message }}
                    </Message>
                    <Message v-if="autofillMessage" severity="warn" size="small">
                        {{ autofillMessage }}
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
                <div>
                    <label for="type" class="form-label">Type</label>
                    <Select
                        id="type"
                        name="type"
                        :options="TYPE_OPTIONS"
                        optionLabel="label"
                        optionValue="value"
                        placeholder="Select type"
                    />
                    <InputText
                        v-if="$form.type?.value === OTHER"
                        id="typeOther"
                        name="typeOther"
                        class="w-full mt-2"
                        placeholder="Specify type"
                        aria-label="Specify type"
                    />
                    <Message v-if="$form.type?.invalid" severity="error" size="small">
                        {{ $form.type.error?.message }}
                    </Message>
                    <Message v-if="$form.typeOther?.invalid" severity="error" size="small">
                        {{ $form.typeOther.error?.message }}
                    </Message>
                </div>
                <div>
                    <label for="exchange" class="form-label">Exchange</label>
                    <Select
                        id="exchange"
                        name="exchange"
                        :options="EXCHANGE_OPTIONS"
                        optionLabel="label"
                        optionValue="value"
                        placeholder="Select exchange"
                    />
                    <InputText
                        v-if="$form.exchange?.value === OTHER"
                        id="exchangeOther"
                        name="exchangeOther"
                        class="w-full mt-2"
                        placeholder="Specify exchange"
                        aria-label="Specify exchange"
                    />
                    <Message v-if="$form.exchange?.invalid" severity="error" size="small">
                        {{ $form.exchange.error?.message }}
                    </Message>
                    <Message v-if="$form.exchangeOther?.invalid" severity="error" size="small">
                        {{ $form.exchangeOther.error?.message }}
                    </Message>
                </div>
                <div>
                    <label for="notes" class="form-label">Notes</label>
                    <Textarea
                        id="notes"
                        name="notes"
                        rows="3"
                        autoResize
                        class="w-full"
                        placeholder="Optional details about this instrument"
                    />
                    <Message v-if="$form.notes?.invalid" severity="error" size="small">
                        {{ $form.notes.error?.message }}
                    </Message>
                </div>
                <div class="flex justify-content-end gap-3">
                    <Button type="submit" label="Save" icon="ti ti-check" :loading="loading" />
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
</template>
