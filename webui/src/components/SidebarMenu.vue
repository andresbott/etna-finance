<template>
    <div class="relative-sidebar-wrapper">
        <Transition name="slide-left">
            <div v-if="uiStore.isDrawerVisible" class="sidebar-panel">
                <ul class="list-none p-0 m-0 w-full">
                    <li>
                        <router-link
                            to="/"
                            class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                        >
                            <i class="pi pi-home mr-2"></i>
                            <span class="font-medium">Overview</span>
                        </router-link>
                    </li>

                    <li>
                        <a
                            v-styleclass="{
                                selector: '@next',
                                toggleClass: 'hidden',
                                enterFromClass: 'hidden',
                                enterActiveClass: 'animate-slidedown',
                                leaveActiveClass: 'animate-slideup',
                                leaveToClass: 'hidden'
                            }"
                            class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                        >
                            <i class="pi pi-chart-line mr-2"></i>
                            <span class="font-medium">Accounts</span>
                            <i class="pi pi-chevron-down ml-auto"></i>
                        </a>

                        <ul
                            class="list-none py-0 pl-4 pr-0 m-0 hidden transition-all duration-[400ms] ease-in-out"
                        >
                            <li v-for="provider in accounts" :key="provider.id">
                                <a
                                    v-styleclass="{
                                        selector: '@next',
                                        toggleClass: 'hidden',
                                        enterFromClass: 'hidden',
                                        enterActiveClass: 'animate-slidedown',
                                        leaveToClass: 'hidden',
                                        leaveActiveClass: 'animate-slideup'
                                    }"
                                    class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                                >
                                    <i class="pi pi-wallet mr-2"></i>
                                    <span class="font-medium">{{ provider.name }}</span>
                                    <i
                                        v-if="provider.accounts.length > 0"
                                        class="pi pi-chevron-down ml-auto transition-transform duration-300"
                                    ></i>
                                </a>

                                <ul class="list-none hidden overflow-hidden">
                                    <li v-for="account in provider.accounts" :key="account.id">
                                        <router-link
                                            :to="`/entries/${account.id}`"
                                            class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                                        >
                                            <i
                                                :class="getAccountIcon(account.type)"
                                                class="mr-2"
                                            ></i>
                                            <div class="flex flex-col">
                                                <span class="font-medium">
                                                    {{ account.name }}
                                                </span>
                                            </div>
                                        </router-link>
                                    </li>
                                </ul>
                            </li>
                        </ul>
                    </li>

                    <li>
                        <a
                            v-styleclass="{
                                selector: '@next',
                                toggleClass: 'hidden',
                                enterFromClass: 'hidden',
                                enterActiveClass: 'animate-slidedown',
                                leaveActiveClass: 'animate-slideup',
                                leaveToClass: 'hidden'
                            }"
                            class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                        >
                            <i class="pi pi-chart-line mr-2"></i>
                            <span class="font-medium">Reports</span>
                            <i class="pi pi-chevron-down ml-auto"></i>
                        </a>

                        <ul
                            class="list-none py-0 pl-4 pr-0 m-0 hidden transition-all duration-[400ms] ease-in-out"
                        >
                            <li>
                                <router-link
                                    to="/reports"
                                    class="flex items-center cursor-pointer px-4 py-3 hover:bg-gray-100"
                                >
                                    <i class="pi pi-arrow-up-right mr-2"></i>
                                    <span class="font-medium">Expense & Income</span>
                                </router-link>
                            </li>
                        </ul>
                    </li>
                </ul>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { useUiStore } from '@/store/uiStore.js'
import { useAccounts } from '@/composables/useAccounts.js'

const uiStore = useUiStore()
const { accounts } = useAccounts()

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
    background: var(--c-card-background);
    transition: transform 0.3s;
    overflow-y: auto;
    padding: 20px 10px;
}

.slide-left-enter-from,
.slide-left-leave-to {
    transform: translateX(-100%);
}

.slide-left-enter-active,
.slide-left-leave-active {
    transition: transform 0.3s ease-out;
}

i {
    line-height: unset !important;
}
</style>
