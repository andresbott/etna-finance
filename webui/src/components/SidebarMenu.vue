<template>
    <div class="relative-sidebar-wrapper">
        <Transition name="slide-left">
            <div v-if="uiStore.isDrawerVisible" class="sidebar-panel">
                <ul class="menu-list">
                    <!-- REPORTS SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Reports</div>
                    </li>
                    <li>
                        <router-link to="/reports/overview" class="menu-item">
                            <i class="pi pi-home menu-icon"></i>
                            <span class="menu-label">Overview</span>
                        </router-link>
                    </li>
                    <li>
                        <router-link to="/reports/income-expense" class="menu-item">
                            <i class="pi pi-chart-line menu-icon"></i>
                            <span class="menu-label">Income/Expense</span>
                        </router-link>
                    </li>
                    <li v-if="settings.instruments">
                        <router-link to="/reports/investment" class="menu-item">
                            <i class="pi pi-chart-pie menu-icon"></i>
                            <span class="menu-label">Investment Report</span>
                        </router-link>
                    </li>

                    <li class="menu-spacer"></li>

                    <!-- TRANSACTIONS SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Transactions</div>
                    </li>
                    <li>
                        <router-link to="/entries" class="menu-item">
                            <i class="pi pi-th-large menu-icon"></i>
                            <span class="menu-label">All Transactions</span>
                        </router-link>
                    </li>
                    <li>
                        <a
                            @click="expandAllAccounts"
                            class="menu-item"
                        >
                            <i class="pi pi-filter menu-icon"></i>
                            <span class="menu-label">Cash accounts</span>
                            <i 
                                class="pi pi-chevron-down menu-toggle" 
                                :class="{ 'rotate-180': isMyAccountsExpanded }"
                            ></i>
                        </a>

                        <ul class="menu-submenu" :class="{ hidden: !isMyAccountsExpanded }">
                            <li v-for="provider in accountsCashOnly" :key="provider.id">
                                <div class="menu-item submenu-item menu-item-label">
                                    <i :class="['pi', provider.icon || 'pi-building', 'menu-icon']"></i>
                                    <span class="menu-label">{{ provider.name }}</span>
                                </div>

                                <ul class="menu-submenu">
                                    <li v-for="account in provider.accounts" :key="account.id">
                                        <router-link
                                            :to="`/entries/${account.id}`"
                                            class="menu-item submenu-item"
                                        >
                                            <i :class="['pi', account.icon || 'pi-wallet', 'menu-icon']"></i>
                                            <span class="menu-label">{{ account.name }}</span>
                                        </router-link>
                                    </li>
                                </ul>
                            </li>
                        </ul>
                    </li>
                    <li>
                        <router-link to="/financial-transactions" class="menu-item">
                            <i class="pi pi-wallet menu-icon"></i>
                            <span class="menu-label">Financial Transactions</span>
                        </router-link>
                    </li>
                    <li v-if="settings.instruments">
                        <a
                            @click="expandAllInvestment"
                            class="menu-item"
                        >
                            <i class="pi pi-chart-line menu-icon"></i>
                            <span class="menu-label">Investment</span>
                            <i
                                class="pi pi-chevron-down menu-toggle"
                                :class="{ 'rotate-180': isInvestmentExpanded }"
                            ></i>
                        </a>

                        <ul class="menu-submenu" :class="{ hidden: !isInvestmentExpanded }">
                            <li v-for="provider in accountsInvestmentOnly" :key="provider.id">
                                <div class="menu-item submenu-item menu-item-label">
                                    <i :class="['pi', provider.icon || 'pi-building', 'menu-icon']"></i>
                                    <span class="menu-label">{{ provider.name }}</span>
                                </div>

                                <ul class="menu-submenu">
                                    <li v-for="account in provider.accounts" :key="account.id">
                                        <router-link
                                            :to="`/entries/${account.id}`"
                                            class="menu-item submenu-item"
                                        >
                                            <i :class="['pi', account.icon || 'pi-wallet', 'menu-icon']"></i>
                                            <span class="menu-label">{{ account.name }}</span>
                                        </router-link>
                                    </li>
                                </ul>
                            </li>
                        </ul>
                    </li>

                    <li class="menu-spacer"></li>

                    <!-- MARKET DATA SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Market Data</div>
                    </li>
                    <li>
                        <router-link to="/market-data/currency-exchange" class="menu-item">
                            <i class="pi pi-dollar menu-icon"></i>
                            <span class="menu-label">Currency Exchange</span>
                        </router-link>
                    </li>
                    <li v-if="settings.instruments">
                        <router-link to="/market-data/stock-market" class="menu-item">
                            <i class="pi pi-chart-line menu-icon"></i>
                            <span class="menu-label">Stock Market</span>
                        </router-link>
                    </li>

                    <li class="menu-spacer"></li>

                    <!-- TOOLS SECTION -->
                    <template v-if="settings.tools">
                        <li class="menu-section">
                            <div class="menu-section-label">Tools</div>
                        </li>
                        <li>
                            <router-link to="/financial-simulator" class="menu-item">
                                <i class="pi pi-calculator menu-icon"></i>
                                <span class="menu-label">Financial Simulator</span>
                            </router-link>
                        </li>
                    </template>

                    <li class="menu-spacer"></li>

                    <!-- SETTINGS SECTION -->
                    <li>
                        <router-link to="/settings" class="menu-item">
                            <i class="pi pi-cog menu-icon"></i>
                            <span class="menu-label">System</span>
                        </router-link>
                    </li>
                </ul>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/store/uiStore.js'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore.js'
