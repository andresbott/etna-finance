<template>
    <Drawer v-model:visible="uiStore.isDrawerVisible" position="left" header=" ">
        <template #default>
            <div class="flex flex-col h-full">
                <ul class="list-none p-0 m-0 overflow-y-auto w-full h-full">
                    <!-- Overview -->
                    <li>
                        <router-link to="/" class="flex items-center cursor-pointer px-4 py-3">
                            <i class="pi pi-home mr-2"></i>
                            <span class="font-medium">Overview</span>
                        </router-link>
                    </li>

                    <!-- Accounts -->
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
                            class="flex items-center cursor-pointer px-4 py-3"
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
                                    class="flex items-center cursor-pointer px-4 py-3"
                                >
                                    <i class="pi pi-wallet mr-2"></i>
                                    <span class="font-medium">{{ provider.name }}</span>
                                    <i
                                        v-if="provider.accounts.length > 0"
                                        class="pi pi-chevron-down ml-auto transition-transform duration-300"
                                    ></i>
                                </a>

                                <!-- Accounts Submenu -->
                                <ul class="list-none hidden overflow-hidden">
                                    <li v-for="account in provider.accounts" :key="account.id">
                                        <router-link
                                            :to="`/entries/${account.id}`"
                                            class="flex items-center cursor-pointer px-4 py-3"
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

                    <!-- Reports -->
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
                            class="flex items-center cursor-pointer px-4 py-3"
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
                                    class="flex items-center cursor-pointer px-4 py-3"
                                >
                                    <i class="pi pi-arrow-up-right mr-2"></i>
                                    <span class="font-medium">Expense</span>
                                </router-link>
                            </li>
                            <li>
                                <router-link
                                    to="/reports"
                                    class="flex items-center cursor-pointer px-4 py-3"
                                >
                                    <i class="pi pi-arrow-down-right mr-2"></i>
                                    <span class="font-medium">Income</span>
                                </router-link>
                            </li>
                        </ul>
                    </li>
                </ul>
            </div>
        </template>
    </Drawer>
</template>

<script setup>
import { Drawer } from 'primevue'
import { useUiStore } from '@/store/uiStore.js'
import { useAccounts } from '@/composables/useAccounts.js'
import { onMounted, onUnmounted } from 'vue'

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

const checkScreenWidth = () => {
    if (window.innerWidth >= 1024) {
        useUiStore().closeDrawer()
    }
}

onMounted(() => {
    checkScreenWidth()
    window.addEventListener('resize', checkScreenWidth)
})

onUnmounted(() => {
    window.removeEventListener('resize', checkScreenWidth)
})
</script>

<style scoped>
i {
    line-height: unset !important;
}
</style>
