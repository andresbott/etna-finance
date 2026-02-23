<script setup>
import Card from 'primevue/card'
import { useHoldings } from '@/composables/useHoldings'
import { formatAmount } from '@/utils/currency'
import { getAccountTypeLabel } from '@/types/account'

const { providersWithHoldings, isLoading } = useHoldings()
</script>

<template>
    <Card>
        <template #title>Financial Instruments</template>
        <template #content>
            <div v-if="providersWithHoldings.length === 0 && !isLoading" class="text-center p-3 text-500">
                No investment or unvested accounts
            </div>
            <div v-else-if="isLoading" class="text-center p-3 text-500">
                Loading holdings…
            </div>
            <div v-else class="flex flex-column gap-4">
                <div
                    v-for="provider in providersWithHoldings"
                    :key="provider.id"
                    class="flex flex-column gap-2"
                >
                    <!-- Provider Header -->
                    <div
                        class="flex align-items-center gap-2 pb-2"
                        style="border-bottom: 1px solid rgba(0, 0, 0, 0.06)"
                    >
                        <i :class="['pi', provider.icon || 'pi-building', 'text-primary']"></i>
                        <span class="font-bold text-lg">{{ provider.name }}</span>
                    </div>

                    <!-- Accounts under this provider -->
                    <div class="flex flex-column gap-3 ml-3">
                        <div
                            v-for="account in provider.accounts"
                            :key="account.id"
                            class="flex flex-column gap-2"
                        >
                            <div
                                class="flex justify-content-between align-items-center p-2 border-round"
                                style="background: var(--surface-ground)"
                            >
                                <div class="flex align-items-center gap-2">
                                    <i :class="['pi', account.icon || 'pi-chart-line']"></i>
                                    <span class="font-semibold">{{ account.name }}</span>
                                    <span class="text-500 text-sm">({{ getAccountTypeLabel(account.type) }})</span>
                                </div>
                                <div class="flex align-items-center gap-2">
                                    <span class="font-bold">{{ formatAmount(account.totalValue) }}</span>
                                    <span class="text-500">{{ account.currency }}</span>
                                </div>
                            </div>

                            <!-- Holdings (instruments) under this account -->
                            <div
                                v-if="account.holdings.length > 0"
                                class="ml-4 flex flex-column gap-1"
                            >
                                <div
                                    v-for="h in account.holdings"
                                    :key="h.instrumentId"
                                    class="flex justify-content-between align-items-center py-1 px-2 border-round text-sm"
                                    style="background: var(--surface-50)"
                                >
                                    <span class="text-600">{{ h.symbol || 'Instrument #' + h.instrumentId }}</span>
                                    <div class="flex align-items-center gap-3">
                                        <span>{{ h.quantity }} shares</span>
                                        <span class="font-semibold">{{ formatAmount(h.value) }} {{ h.currency }}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </template>
    </Card>
</template>
