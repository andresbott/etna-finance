<template>
    <div class="relative-sidebar-wrapper">
        <Transition name="slide-left">
            <div v-if="uiStore.isDrawerVisible" class="sidebar-panel">
                <ul class="menu-list">
                    <!-- TRANSACTIONS SECTION -->
                    <li class="menu-section">
                        <div class="menu-section-label">Transactions</div>
                    </li>
                    <li>
                        <router-link to="/entries" class="menu-item">
                            <i class="pi pi-list menu-icon"></i>
                            <span class="menu-label">All</span>
                        </router-link>
                    </li>
                    <li>
                        <a
                            @click="expandAllAccounts"
                            class="menu-item"
                        >
                            <i class="pi pi-wallet menu-icon"></i>
                            <span class="menu-label">Accounts</span>
                            <i 
                                class="pi pi-chevron-down menu-toggle" 
                                :class="{ 'rotate-180': isMyAccountsExpanded }"
                            ></i>
                        </a>

                        <ul class="menu-submenu" :class="{ hidden: !isMyAccountsExpanded }">
                            <li v-for="provider in accounts" :key="provider.id">
                                <a
                                    @click="toggleProvider(provider.id)"
                                    class="menu-item submenu-item"
                                >
                                    <i class="pi pi-building menu-icon"></i>
                                    <span class="menu-label">{{ provider.name }}</span>
                                    <i
                                        v-if="provider.accounts.length > 0"
                                        class="pi pi-chevron-down menu-toggle"
                                        :class="{ 'rotate-180': expandedProviders[provider.id] }"
                                    ></i>
                                </a>

                                <ul 
                                    class="menu-submenu" 
                                    :class="{ hidden: !expandedProviders[provider.id] }"
                                >
                                    <li v-for="account in provider.accounts" :key="account.id">
                                        <router-link
                                            :to="`/entries/${account.id}`"
                                            class="menu-item submenu-item"
                                        >
                                            <i :class="getAccountIcon(account.type)" class="menu-icon"></i>
                                            <span class="menu-label">{{ account.name }}</span>
                                        </router-link>
                                    </li>
                                </ul>
                            </li>
                        </ul>
                    </li>

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
                </ul>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useUiStore } from '@/store/uiStore.js'
import { useAccounts } from '@/composables/useAccounts.js'

const uiStore = useUiStore()
const { accounts } = useAccounts()

const isMyAccountsExpanded = ref(false)
const expandedProviders = reactive({})

// Initialize expandedProviders when accounts are loaded
watch(accounts, (newAccounts) => {
    if (newAccounts) {
        newAccounts.forEach(provider => {
            if (!(provider.id in expandedProviders)) {
                expandedProviders[provider.id] = false
            }
        })
    }
}, { immediate: true })

const expandAllAccounts = () => {
    if (!isMyAccountsExpanded.value) {
        // Expanding: expand My Accounts and all providers
        isMyAccountsExpanded.value = true
        accounts.value?.forEach(provider => {
            expandedProviders[provider.id] = true
        })
    } else {
        // Collapsing: just toggle My Accounts
        isMyAccountsExpanded.value = false
    }
}

const toggleProvider = (providerId) => {
    expandedProviders[providerId] = !expandedProviders[providerId]
}

const getAccountIcon = (type) => {
    const icons = {
        cash: 'pi pi-money-bill',
        bank: 'pi pi-building',
        investment: 'pi pi-chart-line',
        credit: 'pi pi-credit-card',
        savings: 'pi pi-piggy-bank'
    }
    return icons[type] || 'pi pi-wallet'
}
</script>

<style scoped>
.sidebar-panel {
    position: relative;
    height: 100%;
    width: 300px;
    background: var(--p-surface-0);
    border-right: 1px solid var(--p-surface-border);
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
    color: var(--p-text-muted-color);
    padding: 0.5rem 1.5rem;
    margin-bottom: 0.25rem;
}

/* Menu Items */
.menu-item {
    display: flex;
    align-items: center;
    padding: 0.75rem 1.5rem;
    color: var(--p-text-color);
    text-decoration: none;
    cursor: pointer;
    transition: all 0.2s ease;
    border-left: 3px solid transparent;
}

.menu-item:hover {
    background-color: var(--p-surface-hover);
    color: var(--p-primary-color);
}

.menu-item.router-link-active {
    background-color: var(--p-primary-50);
    color: var(--p-primary-color);
    font-weight: 600;
    border-left-color: var(--p-primary-color);
}

/* Submenu Items */
.submenu-item {
    padding-left: 3rem;
}

.menu-submenu {
    list-style: none;
    padding: 0;
    margin: 0;
    overflow: hidden;
    transition: all 0.4s ease-in-out;
}

.menu-submenu .menu-submenu .submenu-item {
    padding-left: 4.5rem;
}

/* Icons */
.menu-icon {
    margin-right: 0.75rem;
    font-size: 1.125rem;
    line-height: 1 !important;
    color: var(--p-text-secondary-color);
    transition: color 0.2s ease;
}

.menu-item:hover .menu-icon,
.menu-item.router-link-active .menu-icon {
    color: var(--p-primary-color);
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
