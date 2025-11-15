<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref } from 'vue'
import TimeBalance from './TimeBalance.vue'
import AccountsList from './AccountsList.vue'
import AccountTypesList from './AccountTypesList.vue'
import AccountDistribution from './AccountDistribution.vue'

const leftSidebarCollapsed = ref(true)
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="grid p-3">
                <!-- Financial Overview -->
                <div class="col-12">
                    <TimeBalance />
                </div>

                <!-- Account Types List -->
                <div class="col-12">
                    <AccountTypesList />
                </div>

                <!-- Accounts List -->
                <div class="col-12 lg:col-6">
                    <AccountsList />
                </div>

                <!-- Account Distribution -->
                <div class="col-12 lg:col-6">
                    <AccountDistribution />
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
