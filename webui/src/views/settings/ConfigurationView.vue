<script setup>
import Card from 'primevue/card'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import { useSettingsStore } from '@/store/settingsStore'

const settings = useSettingsStore()
</script>

<template>
    <div>
        <!-- Error state -->
        <Message v-if="settings.error" severity="error" :closable="false" icon="ti ti-alert-triangle">
            Failed to load settings: {{ settings.error }}
        </Message>

        <!-- Loading state -->
        <div v-if="settings.isLoading" class="flex justify-content-center p-5">
            <i class="ti ti-loader-2 spin-icon" style="font-size: 2rem"></i>
        </div>

        <div v-else-if="settings.isLoaded" class="flex flex-column gap-3">
            <!-- General Settings -->
            <Card>
                <template #title>
                    <div class="flex align-items-center gap-2">
                        <i class="ti ti-settings"></i>
                        <span>General Settings</span>
                    </div>
                </template>
                <template #content>
                    <Message severity="info" :closable="false" icon="ti ti-info-circle">
                        These settings are controlled by the server configuration file and cannot be changed from the UI.
                        See the <router-link to="/docs/getting-started/configuration">documentation</router-link> for details on how to configure these values.
                    </Message>

                    <h4 class="text-sm font-semibold text-color-secondary mt-3 mb-2">General</h4>
                    <div class="settings-list">
                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-calendar mr-2"></i>
                                <span>Date Format</span>
                            </div>
                            <div class="setting-value">
                                <code>{{ settings.dateFormat }}</code>
                            </div>
                        </div>
                    </div>

                    <h4 class="text-sm font-semibold text-color-secondary mt-3 mb-2">Currencies</h4>
                    <div class="settings-list">
                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-wallet mr-2"></i>
                                <span>Main Currency</span>
                            </div>
                            <div class="setting-value">
                                <Tag :value="settings.mainCurrency" severity="primary" />
                            </div>
                        </div>

                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-list mr-2"></i>
                                <span>Available Currencies</span>
                            </div>
                            <div class="setting-value">
                                <div class="flex gap-2 flex-wrap">
                                    <Tag
                                        v-for="currency in settings.currencies"
                                        :key="currency"
                                        :value="currency"
                                        :severity="currency === settings.mainCurrency ? 'primary' : 'secondary'"
                                    />
                                </div>
                            </div>
                        </div>
                    </div>

                    <h4 class="text-sm font-semibold text-color-secondary mt-3 mb-2">Features</h4>
                    <div class="settings-list">
                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-chart-line mr-2"></i>
                                <span>Investment Instruments</span>
                            </div>
                            <div class="setting-value">
                                <Tag
                                    :value="settings.investmentInstruments ? 'Enabled' : 'Disabled'"
                                    :severity="settings.investmentInstruments ? 'success' : 'secondary'"
                                />
                            </div>
                        </div>

                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-gift mr-2"></i>
                                <span>RSU (Restricted Stock Units)</span>
                            </div>
                            <div class="setting-value">
                                <Tag
                                    :value="settings.rsu ? 'Enabled' : 'Disabled'"
                                    :severity="settings.rsu ? 'success' : 'secondary'"
                                />
                            </div>
                        </div>

                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-calculator mr-2"></i>
                                <span>Financial Simulator</span>
                            </div>
                            <div class="setting-value">
                                <Tag
                                    :value="settings.financialSimulator ? 'Enabled' : 'Disabled'"
                                    :severity="settings.financialSimulator ? 'success' : 'secondary'"
                                />
                            </div>
                        </div>
                    </div>
                </template>
            </Card>
        </div>
    </div>
</template>

<style scoped>
.p-message-enter-active {
    animation: none !important;
}

:deep(.p-message-icon) {
    font-size: 1.5rem !important;
    width: 1.5rem !important;
    height: 1.5rem !important;
}

:deep(.p-message) {
    margin-bottom: 1rem;
}

:deep(.p-message a) {
    color: inherit;
    text-decoration: underline;
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

.setting-value code {
    background-color: var(--surface-100);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.9rem;
}
</style>
