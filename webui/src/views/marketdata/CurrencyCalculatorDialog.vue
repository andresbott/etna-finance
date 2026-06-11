<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import Button from 'primevue/button'
import type { FXOverviewRow } from '@/composables/useCurrencyRates'

const props = defineProps<{
    visible: boolean
    mainCurrency: string
    rows: FXOverviewRow[]
}>()

const emit = defineEmits<{
    (e: 'update:visible', value: boolean): void
}>()

const dialogVisible = computed({
    get: () => props.visible,
    set: (v) => emit('update:visible', v)
})

const amount = ref(100)
const sliderAmount = computed({
    get: () => Math.min(Math.max(amount.value, 1), 1000),
    set: (v: number) => { amount.value = v }
})
const fromCurrency = ref('')
const toCurrency = ref('')

const currencies = computed(() => {
    const set = new Set<string>()
    if (props.mainCurrency) set.add(props.mainCurrency)
    for (const row of props.rows) set.add(row.currency)
    return [...set].sort()
})

// Auto-select sensible defaults when dialog opens
watch(dialogVisible, (v) => {
    if (v && currencies.value.length >= 2) {
        if (!fromCurrency.value) fromCurrency.value = props.mainCurrency || currencies.value[0]
        if (!toCurrency.value) {
            toCurrency.value = currencies.value.find((c) => c !== fromCurrency.value) ?? currencies.value[1]
        }
    }
})

function swapCurrencies() {
    const tmp = fromCurrency.value
    fromCurrency.value = toCurrency.value
    toCurrency.value = tmp
}

// Units of `currency` per 1 main currency. The main currency itself is 1.
function ratePerMain(currency: string): number | null {
    if (!currency) return null
    if (currency === props.mainCurrency) return 1
    const row = props.rows.find((r) => r.currency === currency)
    if (!row || row.rate == null || row.rate === 0) return null
    return row.rate
}

const lookupResult = computed(() => {
    const from = fromCurrency.value
    const to = toCurrency.value
    if (!from || !to) return null
    if (from === to) return { rate: 1, found: true }
    const fromRate = ratePerMain(from)
    const toRate = ratePerMain(to)
    if (fromRate == null || toRate == null) return { rate: 0, found: false }
    // amount in `from` -> main (divide) -> `to` (multiply)
    return { rate: toRate / fromRate, found: true }
})

const convertedAmount = computed(() => {
    if (!lookupResult.value || !lookupResult.value.found) return null
    return amount.value * lookupResult.value.rate
})
</script>

<template>
    <Dialog
        v-model:visible="dialogVisible"
        header="Currency Calculator"
        modal
        class="entry-dialog"
    >
        <div class="flex flex-column gap-3 py-2">
            <div class="flex align-items-end gap-2">
                <div class="field flex-1">
                    <label for="calc-from">From</label>
                    <Select
                        id="calc-from"
                        v-model="fromCurrency"
                        :options="currencies"
                        placeholder="Select"
                        class="w-full"
                    />
                </div>
                <Button
                    icon="ti ti-arrows-exchange"
                    text
                    rounded
                    severity="secondary"
                    @click="swapCurrencies"
                    class="mb-1"
                />
                <div class="field flex-1">
                    <label for="calc-to">To</label>
                    <Select
                        id="calc-to"
                        v-model="toCurrency"
                        :options="currencies"
                        placeholder="Select"
                        class="w-full"
                    />
                </div>
            </div>

            <div class="field">
                <label for="calc-amount">Amount</label>
                <InputNumber
                    id="calc-amount"
                    v-model="amount"
                    :min="0"
                    :minFractionDigits="0"
                    :maxFractionDigits="2"
                    class="w-full"
                />
            </div>

            <Slider v-model="sliderAmount" :min="1" :max="1000" class="w-full" />

            <div v-if="lookupResult && lookupResult.found" class="result-box">
                <span class="result-amount">{{ convertedAmount!.toFixed(2) }} {{ toCurrency }}</span>
                <span class="result-rate text-color-secondary">
                    1 {{ fromCurrency }} = {{ lookupResult.rate.toFixed(4) }} {{ toCurrency }}
                </span>
            </div>
            <div v-else-if="lookupResult && !lookupResult.found" class="result-box text-color-secondary">
                This pair is not tracked
            </div>
        </div>
    </Dialog>
</template>

<style scoped>
.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}

.result-box {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0.75rem 1rem;
    background: var(--c-surface-100);
    border-radius: var(--c-border-radius);
}

.result-amount {
    font-size: 1.25rem;
    font-weight: 700;
}

.result-rate {
    font-size: 0.85rem;
}
</style>
