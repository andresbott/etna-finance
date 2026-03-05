<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import TimeBalance from './TimeBalance.vue'
import CashAccountsCard from './CashAccountsCard.vue'
import InvestmentAccountsCard from './InvestmentAccountsCard.vue'
import AccountTypesList from './AccountTypesList.vue'
import AccountDistribution from './AccountDistribution.vue'
import { useSettingsStore } from '@/store/settingsStore.js'

const leftSidebarCollapsed = ref(true)
const settings = useSettingsStore()
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="grid p-3">
                <!-- Financial Overview -->
                <div class="col-12">
                    <TimeBalance />
                </div>

                <!-- Account Types List + Account Distribution (same row, 50% each) -->
                <div class="col-12 lg:col-6">
                    <AccountTypesList />
                </div>
                <div class="col-12 lg:col-6">
                    <AccountDistribution />
                </div>

                <!-- Account Balances: Cash (left) + Investment (right) -->
                <div :class="settings.instruments ? 'col-12 lg:col-6' : 'col-12'">
                    <CashAccountsCard />
                </div>
                <div v-if="settings.instruments" class="col-12 lg:col-6">
                    <InvestmentAccountsCard />
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.card {
    height: 100%;
}
</style>
