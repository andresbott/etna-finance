<script setup lang="ts">
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import { COMPARE_PALETTE } from '@/composables/useCompareChart'
import {
    formatPrice,
    formatVolume,
    formatPct,
    formatChange,
    getChangeSeverity
} from '@/composables/useMarketData'
import type { CompareStatsRow } from '@/composables/useCompareStats'

defineProps<{ rows: CompareStatsRow[] }>()

// Symbol order matches the chart series order, so dot colors line up with chart lines.
function colorFor(index: number): string {
    return COMPARE_PALETTE[index % COMPARE_PALETTE.length]
}
</script>

<template>
    <DataTable
        :value="rows"
        stripedRows
        dataKey="symbol"
        class="p-datatable-sm compare-stats-table"
    >
        <Column field="symbol" header="Symbol" sortable>
            <template #body="{ data, index }">
                <span class="symbol-cell">
                    <span
                        class="color-dot"
                        :style="{ backgroundColor: colorFor(index) }"
                        :title="data.symbol"
                        aria-hidden="true"
                    ></span>
                    <span class="font-bold">{{ data.symbol }}</span>
                </span>
            </template>
        </Column>
        <Column field="open" header="Open" sortable>
            <template #body="{ data }">{{ formatPrice(data.open) }}</template>
        </Column>
        <Column field="close" header="Close" sortable>
            <template #body="{ data }">{{ formatPrice(data.close) }}</template>
        </Column>
        <Column field="change" header="Change" sortable>
            <template #body="{ data }">
                <span :class="data.change >= 0 ? 'text-green-600' : 'text-red-600'" class="font-semibold">
                    {{ formatChange(data.change) }}
                </span>
            </template>
        </Column>
        <Column field="changePct" header="Change %" sortable>
            <template #body="{ data }">
                <Tag :value="formatPct(data.changePct)" :severity="getChangeSeverity(data.changePct)" />
            </template>
        </Column>
        <Column field="high" header="High" sortable>
            <template #body="{ data }">{{ formatPrice(data.high) }}</template>
        </Column>
        <Column field="low" header="Low" sortable>
            <template #body="{ data }">{{ formatPrice(data.low) }}</template>
        </Column>
        <Column field="maxDrawdownPct" header="Max Drawdown" sortable>
            <template #body="{ data }">
                <Tag :value="formatPct(data.maxDrawdownPct)" :severity="getChangeSeverity(data.maxDrawdownPct)" />
            </template>
        </Column>
        <Column field="volatilityPct" header="Volatility" sortable>
            <template #body="{ data }">{{ formatPct(data.volatilityPct) }}</template>
        </Column>
        <Column field="avgVolume" header="Avg Volume" sortable>
            <template #body="{ data }">{{ formatVolume(data.avgVolume) }}</template>
        </Column>
    </DataTable>
</template>

<style scoped>
.symbol-cell {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
}

.color-dot {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
}
</style>
