<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import InputSwitch from 'primevue/inputswitch'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import Message from 'primevue/message'

const leftSidebarCollapsed = ref(true)

// Placeholder configuration sections
const generalSettings = [
    { label: 'Date Format', value: 'DD/MM/YYYY', icon: 'pi pi-calendar' },
    { label: 'Language', value: 'English', icon: 'pi pi-globe' }
]

// Currency options
const availableCurrencies = ref([
    { name: 'Swiss Franc', code: 'CHF' },
    { name: 'US Dollar', code: 'USD' },
    { name: 'Euro', code: 'EUR' },
    { name: 'British Pound', code: 'GBP' },
    { name: 'Japanese Yen', code: 'JPY' },
    { name: 'Canadian Dollar', code: 'CAD' },
    { name: 'Australian Dollar', code: 'AUD' },
    { name: 'Chinese Yuan', code: 'CNY' },
    { name: 'Indian Rupee', code: 'INR' },
    { name: 'Brazilian Real', code: 'BRL' }
])

// Currency settings
const mainCurrency = ref('CHF')
const allowMultipleCurrencies = ref(true)
const selectedCurrencies = ref(['CHF', 'USD', 'EUR'])

// Feature toggles
const enableStockFunctions = ref(false)
const enableMortgageFunctions = ref(false)
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Mock UI Warning -->
                <Message severity="error" :closable="false" class="warning-message">
                    <div class="warning-content">
                        <i class="pi pi-exclamation-triangle"></i>
                        <strong>This is a mock UI only.</strong> Settings are not functional yet and no data will be saved.
                    </div>
                </Message>
                
                <div class="grid">
                <!-- General Settings -->
                <div class="col-12">
                    <Card>
                        <template #title>
                            <div class="flex align-items-center gap-2">
                                <i class="pi pi-cog"></i>
                                <span>General Settings</span>
                            </div>
                        </template>
                        <template #content>
                            <div class="settings-list">
                                <div 
                                    v-for="setting in generalSettings" 
                                    :key="setting.label"
                                    class="setting-item"
                                >
                                    <div class="setting-label">
                                        <i :class="setting.icon" class="mr-2"></i>
                                        <span>{{ setting.label }}</span>
                                    </div>
                                    <div class="setting-value">
                                        {{ setting.value }}
                                    </div>
                                </div>
                            </div>
                        </template>
                    </Card>
                </div>

                <!-- Currencies -->
                <div class="col-12">
                    <Card>
                        <template #title>
                            <div class="flex align-items-center gap-2">
                                <i class="pi pi-dollar"></i>
                                <span>Currencies</span>
                            </div>
                        </template>
                        <template #content>
                            <div class="settings-list">
                                <!-- Main Currency Selector -->
                                <div class="setting-item setting-item-row">
                                    <div class="setting-label">
                                        <i class="pi pi-wallet mr-2"></i>
                                        <span>Main Currency</span>
                                    </div>
                                    <div class="setting-input-inline">
                                        <Select 
                                            v-model="mainCurrency" 
                                            :options="availableCurrencies" 
                                            optionLabel="name" 
                                            optionValue="code"
                                            placeholder="Select main currency"
                                            class="currency-select"
                                        >
                                            <template #value="slotProps">
                                                <div v-if="slotProps.value" class="flex align-items-center gap-2">
                                                    <span class="font-semibold">{{ slotProps.value }}</span>
                                                </div>
                                            </template>
                                            <template #option="slotProps">
                                                <div class="flex align-items-center gap-2">
                                                    <span class="font-semibold">{{ slotProps.option.code }}</span>
                                                    <span class="text-sm text-500">{{ slotProps.option.name }}</span>
                                                </div>
                                            </template>
                                        </Select>
                                    </div>
                                </div>

                                <!-- Multiple Currencies Toggle -->
                                <div class="setting-item">
                                    <div class="setting-label">
                                        <i class="pi pi-money-bill mr-2"></i>
                                        <span>Allow Multiple Currencies</span>
                                    </div>
                                    <div class="setting-toggle">
                                        <InputSwitch v-model="allowMultipleCurrencies" />
                                    </div>
                                </div>
                                
                                <!-- Active Currencies MultiSelect (shown when multiple currencies enabled) -->
                                <div v-if="allowMultipleCurrencies" class="setting-item setting-item-full">
                                    <div class="setting-label mb-2">
                                        <i class="pi pi-list mr-2"></i>
                                        <span>Active Currencies</span>
                                    </div>
                                    <div class="setting-input">
                                        <MultiSelect 
                                            v-model="selectedCurrencies" 
                                            :options="availableCurrencies" 
                                            optionLabel="name" 
                                            optionValue="code"
                                            placeholder="Select currencies"
                                            display="chip"
                                            class="w-full"
                                        >
                                            <template #option="slotProps">
                                                <div class="flex align-items-center gap-2">
                                                    <span class="font-semibold">{{ slotProps.option.code }}</span>
                                                    <span class="text-sm text-500">{{ slotProps.option.name }}</span>
                                                </div>
                                            </template>
                                        </MultiSelect>
                                    </div>
                                </div>
                            </div>
                        </template>
                    </Card>
                </div>

                <!-- Features -->
                <div class="col-12">
                    <Card>
                        <template #title>
                            <div class="flex align-items-center gap-2">
                                <i class="pi pi-star"></i>
                                <span>Features</span>
                            </div>
                        </template>
                        <template #content>
                            <div class="settings-list">
                                <div class="setting-item">
                                    <div class="setting-label">
                                        <i class="pi pi-chart-line mr-2"></i>
                                        <span>Enable Stock Functions</span>
                                    </div>
                                    <div class="setting-toggle">
                                        <InputSwitch v-model="enableStockFunctions" />
                                    </div>
                                </div>
                                <div class="setting-item">
                                    <div class="setting-label">
                                        <i class="pi pi-home mr-2"></i>
                                        <span>Enable Mortgage Functions</span>
                                    </div>
                                    <div class="setting-toggle">
                                        <InputSwitch v-model="enableMortgageFunctions" />
                                    </div>
                                </div>
                            </div>
                        </template>
                    </Card>
                </div>
            </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.card {
    height: 100%;
}

