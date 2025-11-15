<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import { fetchIncomeExpense } from '@/composables/useIncomeExpense.js'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import { Button, Card, Column, TreeTable } from 'primevue'

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

const calculateTotals = (nodes) => {
    const totals = {}
    // Only sum top-level items, don't traverse children
    nodes.forEach((node) => {
        if (node.data.values) {
            Object.entries(node.data.values).forEach(([currency, data]) => {
                if (!totals[currency]) totals[currency] = 0
                totals[currency] += data.amount || 0
            })
        }
    })
    return totals
}

const incomeTotals = computed(() => calculateTotals(incomeTableData.value))
const expenseTotals = computed(() => calculateTotals(expenseTableData.value))

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
                <DateRangePicker
                    v-model:startDate="startDate"
                    v-model:endDate="endDate"
                />
            </div>

            <Card>
                <template #content>
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

                        <div class="totals-row">
                            <div class="totals-cell name-cell">
                                <strong>Total</strong>
                            </div>
                            <div class="totals-cell description-cell"></div>
                            <div
                                v-for="currency in currencies"
                                :key="currency"
                                class="totals-cell amount-cell"
                            >
                                <strong>{{ formatAmount(incomeTotals[currency] || 0) }} {{ currency }}</strong>
                            </div>
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

                        <div class="totals-row">
                            <div class="totals-cell name-cell">
                                <strong>Total</strong>
                            </div>
                            <div class="totals-cell description-cell"></div>
                            <div
                                v-for="currency in currencies"
                                :key="currency"
                                class="totals-cell amount-cell"
                            >
                                <strong>{{ formatAmount(expenseTotals[currency] || 0) }} {{ currency }}</strong>
                            </div>
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
                </template>
            </Card>
        </div>
    </div>
</template>

<style scoped>
.reports {
    padding: 0 1rem;
}

.sidebar-controls {
    margin-top: 30px;
    margin-bottom: 1.5rem;
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

.totals-row {
    display: flex;
    align-items: center;
    padding: 0.75rem 1rem;
    background-color: var(--c-surface-50);
    margin-bottom: 0.5rem;
}

.totals-cell {
    padding: 0 0.5rem;
}

.name-cell {
    flex: 0 0 250px;
    min-width: 250px;
}

.description-cell {
    flex: 1;
    min-width: 200px;
}

.amount-cell {
    flex: 0 0 150px;
    min-width: 150px;
    text-align: right;
    color: var(--c-surface-700);
}

.empty-value {
    color: var(--c-text-color-secondary);
}
</style>
