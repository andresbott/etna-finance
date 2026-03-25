<script setup>
import { computed } from 'vue'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { useAccountTypesData } from '@/composables/useAccountTypesData'
import { formatAmount } from '@/utils/currency'
import { getAccountTypeIcon, getAccountTypeLabel, ACCOUNT_TYPES } from '@/types/account'

const { accountsByType, totalInMainCurrency, mainCurrency } = useAccountTypesData()

// Account types excluded from the grand total (not liquid / not accessible)
const excludedFromTotal = new Set([ACCOUNT_TYPES.RESTRICTED_STOCK, ACCOUNT_TYPES.PREPAID_EXPENSE])

const totalExcluded = computed(() => {
    const rows = accountsByType.value ?? []
    return rows
        .filter((row) => !excludedFromTotal.has(row.type))
        .reduce((sum, row) => sum + totalInMainCurrency(row), 0)
})

const totalTooltipText = 'Values are converted to the main currency. Restricted stock and prepaid expenses are not included in this total.'
</script>

<template>
    <Card>
        <template #title>
            <div class="flex align-items-center gap-2">
                <span>Account Types</span>
                <i
                    class="ti ti-help-circle text-400 cursor-help"
                    style="font-size: 1rem"
                    v-tooltip.bottom="totalTooltipText"
                    aria-label="Info"
                />
            </div>
        </template>
        <template #content>
            <div v-if="accountsByType.length === 0" class="text-center p-3 text-500">
                No accounts available
            </div>
            <DataTable v-else :value="accountsByType" stripedRows>
                <!-- Account Type Column -->
                <Column field="type" header="Type" style="min-width: 200px">
                    <template #body="slotProps">
                        <div class="flex align-items-center gap-2">
                            <i :class="['ti', `ti-${getAccountTypeIcon(slotProps.data.type)}`]"></i>
                            <span class="font-semibold">{{ getAccountTypeLabel(slotProps.data.type) }}</span>
                        </div>
                    </template>
                </Column>

                <!-- Total in main currency -->
                <Column
                    :header="`Total (${mainCurrency})`"
                    class="amount-column total-column"
                    style="min-width: 150px"
                >
                    <template #body="slotProps">
                        <span class="font-semibold">{{ formatAmount(totalInMainCurrency(slotProps.data)) }}</span>
                    </template>
                </Column>
            </DataTable>
            <div v-if="accountsByType.length > 0" class="total-row mt-3 pt-3">
                <span class="total-label">Total (excl. restricted stock & prepaid)</span>
                <span class="total-value">{{ formatAmount(totalExcluded) }} {{ mainCurrency }}</span>
            </div>
        </template>
    </Card>
</template>

<style scoped>
.total-row {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--p-surface-border);
    padding: 0.75rem 0.5rem 0;
}

.total-label {
    font-weight: 600;
    font-size: 1rem;
}

.total-value {
    font-weight: 700;
    font-size: 1.25rem;
}

:deep(.amount-column .p-datatable-column-title) {
    margin-left: auto;
}

:deep(.amount-column .p-column-header-content) {
    justify-content: flex-end;
}

:deep(.amount-column) {
    text-align: right;
}
</style>
