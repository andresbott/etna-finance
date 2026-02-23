<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { useInvestmentReport } from '@/composables/useInvestmentReport'
import { formatAmount } from '@/utils/currency'

const leftSidebarCollapsed = ref(true)
const { productPositions, totalByCurrency, isLoading } = useInvestmentReport()
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <Card>
                    <template #title>
                        <div class="flex align-items-center gap-2">
                            <i class="pi pi-chart-line"></i>
                            <span>Investment Report</span>
                        </div>
                    </template>
                    <template #subtitle>
                        All investment products and current positions
                    </template>
                    <template #content>
                        <div v-if="isLoading" class="text-center p-4 text-500">
                            Loading positions…
                        </div>
                        <div v-else-if="productPositions.length === 0" class="text-center p-4 text-500">
                            No investment products with positions
                        </div>
                        <DataTable
                            v-else
                            :value="productPositions"
                            stripedRows
                            :pt="{
                                bodyRow: { class: 'cursor-default' }
                            }"
                        >
                            <Column field="symbol" header="Symbol" style="min-width: 100px">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.symbol }}</span>
                                    <span v-if="data.name && data.name !== data.symbol" class="text-500 text-sm ml-1">
                                        {{ data.name }}
                                    </span>
                                </template>
                            </Column>
                            <Column field="totalQuantity" header="Quantity">
                                <template #body="{ data }">
                                    {{ data.totalQuantity.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}
                                </template>
                            </Column>
                            <Column header="Price" style="min-width: 120px">
                                <template #body="{ data }">
                                    <span v-if="data.lastPrice != null">
                                        {{ formatAmount(data.lastPrice) }} {{ data.currency }}
                                    </span>
                                    <span v-else class="text-500">—</span>
                                </template>
                            </Column>
                            <Column header="Invested" style="min-width: 140px">
                                <template #body="{ data }">
                                    <span>{{ formatAmount(data.investedAmount) }} {{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column header="Value" style="min-width: 140px">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ formatAmount(data.totalValue) }} {{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column header="Win/Loss" style="min-width: 130px">
                                <template #body="{ data }">
                                    <span :class="data.winLoss >= 0 ? 'text-green-600' : 'text-red-600'">
                                        {{ data.winLoss >= 0 ? '+' : '' }}{{ formatAmount(data.winLoss) }} {{ data.currency }}
                                    </span>
                                </template>
                            </Column>
                        </DataTable>
                        <div
                            v-if="productPositions.length > 0"
                            class="flex justify-content-between align-items-center flex-wrap gap-2 mt-3 pt-3 border-top-1 surface-border"
                        >
                            <span class="text-500">
                                {{ productPositions.length }} product{{ productPositions.length === 1 ? '' : 's' }} with positions
                            </span>
                            <div class="flex gap-4 flex-wrap">
                                <span
                                    v-for="t in totalByCurrency"
                                    :key="t.currency"
                                    class="font-semibold"
                                >
                                    {{ formatAmount(t.value) }} {{ t.currency }}
                                </span>
                            </div>
                        </div>
                    </template>
                </Card>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>
