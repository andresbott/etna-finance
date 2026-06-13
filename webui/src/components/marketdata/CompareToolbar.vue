<!-- webui/src/components/marketdata/CompareToolbar.vue -->
<script setup lang="ts">
import { ref, computed } from 'vue'
import SelectButton from 'primevue/selectbutton'
import Button from 'primevue/button'
import Popover from 'primevue/popover'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import type { PriceHistoryRange } from '@/utils/dateRange'
import type { CompareView } from '@/composables/useCompareChart'

const props = defineProps<{
    view: CompareView
    period: number
    range: PriceHistoryRange
}>()

const emit = defineEmits<{
    'update:view': [CompareView]
    'update:period': [number]
    'update:range': [PriceHistoryRange]
}>()

const rangeOptions: { value: PriceHistoryRange; label: string }[] = [
    { value: '1m', label: '1M' },
    { value: '3m', label: '3M' },
    { value: '6m', label: '6M' },
    { value: '1y', label: '1Y' },
    { value: 'max', label: 'ALL' }
]

const viewOptions: { value: CompareView; label: string }[] = [
    { value: 'price', label: 'Price' },
    { value: 'sma', label: 'SMA' },
    { value: 'ema', label: 'EMA' },
    { value: 'rsi', label: 'RSI' }
]

// Match the single-instrument Chart tab's per-indicator period ranges.
const periodMax = computed(() => (props.view === 'rsi' ? 100 : 500))
const settingsLabel = computed(() =>
    viewOptions.find((o) => o.value === props.view)?.label ?? ''
)

const popoverRef = ref()
function toggleSettings(event: Event) {
    popoverRef.value.toggle(event)
}
</script>

<template>
    <div class="compare-toolbar">
        <SelectButton
            :modelValue="range"
            :options="rangeOptions"
            optionLabel="label"
            optionValue="value"
            :allowEmpty="false"
            @update:modelValue="(v: PriceHistoryRange) => v && emit('update:range', v)"
        />
        <SelectButton
            :modelValue="view"
            :options="viewOptions"
            optionLabel="label"
            optionValue="value"
            :allowEmpty="false"
            @update:modelValue="(v: CompareView) => v && emit('update:view', v)"
        />

        <!-- Period lives behind a gear popup, matching the single-instrument Chart tab. -->
        <Button
            v-if="view !== 'price'"
            icon="ti ti-settings"
            text
            size="small"
            severity="secondary"
            class="settings-btn"
            aria-label="Indicator settings"
            @click="toggleSettings"
        />
        <Popover ref="popoverRef">
            <div class="indicator-settings">
                <div class="setting-field">
                    <label>{{ settingsLabel }} Period</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="period"
                            :min="2"
                            :max="periodMax"
                            class="setting-slider"
                            @update:modelValue="(v) => emit('update:period', Array.isArray(v) ? v[0] : v)"
                        />
                        <InputNumber
                            :modelValue="period"
                            :min="2"
                            :max="periodMax"
                            class="setting-input"
                            @update:modelValue="(v: number) => emit('update:period', v ?? 2)"
                        />
                    </div>
                </div>
            </div>
        </Popover>
    </div>
</template>

<style scoped>
.compare-toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
    margin-bottom: 0.75rem;
    flex-shrink: 0;
}

.settings-btn {
    padding: 0.15rem;
}

/* Mirrors IndicatorSettings.vue so the popup looks identical to the Chart tab. */
.indicator-settings {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    min-width: 260px;
    padding: 0.25rem;
}

.setting-field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
}

.setting-field > label {
    font-size: 0.85rem;
    font-weight: 500;
}

.slider-input-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

.setting-slider {
    flex: 1;
}

.setting-input {
    width: 72px;
    flex-shrink: 0;
}

.setting-input :deep(.p-inputnumber-input) {
    width: 72px;
    padding: 0.35rem 0.5rem;
    font-size: 0.85rem;
}
</style>
