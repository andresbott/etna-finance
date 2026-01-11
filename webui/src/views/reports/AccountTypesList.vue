<script setup>
import { computed, watch } from 'vue'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { useAccounts } from '@/composables/useAccounts.js'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'

const { accounts: accountProviders } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation

// Gather all accounts from all providers
const allAccounts = computed(() => {
    if (!accountProviders.value) return []
    
    const accounts = []
    for (const provider of accountProviders.value) {
        if (provider.accounts && Array.isArray(provider.accounts)) {
            accounts.push(...provider.accounts)
        }
    }
    return accounts
})

// Group accounts by type and currency with balance data
const accountsByType = computed(() => {
    if (!allAccounts.value || !balanceReport.value) return []
    
    // First, create a map of accounts with their balance data
    const accountsWithBalances = allAccounts.value
        .map((account) => {
            const reportData = balanceReport.value?.accounts?.[account.id]
            if (!reportData) return null
            
            return {
                ...account,
                reportData
            }
        })
        .filter(Boolean)
    
    // Group by account type and aggregate by currency
    const grouped = {}
    for (const account of accountsWithBalances) {
        const type = account.type || 'Other'
        const currency = account.currency || 'CHF'
        
        if (!grouped[type]) {
            grouped[type] = {
                type,
                currencies: {}
            }
        }
        
        if (!grouped[type].currencies[currency]) {
            grouped[type].currencies[currency] = 0
        }
        
        grouped[type].currencies[currency] += getLatestBalance(account)
    }
    
    return Object.values(grouped)
})

// Get all unique currencies across all account types
const allCurrencies = computed(() => {
    const currencies = new Set()
    accountsByType.value.forEach(typeGroup => {
        Object.keys(typeGroup.currencies).forEach(currency => currencies.add(currency))
    })
    return Array.from(currencies).sort()
})

const getLatestBalance = (account) => {
    if (!account.reportData || account.reportData.length === 0) {
        return 0
    }
    // Get the last entry without mutating the array
    const latestEntry = account.reportData[account.reportData.length - 1]
    return latestEntry?.Sum || 0
}

const getAccountTypeIcon = (type) => {
    switch (type) {
        case 'Cash':
            return 'pi pi-money-bill'
        case 'Bank':
            return 'pi pi-credit-card'
        case 'Investment':
            return 'pi pi-chart-line'
        case 'Credit':
            return 'pi pi-credit-card'
        default:
            return 'pi pi-wallet'
    }
}

// Fetch balance reports when accounts are loaded
watch(
    allAccounts,
    (accounts) => {
        if (accounts && accounts.length > 0) {
            const accountIds = accounts.map((account) => account.id).filter(Boolean)
            
            if (accountIds.length > 0) {
                mutate({
                    accountIds,
                    steps: 30,
                    startDate: '2025-01-03'
                })
            }
        }
    },
    { immediate: true }
)
</script>

<template>
    <Card>
        <template #title>Account Types</template>
        <template #content>
            <div v-if="accountsByType.length === 0" class="text-center p-3 text-500">
                No accounts available
            </div>
            <DataTable v-else :value="accountsByType" stripedRows>
                <!-- Account Type Column -->
                <Column field="type" header="Type" style="min-width: 200px">
                    <template #body="slotProps">
                        <div class="flex align-items-center gap-2">
                            <i :class="getAccountTypeIcon(slotProps.data.type)"></i>
                            <span class="font-semibold">{{ slotProps.data.type }}</span>
                        </div>
                    </template>
                </Column>
                
                <!-- Dynamic Currency Columns -->
                <Column 
                    v-for="currency in allCurrencies" 
                    :key="currency" 
                    :header="currency"
                    class="amount-column"
                    style="min-width: 150px"
                >
                    <template #body="slotProps">
                        <div v-if="slotProps.data.currencies[currency]">
                            <span>{{ formatAmount(slotProps.data.currencies[currency]) }}</span>
                        </div>
                        <div v-else class="text-400">
                            —
                        </div>
                    </template>
                </Column>
            </DataTable>
        </template>
    </Card>
</template>

<style scoped>
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
