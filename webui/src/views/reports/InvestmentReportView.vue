<script setup>
import { ref, computed } from 'vue'
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Card from 'primevue/card'
import TreeTable from 'primevue/treetable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import { useInvestmentReport } from '@/composables/useInvestmentReport'
import { formatAmount } from '@/utils/currency'
import { useDateFormat } from '@/composables/useDateFormat'

const leftSidebarCollapsed = ref(true)
const { productPositions, treeNodes, totalByCurrency, totalInMainCurrency, mainCurrency, isLoading } = useInvestmentReport()
const { formatDate } = useDateFormat()

const expandedKeys = ref({})

const expandableKeys = computed(() => {
    const keys = {}
    for (const node of treeNodes.value) {
        if (!node.leaf) keys[node.key] = true
    }
    return keys
})

const isFullyExpanded = computed(
    () => Object.keys(expandedKeys.value).length === Object.keys(expandableKeys.value).length
        && Object.keys(expandableKeys.value).length > 0
)

const toggleExpand = () => {
    expandedKeys.value = isFullyExpanded.value ? {} : { ...expandableKeys.value }
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <Card>
                    <template #content>
                        <div class="section-header">
                            <h2 class="report-title">
                                <i class="ti ti-chart-line mr-2"></i>Current position
                            </h2>
                            <Button
                                v-if="!isLoading && productPositions.length > 0"
                                :icon="isFullyExpanded ? 'ti ti-minus' : 'ti ti-plus'"
                                :label="isFullyExpanded ? 'Collapse' : 'Expand'"
                                class="p-button-sm p-button-text"
                                @click="toggleExpand"
                            />
                        </div>
                        <div v-if="isLoading" class="text-center p-4 text-500">
                            Loading positions…
                        </div>
                        <div v-else-if="productPositions.length === 0" class="text-center p-4 text-500">
                            No investment products with positions
                        </div>
                        <template v-else>
                            <TreeTable
                                :value="treeNodes"
                                v-model:expandedKeys="expandedKeys"
                                :pt="{
                                    bodyRow: { class: 'cursor-default' }
                                }"
                            >
                                <!-- Name / expander -->
                                <Column field="symbol" header="Name" :expander="true" style="min-width: 160px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            <span class="font-semibold">{{ node.data.symbol }}</span>
                                            <span v-if="node.data.name && node.data.name !== node.data.symbol" class="text-500 text-sm ml-1">
                                                {{ node.data.name }}
                                            </span>
                                        </template>
                                        <template v-else>
                                            <span class="text-sm">{{ formatDate(node.data.openDate) }}</span>
                                            <span v-if="node.data.accountName" class="text-500 text-sm ml-1">
                                                {{ node.data.accountName }}
                                            </span>
                                        </template>
                                    </template>
                                </Column>

                                <!-- Quantity -->
                                <Column header="Quantity">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            {{ node.data.totalQuantity.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}
                                        </template>
                                        <template v-else>
                                            {{ node.data.quantity.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}
                                            <span v-if="node.data.isPartial" class="text-500 text-sm">
                                                / {{ node.data.originalQty.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}
                                            </span>
                                        </template>
                                    </template>
                                </Column>

                                <!-- Cost/Share -->
                                <Column header="Cost/Share" style="min-width: 120px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            <span v-if="node.data.avgCostPerShare != null">
                                                {{ formatAmount(node.data.avgCostPerShare) }} {{ node.data.currency }}
                                            </span>
                                            <span v-else class="text-500">—</span>
                                        </template>
                                        <template v-else>
                                            {{ formatAmount(node.data.costPerShare) }} {{ node.data.currency }}
                                        </template>
                                    </template>
                                </Column>

                                <!-- Cost Basis -->
                                <Column header="Cost Basis" style="min-width: 130px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            {{ formatAmount(node.data.investedAmount) }} {{ node.data.currency }}
                                        </template>
                                        <template v-else>
                                            {{ formatAmount(node.data.costBasis) }} {{ node.data.currency }}
                                        </template>
                                    </template>
                                </Column>

                                <!-- Market Value -->
                                <Column header="Market Value" style="min-width: 140px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            <span class="font-semibold">{{ formatAmount(node.data.totalValue) }} {{ node.data.currency }}</span>
                                        </template>
                                        <template v-else>
                                            {{ formatAmount(node.data.marketValue) }} {{ node.data.currency }}
                                        </template>
                                    </template>
                                </Column>

                                <!-- % -->
                                <Column header="%" style="min-width: 70px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            <span v-if="node.data.winLossPercent != null" class="amount" :class="node.data.winLossPercent >= 0 ? 'amount-positive' : 'amount-negative'">
                                                {{ node.data.winLossPercent >= 0 ? '+' : '' }}{{ node.data.winLossPercent.toFixed(1) }}%
                                            </span>
                                            <span v-else class="text-500">—</span>
                                        </template>
                                        <template v-else>
                                            <span v-if="node.data.lotGainLossPct != null" class="amount" :class="node.data.lotGainLossPct >= 0 ? 'amount-positive' : 'amount-negative'">
                                                {{ node.data.lotGainLossPct >= 0 ? '+' : '' }}{{ node.data.lotGainLossPct.toFixed(1) }}%
                                            </span>
                                            <span v-else class="text-500">—</span>
                                        </template>
                                    </template>
                                </Column>

                                <!-- Gain/Loss -->
                                <Column header="Gain/Loss" style="min-width: 130px">
                                    <template #body="{ node }">
                                        <template v-if="node.data.rowType === 'instrument'">
                                            <span class="amount" :class="node.data.winLoss >= 0 ? 'amount-positive' : 'amount-negative'">
                                                {{ node.data.winLoss >= 0 ? '+' : '' }}{{ formatAmount(node.data.winLoss) }} {{ node.data.currency }}
                                            </span>
                                        </template>
                                        <template v-else>
                                            <span class="amount" :class="node.data.lotGainLoss >= 0 ? 'amount-positive' : 'amount-negative'">
                                                {{ node.data.lotGainLoss >= 0 ? '+' : '' }}{{ formatAmount(node.data.lotGainLoss) }} {{ node.data.currency }}
                                            </span>
                                        </template>
                                    </template>
                                </Column>
                            </TreeTable>

                            <div class="flex justify-content-between align-items-center flex-wrap gap-2 mt-3 pt-3 border-top-1 surface-border">
                                <span class="text-500">
                                    {{ productPositions.length }} product{{ productPositions.length === 1 ? '' : 's' }} with positions
                                </span>
                                <div class="flex align-items-center gap-3 flex-wrap">
                                    <span class="font-semibold text-lg">
                                        {{ formatAmount(totalInMainCurrency) }} {{ mainCurrency }}
                                    </span>
                                    <span
                                        v-for="t in totalByCurrency"
                                        v-show="totalByCurrency.length > 1"
                                        :key="t.currency"
                                        class="text-500 text-sm"
                                    >
                                        {{ formatAmount(t.value) }} {{ t.currency }}
                                    </span>
                                </div>
                            </div>
                        </template>
                    </template>
                </Card>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
}

.report-title {
    color: var(--c-surface-700);
    margin: 0;
    font-size: 1.1rem;
}
</style>
