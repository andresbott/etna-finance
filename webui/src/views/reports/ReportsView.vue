<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import { fetchIncomeExpense } from '@/composables/useIncomeExpense.js'
import DatePicker from 'primevue/datepicker'
import { Button, Column, TreeTable } from 'primevue'

const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const incomeTableData = ref([])
const expenseTableData = ref([])
const reportData = ref(null)

const expandedIncomeKeys = ref({})
const expandedExpenseKeys = ref({})

const fetchReportData = async () => {
    try {
        const response = await fetchIncomeExpense(
            formatDateForAPI(startDate.value),
            formatDateForAPI(endDate.value)
        )

        reportData.value = await response

        // Apply zero filtering before assigning
        incomeTableData.value = filterZeroNodes(incomeNodes.value)
        expenseTableData.value = filterZeroNodes(expenseNodes.value)
    } catch (error) {
        console.error('Error fetching report data:', error)
    }
}

const formatAmount = (amount) =>
    amount.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

const formatDateForAPI = (date) => {
    if (!date) return ''
    const d = new Date(date)
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
}

const currencies = computed(() => {
    if (!reportData.value) return []
    const set = new Set()
    const income = reportData.value.income || []
    const expenses = reportData.value.expenses || []
    ;[...income, ...expenses].forEach((item) => {
        Object.keys(item.values || {}).forEach((c) => set.add(c))
    })
    return Array.from(set).sort()
})

const buildTree = (items) => {
    if (!items || items.length === 0) return []
    const map = {}
    items.forEach((item) => (map[item.id] = { ...item, children: [] }))
    const roots = []
    items.forEach((item) => {
        if (item.ParentId === 0 || !map[item.ParentId]) {
            roots.push(map[item.id])
        } else {
            map[item.ParentId].children.push(map[item.id])
        }
    })
    const toNode = (item) => ({
        key: String(item.id),
        data: { name: item.name, description: item.description, values: item.values },
        children: item.children?.map(toNode)
    })
    return roots.map(toNode)
}

const filterZeroNodes = (nodes) => {
    if (!nodes || nodes.length === 0) return []

    const hasNonZeroValues = (values) => {
        if (!values) return false
        return Object.values(values).some((v) => v.amount && v.amount !== 0)
    }

    const filterRecursive = (list) => {
        return list
            .map((node) => {
                const filteredChildren = filterRecursive(node.children || [])
                const selfHasValue = hasNonZeroValues(node.data.values)
                if (selfHasValue || filteredChildren.length > 0) {
                    return { ...node, children: filteredChildren }
                }
                return null
            })
            .filter((n) => n !== null)
    }

    return filterRecursive(nodes)
}

const incomeNodes = computed(() =>
    reportData.value?.income ? buildTree(reportData.value.income) : []
)
const expenseNodes = computed(() =>
    reportData.value?.expenses ? buildTree(reportData.value.expenses) : []
)

const expandAllNodes = (nodes) => {
    const expanded = {}
    const traverse = (list) => {
        list.forEach((n) => {
            expanded[n.key] = true
            if (n.children && n.children.length > 0) traverse(n.children)
        })
    }
    traverse(nodes)
    return expanded
}

const toggleIncomeVisibility = () => {
    if (Object.keys(expandedIncomeKeys.value).length > 0) {
        expandedIncomeKeys.value = {}
    } else {
        expandedIncomeKeys.value = expandAllNodes(incomeTableData.value)
    }
}

const toggleExpenseVisibility = () => {
    if (Object.keys(expandedExpenseKeys.value).length > 0) {
        expandedExpenseKeys.value = {}
    } else {
        expandedExpenseKeys.value = expandAllNodes(expenseTableData.value)
    }
}

watch([startDate, endDate], () => {
    if (startDate.value && endDate.value) fetchReportData()
})

watch(incomeNodes, (newNodes) => {
    incomeTableData.value = filterZeroNodes(newNodes)
})
watch(expenseNodes, (newNodes) => {
    expenseTableData.value = filterZeroNodes(newNodes)
})

onMounted(() => fetchReportData())
</script>

<template>
    <div class="main-app-content">
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
                        />
                    </div>
                </div>
            </div>

            <!-- Income Section -->
            <div class="report-section">
                <div class="section-header">
                    <h2 class="report-title">Income</h2>
                    <Button
                        :icon="
                            Object.keys(expandedIncomeKeys).length > 0
                                ? 'pi pi-minus'
                                : 'pi pi-plus'
                        "
                        :label="
                            Object.keys(expandedIncomeKeys).length > 0
                                ? 'Collapse All'
                                : 'Expand All'
                        "
                        class="p-button-sm p-button-text"
                        @click="toggleIncomeVisibility"
                    />
                </div>

                <TreeTable
                    :value="incomeTableData"
                    v-model:expandedKeys="expandedIncomeKeys"
                    class="report-table"
                >
                    <Column field="name" header="Name" :expander="true" />
                    <Column field="description" header="Description" />
                    <Column
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
                                {{ formatAmount(slotProps.node.data.values[currency].amount) }}
                            </div>
                            <span v-else class="empty-value">-</span>
                        </template>
                    </Column>
                </TreeTable>
            </div>

            <!-- Expense Section -->
            <div class="report-section">
                <div class="section-header">
                    <h2 class="report-title">Expenses</h2>
                    <Button
                        :icon="
                            Object.keys(expandedExpenseKeys).length > 0
                                ? 'pi pi-minus'
                                : 'pi pi-plus'
                        "
                        :label="
                            Object.keys(expandedExpenseKeys).length > 0
                                ? 'Collapse All'
                                : 'Expand All'
                        "
                        class="p-button-sm p-button-text"
                        @click="toggleExpenseVisibility"
                    />
                </div>

                <TreeTable
                    :value="expenseTableData"
                    v-model:expandedKeys="expandedExpenseKeys"
                    class="report-table"
                >
                    <Column field="name" header="Name" :expander="true" />
                    <Column field="description" header="Description" />
                    <Column
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
                                {{ formatAmount(slotProps.node.data.values[currency].amount) }}
                            </div>
                            <span v-else class="empty-value">-</span>
                        </template>
                    </Column>
                </TreeTable>
            </div>
        </div>
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
.empty-value {
    color: var(--c-text-color-secondary);
}
</style>
