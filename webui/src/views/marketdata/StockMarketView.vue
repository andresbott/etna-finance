<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import { useDateFormat } from '@/composables/useDateFormat'
import {
    useMarketInstruments,
    formatPrice,
    formatPct,
    getChangeSeverity
} from '@/composables/useMarketData'

const { formatDate } = useDateFormat()

const router = useRouter()
const { instruments, isLoading, isError, error, refetch } = useMarketInstruments()
const leftSidebarCollapsed = ref(true)

const onRowClick = (event) => {
    router.push({ name: 'stock-detail', params: { id: event.data.id, tab: 'overview' } })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="header">
                    <h1>
                        <i class="pi pi-chart-line"></i>
                        Stock Market
                    </h1>
                    <p class="text-color-secondary mt-0 mb-3">
                        Investment instruments overview with market data
                    </p>
                </div>

                <Message v-if="isError" severity="error" :closable="false" class="mb-3">
                    <div class="flex align-items-center gap-2 flex-wrap">
                        <i class="pi pi-exclamation-triangle"></i>
                        <span>{{ error?.message ?? 'Failed to load market data.' }}</span>
                        <Button label="Retry" icon="pi pi-refresh" text size="small" @click="refetch" />
                    </div>
                </Message>

                <Card v-if="!instruments.length && !isLoading">
                    <template #content>
                        <div class="empty-message">
                            No instruments configured. Add instruments in
                            <router-link to="/instruments">Investment Instruments</router-link>
                            to see market data here.
                        </div>
                    </template>
                </Card>

                <Card v-else>
                    <template #content>
                        <DataTable
                            :value="instruments"
                            :loading="isLoading"
                            dataKey="id"
                            stripedRows
                            :paginator="instruments.length > 15"
                            :rows="15"
                            class="p-datatable-sm clickable-rows stock-table"
                            selectionMode="single"
                            @rowClick="onRowClick"
                        >
                            <Column field="symbol" header="Symbol">
                                <template #body="{ data }">
                                    <span class="font-bold">{{ data.symbol }}</span>
                                </template>
                            </Column>
                            <Column field="name" header="Name" />
                            <Column field="lastPrice" header="Price">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ formatPrice(data.lastPrice) }}</span>
                                    <span class="text-color-secondary text-sm ml-1">{{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column field="changePct" header="Change">
                                <template #body="{ data }">
                                    <Tag
                                        :value="formatPct(data.changePct)"
                                        :severity="getChangeSeverity(data.changePct)"
                                    />
                                </template>
                            </Column>
                            <Column field="lastUpdate" header="Last update" bodyClass="last-update-cell" headerClass="last-update-cell">
                                <template #body="{ data }">
                                    {{ data.lastUpdate ? formatDate(data.lastUpdate) : '-' }}
                                </template>
                            </Column>
                        </DataTable>
                        <p class="text-color-secondary text-sm mt-2 mb-0">
                            <i class="pi pi-info-circle"></i> Click a row to open details, chart and edit data.
                        </p>
                    </template>
                </Card>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>

.header {
    margin-bottom: 0.5rem;
}

.header h1 {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin: 0 0 0.5rem 0;
}

.header h1 i {
    color: var(--p-primary-color);
}

.empty-message {
    padding: 1rem;
    text-align: center;
    color: var(--p-text-muted-color);
}

.empty-message a {
    color: var(--p-primary-color);
    text-decoration: none;
    font-weight: 600;
}

.empty-message a:hover {
    text-decoration: underline;
}

:deep(.clickable-rows .p-datatable-tbody > tr) {
    cursor: pointer;
}

:deep(.last-update-cell) {
    text-align: right;
}

:deep(.stock-table .p-datatable-thead th.last-update-cell .p-datatable-column-header-content) {
    justify-content: flex-end;
}
</style>
