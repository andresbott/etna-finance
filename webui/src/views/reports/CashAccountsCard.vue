<script setup>
import { computed, watch } from 'vue'
import Card from 'primevue/card'
import { useAccounts } from '@/composables/useAccounts'
import { useBalance } from '@/composables/useGetBalanceReport'
import { formatAmount } from '@/utils/currency'
import { ACCOUNT_TYPES } from '@/types/account'

const CASH_ACCOUNT_TYPES = [ACCOUNT_TYPES.CASH, ACCOUNT_TYPES.CHECKING, ACCOUNT_TYPES.SAVINGS, ACCOUNT_TYPES.LENT]

const { accounts: accountProviders } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation

// Ensure balance report is fetched (shared with TimeBalance/AccountTypesList; must include all accounts)
watch(
    accountProviders,
    (providers) => {
        if (providers && providers.length > 0) {
            const accountIds = []
            for (const provider of providers) {
                if (provider.accounts && Array.isArray(provider.accounts)) {
                    accountIds.push(...provider.accounts.map((a) => a.id).filter(Boolean))
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

// Group only cash-type accounts by provider
const accountsByProvider = computed(() => {
    if (!accountProviders.value || !balanceReport.value) return []

    return accountProviders.value
        .map((provider) => {
            if (!provider.accounts || !Array.isArray(provider.accounts)) return null

            const accountsWithBalances = provider.accounts
                .filter((acc) => CASH_ACCOUNT_TYPES.includes(acc.type))
                .map((account) => {
                    const reportData = balanceReport.value?.accounts?.[account.id]
                    if (!reportData) return null
                    return { ...account, reportData }
                })
                .filter(Boolean)

            if (accountsWithBalances.length === 0) return null

            return {
                id: provider.id,
                name: provider.name,
                icon: provider.icon,
                accounts: accountsWithBalances
            }
        })
        .filter(Boolean)
})

const getLatestBalance = (account) => {
    if (!account.reportData || account.reportData.length === 0) return 0
    const latestEntry = account.reportData[account.reportData.length - 1]
    return latestEntry?.sum || 0
}
</script>

<template>
    <Card>
        <template #title>Cash Accounts</template>
        <template #content>
            <div v-if="accountsByProvider.length === 0" class="text-center p-3 text-500">
                No cash accounts available
            </div>
            <div v-else class="flex flex-column gap-4">
                <div
                    v-for="provider in accountsByProvider"
                    :key="provider.id"
                    class="flex flex-column gap-2"
                >
                    <div
                        class="flex align-items-center gap-2 pb-2"
                        style="border-bottom: 1px solid rgba(0, 0, 0, 0.06)"
                    >
                        <i :class="['ti', `ti-${provider.icon || 'building-bank'}`, 'text-primary']"></i>
                        <span class="font-bold text-lg">{{ provider.name }}</span>
                    </div>
                    <div class="flex flex-column gap-2 ml-3">
                        <div
                            v-for="account in provider.accounts"
                            :key="account.id"
                            class="flex justify-content-between align-items-center p-2 border-round"
                            style="background: var(--surface-ground)"
                        >
                            <div class="flex align-items-center gap-2">
                                <i :class="['ti', `ti-${account.icon || 'wallet'}`]"></i>
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
