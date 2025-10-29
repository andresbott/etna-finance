<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { VerticalLayout, SidebarContent } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import { fetchIncomeExpense } from '@/composables/useIncomeExpense.js'

import TopBar from '@/views/topbar.vue'
import Footer from '@/views/parts/Footer.vue'
import DatePicker from 'primevue/datepicker'

import { Button, Column, TreeTable } from 'primevue'

const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const incomeTableData = ref([])
const expenseTableData = ref([])
const incomeVisible = ref(true)
const expenseVisible = ref(true)

const reportData = ref(null)
const leftSidebarCollapsed = ref(true)

const fetchReportData = async () => {
    try {
        const response = await fetchIncomeExpense(
            formatDateForAPI(startDate.value),
            formatDateForAPI(endDate.value)
        )

        reportData.value = await response

        incomeTableData.value = incomeNodes.value
        expenseTableData.value = expenseNodes.value
    } catch (error) {
        console.error('Error fetching report data:', error)
    }
}

const formatAmount = (amount) =>
    amount.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

const formatDateForAPI = (date) => {
    if (!date) return ''

    const dateObj = new Date(date)

    const year = dateObj.getFullYear()
    const month = String(dateObj.getMonth() + 1).padStart(2, '0')
    const day = String(dateObj.getDate()).padStart(2, '0')

    return `${year}-${month}-${day}`
}

const currencies = computed(() => {
    if (!reportData.value) return []
    const currencySet = new Set()
    const income = reportData.value.income || []
    const expenses = reportData.value.expenses || []
    ;[...income, ...expenses].forEach((item) => {
        Object.keys(item.values || {}).forEach((c) => currencySet.add(c))
    })
    return Array.from(currencySet).sort()
})

/* ------------------- TREE BUILDING ------------------- */
const buildTree = (items) => {
    if (!items || items.length === 0) return []
    const itemsById = {}
    items.forEach((item) => {
        itemsById[item.id] = { ...item, children: [] }
    })
    const rootItems = []
    items.forEach((item) => {
        if (item.ParentId === 0 || !itemsById[item.ParentId]) {
            rootItems.push(itemsById[item.id])
        } else {
            itemsById[item.ParentId].children.push(itemsById[item.id])
        }
    })
    const buildNode = (item) => ({
        key: String(item.id),
        data: { name: item.name, description: item.description, values: item.values },
        children: item.children?.map(buildNode)
    })
    return rootItems.map(buildNode)
}

const incomeNodes = computed(() =>
    reportData.value?.income ? buildTree(reportData.value.income) : []
)
const expenseNodes = computed(() =>
    reportData.value?.expenses ? buildTree(reportData.value.expenses) : []
)

const toggleIncomeVisibility = () => {
    if (incomeVisible.value) {
        // Hide income table
        incomeTableData.value = []
        incomeVisible.value = false
    } else {
        // Show income table
        incomeTableData.value = incomeNodes.value
        incomeVisible.value = true
    }
}

const toggleExpenseVisibility = () => {
    if (expenseVisible.value) {
        // Hide expense table
        expenseTableData.value = []
        expenseVisible.value = false
    } else {
        // Show expense table
        expenseTableData.value = expenseNodes.value
        expenseVisible.value = true
    }
}

const refetch = () => fetchReportData()

watch([startDate, endDate], () => {
    if (startDate.value && endDate.value) {
        fetchReportData()
    }
})

watch(incomeNodes, (newNodes) => {
    if (incomeVisible.value) {
        incomeTableData.value = newNodes
    }
})

watch(expenseNodes, (newNodes) => {
    if (expenseVisible.value) {
        expenseTableData.value = newNodes
    }
})

onMounted(() => fetchReportData())
</script>

<template>
    <div class="main-app-content">
        <SidebarContent :leftSidebarCollapsed="leftSidebarCollapsed" :rightSidebarCollapsed="true">
            <template #default>
                <div class="reports">
                    <div class="sidebar-controls">
                        <div class="date-filters">
                            <div>
                                <label>From:</label>
                                <DatePicker
                                    v-model="startDate"
                                    :showIcon="true"
                                    :showButtonBar="true"
                                    dateFormat="dd/mm/y"
                                    placeholder="Start date"
                                    @date-select="refetch"
                                />
                            </div>
                            <div>
                                <label>To:</label>
                                <DatePicker
                                    v-model="endDate"
                                    :showIcon="true"
                                    :showButtonBar="true"
                                    dateFormat="dd/mm/y"
                                    placeholder="End date"
                                    @date-select="refetch"
                                />
                            </div>
                        </div>
                    </div>

                    <!-- Income Tree Table -->
                    <div class="report-section">
                        <div class="section-header">
                            <h2 class="report-title">Income</h2>
                            <Button
                                :icon="incomeVisible ? 'pi pi-minus' : 'pi pi-plus'"
                                :label="incomeVisible ? 'Collapse All' : 'Expand All'"
                                class="p-button-sm p-button-text"
                                @click="toggleIncomeVisibility"
                            />
                        </div>

                        <TreeTable :value="incomeTableData" class="report-table">
                            <Column field="name" header="Name" :expander="true" />
                            <Column field="description" header="Description" />
                            <Column
                                v-if="incomeVisible"
                                v-for="currency in currencies"
                                :key="currency"
                                :header="currency"
                                class="amount-column"
                            >
                                <template #body="slotProps">
                                    <div
                                        v-if="
                                            slotProps.node.data.values &&
                                            slotProps.node.data.values[currency]
                                        "
                                    >
                                        <div>
                                            {{
                                                formatAmount(
                                                    slotProps.node.data.values[currency].amount
                                                )
                                            }}
                                        </div>
                                    </div>
                                    <span v-else class="empty-value">-</span>
                                </template>
                            </Column>
                        </TreeTable>
                    </div>

                    <!-- Expense Tree Table -->
                    <div class="report-section">
                        <div class="section-header">
                            <h2 class="report-title">Expenses</h2>
                            <Button
                                :icon="expenseVisible ? 'pi pi-minus' : 'pi pi-plus'"
                                :label="expenseVisible ? 'Collapse All' : 'Expand All'"
                                class="p-button-sm p-button-text"
                                @click="toggleExpenseVisibility"
                            />
                        </div>

                        <TreeTable :value="expenseTableData" class="report-table">
                            <Column field="name" header="Name" :expander="true" />
                            <Column field="description" header="Description" />
                            <Column
                                v-if="expenseVisible"
                                v-for="currency in currencies"
                                :key="currency"
                                :header="currency"
                                class="amount-column"
                            >
                                <template #body="slotProps">
                                    <div
                                        v-if="
                                            slotProps.node.data.values &&
                                            slotProps.node.data.values[currency]
                                        "
                                    >
                                        <div>
                                            {{
                                                formatAmount(
                                                    slotProps.node.data.values[currency].amount
                                                )
                                            }}
                                        </div>
                                    </div>
                                    <span v-else class="empty-value">-</span>
                                </template>
                            </Column>
                        </TreeTable>
                    </div>
                </div>
            </template>
        </SidebarContent>
    </div>
</template>

<style scoped>
.reports {
    padding: 0 1rem;
}
.date-filters {
    display: flex;
    gap: 1rem;
    align-items: center;
    justify-content: center;
    margin-top: 30px;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 2rem 0 1rem 0;
}

.report-title {
    color: var(--c-surface-700);
    margin: 0;
}
</style>
