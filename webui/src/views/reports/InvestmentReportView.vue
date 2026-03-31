<script setup>
import { ref, computed } from 'vue'
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Card from 'primevue/card'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import TreeTable from 'primevue/treetable'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import { useInvestmentReport } from '@/composables/useInvestmentReport'
import { useInvestmentReturns } from '@/composables/useInvestmentReturns'
import { formatAmount } from '@/utils/currency'
import { useDateFormat } from '@/composables/useDateFormat'

const leftSidebarCollapsed = ref(true)
const { productPositions, treeNodes, totalByCurrency, totalInMainCurrency, mainCurrency, isLoading } = useInvestmentReport()
const { returnRows, totals, isLoading: returnsLoading } = useInvestmentReturns()
const { formatDate } = useDateFormat()

const statusSeverity = (status) => {
    if (status === 'closed') return 'danger'
    if (status === 'mixed') return 'warning'
    return 'success'
}
const statusLabel = (status) => {
    if (status === 'closed') return 'Closed'
    if (status === 'mixed') return 'Partial'
    return 'Open'
}

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
                        <TabView>
                            <TabPanel header="Returns">
                                <div v-if="returnsLoading" class="text-center p-4 text-500">
                                    Loading returns…
                                </div>
                                <div v-else-if="returnRows.length === 0" class="text-center p-4 text-500">
                                    No investment activity
                                </div>
                                <template v-else>
                                    <DataTable
                                        :value="returnRows"
                                        :pt="{ bodyRow: { class: 'cursor-default' } }"
                                    >
                                        <Column field="symbol" header="Instrument" style="min-width: 150px">
                                            <template #body="{ data }">
                                                <span class="font-semibold">{{ data.symbol }}</span>
                                            </template>
                                        </Column>

                                        <Column header="Status" style="min-width: 80px">
                                            <template #body="{ data }">
                                                <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
                                            </template>
                                        </Column>

                                        <Column header="Invested" style="min-width: 120px">
                                            <template #body="{ data }">
                                                {{ formatAmount(data.totalInvested) }} {{ data.currency }}
                                            </template>
                                        </Column>

                                        <Column header="Realized" style="min-width: 120px">
                                            <template #body="{ data }">
                                                <span v-if="data.realizedGL !== 0" class="amount" :class="data.realizedGL >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ data.realizedGL >= 0 ? '+' : '' }}{{ formatAmount(data.realizedGL) }} {{ data.currency }}
                                                </span>
                                                <span v-else class="text-500">—</span>
                                            </template>
                                        </Column>

                                        <Column header="Unrealized" style="min-width: 120px">
                                            <template #body="{ data }">
                                                <span v-if="data.unrealizedGL !== 0" class="amount" :class="data.unrealizedGL >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ data.unrealizedGL >= 0 ? '+' : '' }}{{ formatAmount(data.unrealizedGL) }} {{ data.currency }}
                                                </span>
                                                <span v-else class="text-500">—</span>
                                            </template>
                                        </Column>

                                        <Column header="Total Return" style="min-width: 130px">
                                            <template #body="{ data }">
                                                <span class="amount font-semibold" :class="data.totalReturn >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ data.totalReturn >= 0 ? '+' : '' }}{{ formatAmount(data.totalReturn) }} {{ data.currency }}
                                                </span>
                                            </template>
                                        </Column>

                                        <Column header="ROI" style="min-width: 80px">
                                            <template #body="{ data }">
                                                <span v-if="data.roi != null" class="amount" :class="data.roi >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ data.roi >= 0 ? '+' : '' }}{{ data.roi.toFixed(1) }}%
                                                </span>
                                                <span v-else class="text-500">—</span>
                                            </template>
                                        </Column>

                                        <Column header="Annualized" style="min-width: 100px">
                                            <template #body="{ data }">
                                                <span v-if="data.annualizedReturn != null" class="amount" :class="data.annualizedReturn >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ data.annualizedReturn >= 0 ? '+' : '' }}{{ data.annualizedReturn.toFixed(1) }}%
                                                </span>
                                                <span v-else class="text-500">—</span>
                                            </template>
                                        </Column>

                                    </DataTable>

                                    <div class="flex justify-content-between align-items-center flex-wrap gap-2 mt-3 pt-3 border-top-1 surface-border">
                                        <span class="text-500">
                                            {{ returnRows.length }} instrument{{ returnRows.length === 1 ? '' : 's' }}
                                        </span>
                                        <div class="flex align-items-center gap-4 flex-wrap">
                                            <span class="text-sm">
                                                Invested: <span class="font-semibold">{{ formatAmount(totals.totalInvested) }} {{ totals.currency }}</span>
                                            </span>
                                            <span class="text-sm">
                                                Return:
                                                <span class="font-semibold amount" :class="totals.totalReturn >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ totals.totalReturn >= 0 ? '+' : '' }}{{ formatAmount(totals.totalReturn) }} {{ totals.currency }}
                                                </span>
                                            </span>
                                            <span v-if="totals.annualizedReturn != null" class="text-sm">
                                                Annualized:
                                                <span class="font-semibold amount" :class="totals.annualizedReturn >= 0 ? 'amount-positive' : 'amount-negative'">
                                                    {{ totals.annualizedReturn >= 0 ? '+' : '' }}{{ totals.annualizedReturn.toFixed(1) }}%
                                                </span>
                                            </span>
                                        </div>
                                    </div>
                                </template>
                            </TabPanel>
                            <TabPanel header="Open Positions">
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

                                        <!-- Annualized -->
                                        <Column header="Annualized" style="min-width: 100px">
                                            <template #body="{ node }">
                                                <template v-if="node.data.rowType === 'instrument'">
                                                    <span v-if="node.data.annualized != null" class="amount" :class="node.data.annualized >= 0 ? 'amount-positive' : 'amount-negative'">
                                                        {{ node.data.annualized >= 0 ? '+' : '' }}{{ node.data.annualized.toFixed(1) }}%
                                                    </span>
                                                    <span v-else class="text-500">—</span>
                                                </template>
                                                <template v-else>
                                                    <span v-if="node.data.lotAnnualized != null" class="amount" :class="node.data.lotAnnualized >= 0 ? 'amount-positive' : 'amount-negative'">
                                                        {{ node.data.lotAnnualized >= 0 ? '+' : '' }}{{ node.data.lotAnnualized.toFixed(1) }}%
                                                    </span>
                                                    <span v-else class="text-500">—</span>
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
                            </TabPanel>
                        </TabView>
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
