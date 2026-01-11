<script setup>
import { computed, onMounted, watch } from 'vue'
import Card from 'primevue/card'
import { useAccounts } from '@/composables/useAccounts.js'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'

const { accounts: accountProviders } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation

// Group accounts by provider with balance data
const accountsByProvider = computed(() => {
    if (!accountProviders.value || !balanceReport.value) return []
    
    return accountProviders.value
        .map((provider) => {
            if (!provider.accounts || !Array.isArray(provider.accounts)) return null
            
            // Get accounts with their balance data for this provider
            const accountsWithBalances = provider.accounts
                .map((account) => {
                    const reportData = balanceReport.value?.accounts?.[account.id]
                    if (!reportData) return null
                    
                    return {
                        ...account,
                        reportData
                    }
                })
                .filter(Boolean)
            
            // Only include provider if it has accounts with balance data
            if (accountsWithBalances.length === 0) return null
            
            return {
                id: provider.id,
                name: provider.name,
                accounts: accountsWithBalances
            }
        })
        .filter(Boolean)
})

const getLatestBalance = (account) => {
    if (!account.reportData || account.reportData.length === 0) {
        return 0
    }
    // Get the last entry without mutating the array
    const latestEntry = account.reportData[account.reportData.length - 1]
    return latestEntry?.Sum || 0
}

// Fetch balance reports when accounts are loaded
watch(
    accountProviders,
    (providers) => {
        if (providers && providers.length > 0) {
            // Collect all account IDs from all providers
            const accountIds = []
            for (const provider of providers) {
                if (provider.accounts && Array.isArray(provider.accounts)) {
                    accountIds.push(...provider.accounts.map(account => account.id).filter(Boolean))
                }
            }
            
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
        <template #title>Account Balances</template>
        <template #content>
            <div v-if="accountsByProvider.length === 0" class="text-center p-3 text-500">
                No accounts available
            </div>
            <div v-else class="flex flex-column gap-4">
                <!-- Group by Provider -->
                <div
                    v-for="provider in accountsByProvider"
                    :key="provider.id"
                    class="flex flex-column gap-2"
                >
                    <!-- Provider Header -->
                    <div class="flex align-items-center gap-2 pb-2" style="border-bottom: 1px solid rgba(0, 0, 0, 0.06)">
                        <i class="pi pi-building text-primary"></i>
                        <span class="font-bold text-lg">{{ provider.name }}</span>
                    </div>
                    
                    <!-- Accounts under this provider -->
                    <div class="flex flex-column gap-2 ml-3">
                        <div
                            v-for="account in provider.accounts"
                            :key="account.id"
                            class="flex justify-content-between align-items-center p-2 border-round"
                            style="background: var(--surface-ground)"
                        >
                            <div class="flex align-items-center gap-2">
                                <i class="pi pi-wallet"></i>
                                <span>{{ account.name }}</span>
                            </div>
                            <div class="flex align-items-center gap-2">
                                <span class="font-bold">{{ formatAmount(getLatestBalance(account)) }}</span>
                                <span class="text-500">{{ account.currency || 'CHF' }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </template>
    </Card>
</template>

