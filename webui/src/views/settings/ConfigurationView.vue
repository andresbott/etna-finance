<script setup>
import Card from 'primevue/card'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import { useSettingsStore } from '@/store/settingsStore'

const settings = useSettingsStore()
</script>

<template>
    <div>
        <!-- Info message -->
        <Message severity="info" :closable="false" class="info-message">
            <div class="info-content">
                <i class="ti ti-info-circle"></i>
                <span>
                    These settings are controlled by the server configuration file and cannot be changed from the UI.
                    <!-- TODO: add link to documentation once available -->
                    See the <strong>documentation</strong> for details on how to configure these values.
                </span>
            </div>
        </Message>

        <!-- Error state -->
        <Message v-if="settings.error" severity="error" :closable="false" class="info-message">
            <div class="info-content">
                <i class="ti ti-alert-triangle"></i>
                <span>Failed to load settings: {{ settings.error }}</span>
            </div>
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
                </template>
            </Card>

            <!-- Currencies -->
            <Card>
                <template #title>
                    <div class="flex align-items-center gap-2">
                        <i class="ti ti-currency-dollar"></i>
                        <span>Currencies</span>
                    </div>
                </template>
                <template #content>
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
                </template>
            </Card>

            <!-- Features -->
            <Card>
                <template #title>
                    <div class="flex align-items-center gap-2">
                        <i class="ti ti-star"></i>
                        <span>Features</span>
                    </div>
                </template>
                <template #content>
                    <div class="settings-list">
                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-chart-line mr-2"></i>
                                <span>Investment Products</span>
                            </div>
                            <div class="setting-value">
                                <Tag
                                    :value="settings.instruments ? 'Enabled' : 'Disabled'"
                                    :severity="settings.instruments ? 'success' : 'secondary'"
                                />
                            </div>
                        </div>
                        <div class="setting-item">
                            <div class="setting-label">
                                <i class="ti ti-calculator mr-2"></i>
                                <span>Financial Tools</span>
                            </div>
                            <div class="setting-value">
                                <Tag
                                    :value="settings.tools ? 'Enabled' : 'Disabled'"
                                    :severity="settings.tools ? 'success' : 'secondary'"
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
.info-message {
    margin-bottom: 1.5rem;
}

.info-message :deep(.p-message-wrapper) {
    padding: 1rem 1.25rem;
}

.info-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1rem;
}

.info-content i {
    font-size: 1.25rem;
    flex-shrink: 0;
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
