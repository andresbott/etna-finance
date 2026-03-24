<script setup>
import { ref, computed, watch } from 'vue'
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Card from 'primevue/card'
import Divider from 'primevue/divider'
import TreeTable from 'primevue/treetable'
import Column from 'primevue/column'
import { useAccounts } from '@/composables/useAccounts'
import { useBalance } from '@/composables/useGetBalanceReport'
import { useHoldings } from '@/composables/useHoldings'
import { useRouter } from 'vue-router'
import { formatAmount } from '@/utils/currency'
import { getAccountTypeLabel, getAccountTypeIcon, ACCOUNT_TYPES } from '@/types/account'
const router = useRouter()
const leftSidebarCollapsed = ref(true)

const CASH_TYPES = [ACCOUNT_TYPES.CASH, ACCOUNT_TYPES.CHECKING, ACCOUNT_TYPES.SAVINGS, ACCOUNT_TYPES.LENT]
const INVESTMENT_TYPES = [ACCOUNT_TYPES.INVESTMENT, ACCOUNT_TYPES.RESTRICTED_STOCK]
const PENSION_TYPES = [ACCOUNT_TYPES.PENSION]

const { accounts: accountProviders } = useAccounts()
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation
const { providersWithHoldings } = useHoldings()

// Fetch balance report for all accounts
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
                mutate({ accountIds, steps: 30, startDate: '2020-01-01' })
            }
        }
    },
    { immediate: true }
)

// Map accountId -> totalValue from holdings (for investment/restricted stock)
const holdingsTotalMap = computed(() => {
    const map = new Map()
    for (const provider of providersWithHoldings.value) {
        for (const account of provider.accounts) {
            map.set(account.id, { totalValue: account.totalValue, currency: account.currency })
        }
    }
    return map
})

const getLatestBalance = (reportData) => {
    if (!reportData || reportData.length === 0) return 0
    return reportData[reportData.length - 1]?.sum || 0
}

// Build tree nodes (provider -> accounts) filtered by account types
const buildTreeNodes = (types) => {
    if (!accountProviders.value) return []
    const nodes = []
    for (const provider of accountProviders.value) {
        const children = []
        for (const account of (provider.accounts || [])) {
            if (!types.includes(account.type)) continue
            let balance = 0
            let currency = account.currency || 'CHF'

            if (INVESTMENT_TYPES.includes(account.type)) {
                const holding = holdingsTotalMap.value.get(account.id)
                if (holding) {
                    balance = holding.totalValue
                    currency = holding.currency || currency
                }
            } else {
                const reportData = balanceReport.value?.accounts?.[account.id]
                balance = getLatestBalance(reportData)
            }

            children.push({
                key: `acc-${account.id}`,
                data: {
                    isProvider: false,
                    accountId: account.id,
                    name: account.name,
                    icon: account.icon || 'wallet',
                    type: account.type,
                    currency,
                    balance
                }
            })
        }
        if (children.length > 0) {
            nodes.push({
                key: `prov-${provider.id}`,
                data: {
                    isProvider: true,
                    name: provider.name,
                    icon: provider.icon || 'building-bank'
                },
                children
            })
        }
    }
    return nodes
}

const cashNodes = computed(() => buildTreeNodes(CASH_TYPES))
const investmentNodes = computed(() => buildTreeNodes(INVESTMENT_TYPES))
const pensionNodes = computed(() => buildTreeNodes(PENSION_TYPES))

// Expanded keys for all sections
const expandedKeys = (nodes) => {
    const keys = {}
    for (const node of nodes) {
        keys[node.key] = true
    }
    return keys
}

const cashExpanded = computed(() => expandedKeys(cashNodes.value))
const investmentExpanded = computed(() => expandedKeys(investmentNodes.value))
const pensionExpanded = computed(() => expandedKeys(pensionNodes.value))

