<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import Card from 'primevue/card'
import Message from 'primevue/message'

const leftSidebarCollapsed = ref(true)
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <Message severity="info" :closable="false" class="info-message">
                    <div class="info-content">
                        <i class="pi pi-info-circle"></i>
                        <span>
                            Utility tools and helpers. Add tools here as needed.
                        </span>
                    </div>
                </Message>

                <div class="grid">
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-wrench"></i>
                                    <span>Tools</span>
                                </div>
                            </template>
                            <template #content>
                                <p class="text-color-secondary m-0">No tools configured yet.</p>
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
</style>