.warning-message {
    margin-bottom: 2rem;
}

.warning-message :deep(.p-message-wrapper) {
    padding: 1rem 1.25rem;
}

.warning-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1rem;
}

.warning-content i {
    font-size: 1.25rem;
}

.warning-content strong {
    margin-right: 0.25rem;
}

.settings-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.setting-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem;
    background-color: var(--surface-50);
    border-radius: 6px;
    border: 1px solid var(--surface-200);
}

.setting-label {
    display: flex;
    align-items: center;
    font-weight: 500;
    color: var(--text-color);
}

.setting-value {
    color: var(--text-color-secondary);
    font-weight: 600;
}

.setting-toggle {
    display: flex;
    align-items: center;
}

.setting-item-full {
    flex-direction: column;
    align-items: flex-start;
}

.setting-item-row {
    flex-wrap: wrap;
    gap: 1rem;
}

.setting-input {
    width: 100%;
}

.setting-input-inline {
    flex-shrink: 0;
}

.currency-select {
    min-width: 200px;
}

.currency-chip {
    background-color: var(--primary-color);
    color: white;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.875rem;
    font-weight: 600;
}

.selected-currencies-display {
    padding: 1rem;
    background-color: var(--surface-50);
    border-radius: 6px;
    border: 1px solid var(--surface-200);
}

.currency-badge {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0.75rem 1rem;
    background-color: white;
    border: 2px solid var(--primary-color);
    border-radius: 8px;
    min-width: 120px;
}

.currency-code {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--primary-color);
    margin-bottom: 0.25rem;
}

.currency-name {
    font-size: 0.75rem;
    color: var(--text-color-secondary);
    text-align: center;
}
</style>

