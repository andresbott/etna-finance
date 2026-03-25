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
                            <i class="ti ti-home menu-icon"></i>
                            <span class="menu-label">Overview</span>
                        </router-link>
                    </li>
                    <li>
                        <router-link to="/reports/balances" class="menu-item">
                            <i class="ti ti-wallet menu-icon"></i>
                            <span class="menu-label">Balances</span>
                        </router-link>
                    </li>
                    <li>
                        <router-link to="/reports/income-expense" class="menu-item">
                            <i class="ti ti-chart-line menu-icon"></i>
                            <span class="menu-label">Income/Expense</span>
                        </router-link>
                    </li>
                    <li v-if="settings.investmentInstruments">
                        <router-link to="/reports/investment" class="menu-item">
                            <i class="ti ti-chart-pie menu-icon"></i>
                            <span class="menu-label">Open Positions</span>
                        </router-link>
                    </li>
                    <li class="menu-spacer"></li>

                    <!-- TRANSACTIONS SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Transactions</div>
                    </li>
                    <li>
                        <router-link to="/entries" class="menu-item">
                            <i class="ti ti-layout-grid menu-icon"></i>
                            <span class="menu-label">All Accounts</span>
                        </router-link>
                    </li>
                    <li v-for="fav in favoriteAccounts" :key="'fav-' + fav.id">
                        <router-link :to="`/entries/${fav.id}`" class="menu-item favorite-item">
                            <i :class="['ti', `ti-${fav.icon || 'wallet'}`, 'menu-icon']"></i>
                            <span class="menu-label">{{ fav.name }}</span>
                        </router-link>
                    </li>
                    <li>
                        <router-link to="/nav/account-browser" class="menu-item">
                            <i class="ti ti-list-search menu-icon"></i>
                            <span class="menu-label">Account Browser</span>
                        </router-link>
                    </li>

                    <li class="menu-spacer"></li>

                    <!-- MARKET DATA SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Market Data</div>
                    </li>
                    <li>
                        <router-link to="/market-data/currency-exchange" class="menu-item">
                            <i class="ti ti-currency-dollar menu-icon"></i>
                            <span class="menu-label">Currency Exchange</span>
                        </router-link>
                    </li>
                    <li v-if="settings.investmentInstruments">
                        <router-link to="/market-data/stock-market" class="menu-item">
                            <i class="ti ti-chart-line menu-icon"></i>
                            <span class="menu-label">Stock Market</span>
                        </router-link>
                    </li>

                    <template v-if="settings.financialSimulator">
                        <li class="menu-spacer"></li>

                        <!-- TOOLS SECTION -->
                        <li class="menu-section">
                            <div class="menu-section-label">Tools</div>
                        </li>
                        <li>
                            <router-link to="/financial-simulator" class="menu-item">
                                <i class="ti ti-calculator menu-icon"></i>
                                <span class="menu-label">Financial Simulator</span>
                            </router-link>
                        </li>
                    </template>

                    <li class="menu-spacer"></li>

                    <!-- SYSTEM SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">System</div>
                    </li>
                    <li>
                        <router-link to="/settings" class="menu-item">
                            <i class="ti ti-settings menu-icon"></i>
                            <span class="menu-label">Settings</span>
                        </router-link>
                    </li>
                    <li>
                        <router-link to="/docs" class="menu-item">
                            <i class="ti ti-book menu-icon"></i>
                            <span class="menu-label">Documentation</span>
                        </router-link>
                    </li>
                </ul>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUiStore } from '@/store/uiStore.js'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore.js'

const uiStore = useUiStore()
const settings = useSettingsStore()
const { accounts } = useAccounts()

// Flat list of all favorited accounts across all providers
const favoriteAccounts = computed(() => {
    const list = accounts.value
    if (!list) return []
    return list.flatMap(provider =>
        (provider.accounts || [])
            .filter(acc => acc.favorite)
            .map(acc => ({ ...acc, providerIcon: provider.icon }))
    )
})
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
    font-size: 1.35rem;
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
