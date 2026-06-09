<!-- webui/src/components/marketdata/IndicatorSettings.vue -->
<script setup lang="ts">
import { ref } from 'vue'
import Button from 'primevue/button'
import Popover from 'primevue/popover'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import Checkbox from 'primevue/checkbox'

defineProps<{
    type: 'sma' | 'ema' | 'bollinger' | 'rsi' | 'macd'
    params: Record<string, any>
}>()

const emit = defineEmits<{
    (e: 'update:params', value: Record<string, any>): void
}>()

const popoverRef = ref()

function toggle(event: Event) {
    popoverRef.value.toggle(event)
}
</script>

<template>
    <Button
        icon="ti ti-settings"
        text
        size="small"
        severity="secondary"
        @click="toggle"
        class="indicator-settings-btn"
    />
    <Popover ref="popoverRef">
        <div class="indicator-settings">
            <!-- SMA -->
            <template v-if="type === 'sma'">
                <div class="setting-field">
                    <label>Period 1</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period1"
                            @update:modelValue="$emit('update:params', { ...params, period1: $event })"
                            :min="2" :max="500" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period1"
                            @update:modelValue="$emit('update:params', { ...params, period1: $event })"
                            :min="2" :max="500" class="setting-input"
                        />
                    </div>
                </div>
                <div class="setting-row">
                    <Checkbox
                        :modelValue="params.showSecond"
                        @update:modelValue="$emit('update:params', { ...params, showSecond: $event })"
                        :binary="true" inputId="sma-second"
                    />
                    <label for="sma-second">Second line</label>
                </div>
                <div v-if="params.showSecond" class="setting-field">
                    <label>Period 2</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period2"
                            @update:modelValue="$emit('update:params', { ...params, period2: $event })"
                            :min="2" :max="500" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period2"
                            @update:modelValue="$emit('update:params', { ...params, period2: $event })"
                            :min="2" :max="500" class="setting-input"
                        />
                    </div>
                </div>
            </template>

            <!-- EMA -->
            <template v-if="type === 'ema'">
                <div class="setting-field">
                    <label>Period 1</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period1"
                            @update:modelValue="$emit('update:params', { ...params, period1: $event })"
                            :min="2" :max="500" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period1"
                            @update:modelValue="$emit('update:params', { ...params, period1: $event })"
                            :min="2" :max="500" class="setting-input"
                        />
                    </div>
                </div>
                <div class="setting-row">
                    <Checkbox
                        :modelValue="params.showSecond"
                        @update:modelValue="$emit('update:params', { ...params, showSecond: $event })"
                        :binary="true" inputId="ema-second"
                    />
                    <label for="ema-second">Second line</label>
                </div>
                <div v-if="params.showSecond" class="setting-field">
                    <label>Period 2</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period2"
                            @update:modelValue="$emit('update:params', { ...params, period2: $event })"
                            :min="2" :max="500" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period2"
                            @update:modelValue="$emit('update:params', { ...params, period2: $event })"
                            :min="2" :max="500" class="setting-input"
                        />
                    </div>
                </div>
            </template>

            <!-- Bollinger -->
            <template v-if="type === 'bollinger'">
                <div class="setting-field">
                    <label>Period</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period"
                            @update:modelValue="$emit('update:params', { ...params, period: $event })"
                            :min="2" :max="500" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period"
                            @update:modelValue="$emit('update:params', { ...params, period: $event })"
                            :min="2" :max="500" class="setting-input"
                        />
                    </div>
                </div>
                <div class="setting-field">
                    <label>Std Dev</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.stdDev"
                            @update:modelValue="$emit('update:params', { ...params, stdDev: $event })"
                            :min="0.5" :max="5" :step="0.5" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.stdDev"
                            @update:modelValue="$emit('update:params', { ...params, stdDev: $event })"
                            :min="0.5" :max="5" :step="0.5" :minFractionDigits="1" class="setting-input"
                        />
                    </div>
                </div>
            </template>

            <!-- RSI -->
            <template v-if="type === 'rsi'">
                <div class="setting-field">
                    <label>Period</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.period"
                            @update:modelValue="$emit('update:params', { ...params, period: $event })"
                            :min="2" :max="100" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.period"
                            @update:modelValue="$emit('update:params', { ...params, period: $event })"
                            :min="2" :max="100" class="setting-input"
                        />
                    </div>
                </div>
            </template>

            <!-- MACD -->
            <template v-if="type === 'macd'">
                <div class="setting-field">
                    <label>Fast</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.fast"
                            @update:modelValue="$emit('update:params', { ...params, fast: $event })"
                            :min="2" :max="100" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.fast"
                            @update:modelValue="$emit('update:params', { ...params, fast: $event })"
                            :min="2" :max="100" class="setting-input"
                        />
                    </div>
                </div>
                <div class="setting-field">
                    <label>Slow</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.slow"
                            @update:modelValue="$emit('update:params', { ...params, slow: $event })"
                            :min="2" :max="200" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.slow"
                            @update:modelValue="$emit('update:params', { ...params, slow: $event })"
                            :min="2" :max="200" class="setting-input"
                        />
                    </div>
                </div>
                <div class="setting-field">
                    <label>Signal</label>
                    <div class="slider-input-row">
                        <Slider
                            :modelValue="params.signal"
                            @update:modelValue="$emit('update:params', { ...params, signal: $event })"
                            :min="2" :max="100" class="setting-slider"
                        />
                        <InputNumber
                            :modelValue="params.signal"
                            @update:modelValue="$emit('update:params', { ...params, signal: $event })"
                            :min="2" :max="100" class="setting-input"
                        />
                    </div>
                </div>
            </template>
        </div>
    </Popover>
</template>

<style scoped>
.indicator-settings-btn {
    padding: 0.15rem;
}

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

.setting-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.setting-row label {
    font-size: 0.85rem;
    font-weight: 500;
}
</style>