import { ACCOUNT_TYPES } from '@/types/account'

const route = useRoute()
const uiStore = useUiStore()
const settings = useSettingsStore()
const { accounts } = useAccounts()

const CASH_ACCOUNT_TYPES = [ACCOUNT_TYPES.CASH, ACCOUNT_TYPES.CHECKING, ACCOUNT_TYPES.SAVINGS, ACCOUNT_TYPES.LENT]
const INVESTMENT_ACCOUNT_TYPES = [ACCOUNT_TYPES.INVESTMENT, ACCOUNT_TYPES.UNVESTED]

// By Account tree: only cash accounts (cash, checking, savings)
const accountsCashOnly = computed(() => {
    const list = accounts.value
    if (!list) return []
    return list
        .map(provider => ({
            ...provider,
            accounts: (provider.accounts || []).filter(acc =>
                CASH_ACCOUNT_TYPES.includes(acc.type)
            )
        }))
        .filter(provider => provider.accounts.length > 0)
})

// Investment section: only financial investment accounts (investment, unvested)
const accountsInvestmentOnly = computed(() => {
    const list = accounts.value
    if (!list) return []
    return list
        .map(provider => ({
            ...provider,
            accounts: (provider.accounts || []).filter(acc =>
                INVESTMENT_ACCOUNT_TYPES.includes(acc.type)
            )
        }))
        .filter(provider => provider.accounts.length > 0)
})

const isMyAccountsExpanded = ref(false)
const isInvestmentExpanded = ref(false)

// Watch route to auto-expand when viewing account entries
watch(() => route.path, (newPath) => {
    const accountEntriesMatch = newPath.match(/^\/entries\/(\d+)$/)
    if (!accountEntriesMatch) return

    const accountId = accountEntriesMatch[1]

    const inCash = accountsCashOnly.value.some(provider =>
        provider.accounts.some(account => String(account.id) === accountId)
    )
    if (inCash) isMyAccountsExpanded.value = true

    const inInvestment = accountsInvestmentOnly.value.some(provider =>
        provider.accounts.some(account => String(account.id) === accountId)
    )
    if (inInvestment) isInvestmentExpanded.value = true
}, { immediate: true })

const expandAllAccounts = () => {
    isMyAccountsExpanded.value = !isMyAccountsExpanded.value
}

const expandAllInvestment = () => {
    isInvestmentExpanded.value = !isInvestmentExpanded.value
}
</script>

<style scoped>
.sidebar-panel {
    position: relative;
    height: 100%;
    width: 300px;
    background: var(--c-primary-500);
    border-right: 1px solid var(--c-primary-600);
    transition: transform 0.3s;
    overflow-y: auto;
    padding: 1rem 0;
}

.menu-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

/* Section Labels */
.menu-section {
    margin-top: 1.5rem;
}

.menu-section:first-child {
    margin-top: 0.5rem;
}

.menu-section-label {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--c-primary-50);
    padding: 0.5rem 1.5rem;
    margin-bottom: 0.25rem;
}

/* Section Spacer */
.menu-spacer {
    height: 1px;
    margin: 1rem 1.5rem;
    background-color: var(--c-primary-400);
}

/* Menu Items */
.menu-item {
    display: flex;
    align-items: center;
    padding: 0.75rem 1.5rem;
    color: white;
    text-decoration: none;
    cursor: pointer;
    transition: all 0.2s ease;
    border-left: 3px solid transparent;
}

.menu-item:hover {
    background-color: var(--c-primary-400);
    color: white;
}

.menu-item.router-link-active {
    background-color: var(--c-primary-300);
    color: var(--c-primary-900);
    font-weight: 600;
    border-left-color: var(--c-primary-50);
}

/* Submenu Items */
.submenu-item {
    padding-left: 3rem;
}

.menu-item-label {
    cursor: default;
}

.menu-submenu {
    list-style: none;
    padding: 0;
    margin: 0;
    overflow: hidden;
    transition: all 0.4s ease-in-out;
    background-color: var(--c-primary-500);
}

.menu-submenu .menu-submenu {
    background-color: var(--c-primary-600);
}

.menu-submenu .menu-submenu .submenu-item {
    padding-left: 4.5rem;
}

/* Icons */
.menu-icon {
    margin-right: 0.75rem;
    font-size: 1.125rem;
    line-height: 1 !important;
    color: var(--c-primary-50);
    transition: all 0.2s ease;
}

.menu-item:hover .menu-icon {
    color: white;
}

.menu-item.router-link-active .menu-icon {
    color: var(--c-primary-900);
}

.menu-label {
    flex: 1;
    font-size: 0.9375rem;
}

.menu-toggle {
    margin-left: auto;
    font-size: 0.875rem;
    transition: transform 0.3s ease;
    line-height: 1 !important;
    color: var(--c-primary-50);
}

.rotate-180 {
    transform: rotate(180deg);
}

/* Animations */
.slide-left-enter-from,
.slide-left-leave-to {
    transform: translateX(-100%);
}

.slide-left-enter-active,
.slide-left-leave-active {
    transition: transform 0.3s ease-out;
}

@keyframes slidedown {
    from {
        max-height: 0;
        opacity: 0;
    }
    to {
        max-height: 1000px;
        opacity: 1;
    }
}

@keyframes slideup {
    from {
        max-height: 1000px;
        opacity: 1;
    }
    to {
        max-height: 0;
        opacity: 0;
    }
}

.animate-slidedown {
    animation: slidedown 0.4s ease-in-out;
}

.animate-slideup {
    animation: slideup 0.4s ease-in-out;
}
</style>
