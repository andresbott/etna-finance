<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import Card from 'primevue/card'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import { useSettingsStore } from '@/store/settingsStore'

const leftSidebarCollapsed = ref(true)
const settings = useSettingsStore()
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Info message -->
                <Message severity="info" :closable="false" class="info-message">
                    <div class="info-content">
                        <i class="pi pi-info-circle"></i>
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
                        <i class="pi pi-exclamation-triangle"></i>
                        <span>Failed to load settings: {{ settings.error }}</span>
                    </div>
                </Message>

                <!-- Loading state -->
                <div v-if="settings.isLoading" class="flex justify-content-center p-5">
                    <i class="pi pi-spin pi-spinner" style="font-size: 2rem"></i>
                </div>

                <div v-else-if="settings.isLoaded" class="grid">
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
                                    <div class="setting-item">
                                        <div class="setting-label">
                                            <i class="pi pi-calendar mr-2"></i>
                                            <span>Date Format</span>
                                        </div>
                                        <div class="setting-value">
                                            <code>{{ settings.dateFormat }}</code>
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
                                    <div class="setting-item">
                                        <div class="setting-label">
                                            <i class="pi pi-wallet mr-2"></i>
                                            <span>Main Currency</span>
                                        </div>
                                        <div class="setting-value">
                                            <Tag :value="settings.mainCurrency" severity="primary" />
                                        </div>
                                    </div>

                                    <div class="setting-item">
                                        <div class="setting-label">
                                            <i class="pi pi-list mr-2"></i>
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
                                            <span>Investment Instruments</span>
                                        </div>
                                        <div class="setting-value">
                                            <Tag
                                                :value="settings.instruments ? 'Enabled' : 'Disabled'"
                                                :severity="settings.instruments ? 'success' : 'secondary'"
                                            />
                                        </div>
                                    </div>
                                    <div v-if="settings.marketDataSymbols && settings.marketDataSymbols.length > 0" class="setting-item">
                                        <div class="setting-label">
                                            <i class="pi pi-list mr-2"></i>
                                            <span>Symbols with market data</span>
                                        </div>
                                        <div class="setting-value">
                                            <div class="flex gap-2 flex-wrap">
                                                <Tag
                                                    v-for="sym in settings.marketDataSymbols"
                                                    :key="sym"
                                                    :value="sym"
                                                    severity="secondary"
                                                />
                                            </div>
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
