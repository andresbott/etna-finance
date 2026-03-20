<script setup lang="ts">
import { ref } from 'vue'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import ExpenseCategoriesView from '@/views/categories/ExpenseCategoriesView.vue'
import IncomeCategoriesView from '@/views/categories/IncomeCategoriesView.vue'

const activeTab = ref('expense')
</script>

<template>
    <div>
        <div class="flex justify-content-between align-items-start mb-4 gap-3">
            <div>
                <h1 class="text-2xl font-bold mb-2 text-color">Categories</h1>
                <p class="text-color-secondary m-0 text-base">
                    Manage expense and income categories to organize your transactions
                </p>
            </div>
        </div>
        <Tabs v-model:value="activeTab">
            <TabList>
                <Tab value="expense">Expense</Tab>
                <Tab value="income">Income</Tab>
            </TabList>
            <TabPanels>
                <TabPanel value="expense">
                    <ExpenseCategoriesView />
                </TabPanel>
                <TabPanel value="income">
                    <IncomeCategoriesView />
                </TabPanel>
            </TabPanels>
        </Tabs>
    </div>
</template>