const hasAnyAccounts = computed(() =>
    cashNodes.value.length > 0 || investmentNodes.value.length > 0 || pensionNodes.value.length > 0
)

const sectionTotal = (nodes) => {
    const map = {}
    for (const node of nodes) {
        for (const child of node.children || []) {
            const { currency, balance } = child.data
            map[currency] = (map[currency] || 0) + balance
        }
    }
    return Object.entries(map).map(([currency, total]) => ({ currency, total }))
}

</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3 flex flex-column gap-3">
                <div>
                    <h1 class="flex align-items-center gap-3 m-0 mb-2">
                        <i class="ti ti-wallet text-primary"></i>
                        All Account Balances
                    </h1>
                </div>

                <Card>
                    <template #content>
                        <div v-if="!hasAnyAccounts" class="text-center p-4 text-500">
                            No accounts configured
                        </div>

                        <template v-else>
                            <!-- Cash Accounts -->
                            <div v-if="cashNodes.length > 0" class="mb-4 balance-section">
                                <TreeTable :value="cashNodes" :expandedKeys="cashExpanded">
                                    <Column style="min-width: 250px">
                                        <template #header>
                                            <span class="flex align-items-center gap-2">
                                                <i class="ti ti-cash"></i>Cash Accounts
                                            </span>
                                        </template>
                                        <template #body="{ node }">
                                            <div class="flex align-items-center gap-2" :style="node.data.isProvider ? '' : 'padding-left: 2rem'">
                                                <i :class="['ti', `ti-${node.data.icon}`, node.data.isProvider ? 'text-primary' : '']"></i>
                                                <span v-if="node.data.isProvider" class="font-bold text-lg">{{ node.data.name }}</span>
                                                <a v-else class="font-semibold cursor-pointer hover:underline" @click="router.push(`/entries/${node.data.accountId}`)">{{ node.data.name }}</a>
                                            </div>
                                        </template>
                                    </Column>
                                    <Column header="" style="min-width: 130px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <div class="flex align-items-center gap-2">
                                                    <i :class="['ti', `ti-${getAccountTypeIcon(node.data.type)}`]" class="text-500"></i>
                                                    <span class="text-500">{{ getAccountTypeLabel(node.data.type) }}</span>
                                                </div>
                                            </template>
                                        </template>
                                    </Column>
                                    <Column header="" class="bal-col" style="min-width: 150px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <span class="font-semibold" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ formatAmount(node.data.balance) }}</span>
                                                <span :class="['ml-1', node.data.balance < 0 ? '' : 'text-500']" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ node.data.currency }}</span>
                                            </template>
                                        </template>
                                    </Column>
                                </TreeTable>
                                <div class="flex justify-content-end gap-3 mt-2">
                                    <span v-for="t in sectionTotal(cashNodes)" :key="t.currency" class="font-semibold">
                                        {{ formatAmount(t.total) }} {{ t.currency }}
                                    </span>
                                </div>
                            </div>

                            <Divider v-if="cashNodes.length > 0 && investmentNodes.length > 0" class="balance-divider" />

                            <!-- Investment Accounts -->
                            <div v-if="investmentNodes.length > 0" class="mb-4 balance-section">
                                <TreeTable :value="investmentNodes" :expandedKeys="investmentExpanded">
                                    <Column style="min-width: 250px">
                                        <template #header>
                                            <span class="flex align-items-center gap-2">
                                                <i class="ti ti-chart-pie"></i>Investment Accounts
                                            </span>
                                        </template>
                                        <template #body="{ node }">
                                            <div class="flex align-items-center gap-2" :style="node.data.isProvider ? '' : 'padding-left: 2rem'">
                                                <i :class="['ti', `ti-${node.data.icon}`, node.data.isProvider ? 'text-primary' : '']"></i>
                                                <span v-if="node.data.isProvider" class="font-bold text-lg">{{ node.data.name }}</span>
                                                <a v-else class="font-semibold cursor-pointer hover:underline" @click="router.push(`/entries/${node.data.accountId}`)">{{ node.data.name }}</a>
                                            </div>
                                        </template>
                                    </Column>
                                    <Column header="" style="min-width: 130px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <div class="flex align-items-center gap-2">
                                                    <i :class="['ti', `ti-${getAccountTypeIcon(node.data.type)}`]" class="text-500"></i>
                                                    <span class="text-500">{{ getAccountTypeLabel(node.data.type) }}</span>
                                                </div>
                                            </template>
                                        </template>
                                    </Column>
                                    <Column header="" class="bal-col" style="min-width: 150px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <span class="font-semibold" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ formatAmount(node.data.balance) }}</span>
                                                <span :class="['ml-1', node.data.balance < 0 ? '' : 'text-500']" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ node.data.currency }}</span>
                                            </template>
                                        </template>
                                    </Column>
                                </TreeTable>
                                <div class="flex justify-content-end gap-3 mt-2">
                                    <span v-for="t in sectionTotal(investmentNodes)" :key="t.currency" class="font-semibold">
                                        {{ formatAmount(t.total) }} {{ t.currency }}
                                    </span>
                                </div>
                            </div>

                            <Divider v-if="(cashNodes.length > 0 || investmentNodes.length > 0) && pensionNodes.length > 0" class="balance-divider" />

                            <!-- Pension Accounts -->
                            <div v-if="pensionNodes.length > 0" class="mb-4 balance-section">
                                <TreeTable :value="pensionNodes" :expandedKeys="pensionExpanded">
                                    <Column style="min-width: 250px">
                                        <template #header>
                                            <span class="flex align-items-center gap-2">
                                                <i class="ti ti-building-bank"></i>Pension Accounts
                                            </span>
                                        </template>
                                        <template #body="{ node }">
                                            <div class="flex align-items-center gap-2" :style="node.data.isProvider ? '' : 'padding-left: 2rem'">
                                                <i :class="['ti', `ti-${node.data.icon}`, node.data.isProvider ? 'text-primary' : '']"></i>
                                                <span v-if="node.data.isProvider" class="font-bold text-lg">{{ node.data.name }}</span>
                                                <a v-else class="font-semibold cursor-pointer hover:underline" @click="router.push(`/entries/${node.data.accountId}`)">{{ node.data.name }}</a>
                                            </div>
                                        </template>
                                    </Column>
                                    <Column header="" style="min-width: 130px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <div class="flex align-items-center gap-2">
                                                    <i :class="['ti', `ti-${getAccountTypeIcon(node.data.type)}`]" class="text-500"></i>
                                                    <span class="text-500">{{ getAccountTypeLabel(node.data.type) }}</span>
                                                </div>
                                            </template>
                                        </template>
                                    </Column>
                                    <Column header="" class="bal-col" style="min-width: 150px">
                                        <template #body="{ node }">
                                            <template v-if="!node.data.isProvider">
                                                <span class="font-semibold" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ formatAmount(node.data.balance) }}</span>
                                                <span :class="['ml-1', node.data.balance < 0 ? '' : 'text-500']" :style="node.data.balance < 0 ? { color: 'var(--c-red-600)' } : {}">{{ node.data.currency }}</span>
                                            </template>
                                        </template>
                                    </Column>
                                </TreeTable>
                                <div class="flex justify-content-end gap-3 mt-2">
                                    <span v-for="t in sectionTotal(pensionNodes)" :key="t.currency" class="font-semibold">
                                        {{ formatAmount(t.total) }} {{ t.currency }}
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
.balance-divider {
    margin: 2.8125rem 0;
}
</style>

<style>
.balance-section .p-treetable-node-toggle-button {
    display: none !important;
}

.balance-section .bal-col .p-treetable-column-header-content {
    justify-content: flex-end;
}

.balance-section td.bal-col .p-treetable-body-cell-content {
    justify-content: flex-end;
}
</style>
