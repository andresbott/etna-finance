<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import { fetchIncomeExpense } from '@/composables/useIncomeExpense'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import { Button, Card, Column, TreeTable } from 'primevue'

const router = useRouter()

const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const incomeTableData = ref([])
const expenseTableData = ref([])
const reportData = ref(null)

const expandedIncomeKeys = ref({ total: true })
const expandedExpenseKeys = ref({ total: true })

const fetchReportData = async () => {
    try {
        const response = await fetchIncomeExpense(
            formatDateForAPI(startDate.value),
            formatDateForAPI(endDate.value)
        )

        reportData.value = await response

        // Apply zero filtering and sorting before assigning
        incomeTableData.value = sortNodesByValue(filterZeroNodes(incomeNodes.value))
        expenseTableData.value = sortNodesByValue(filterZeroNodes(expenseNodes.value))
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
        data: { name: item.name, description: item.description, icon: item.icon, values: item.values },
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

const getTotalValue = (node) => {
    if (!node.data.values) return 0
    return Object.values(node.data.values).reduce((sum, v) => sum + Math.abs(v.amount || 0), 0)
}

const sortNodesByValue = (nodes) => {
    if (!nodes || nodes.length === 0) return []

    return [...nodes]
        .map((node) => ({
            ...node,
            children: sortNodesByValue(node.children || [])
        }))
        .sort((a, b) => {
            if (a.key === '0') return 1
            if (b.key === '0') return -1
            return getTotalValue(b) - getTotalValue(a)
        })
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

const addTotalNode = (nodes) => {
    if (!nodes || nodes.length === 0) return []
    
    const totals = calculateTotals(nodes)
    const totalValues = {}
    
    Object.entries(totals).forEach(([currency, amount]) => {
        totalValues[currency] = { amount }
    })
    
    const totalNode = {
        key: 'total',
        data: {
            name: 'Total',
            description: '',
            values: totalValues
        },
        children: nodes
    }
    
    return [totalNode]
}

const incomeTableDataWithTotal = computed(() => addTotalNode(incomeTableData.value))
const expenseTableDataWithTotal = computed(() => addTotalNode(expenseTableData.value))

const incomeTotals = computed(() => calculateTotals(incomeTableData.value))
const expenseTotals = computed(() => calculateTotals(expenseTableData.value))

const toggleIncomeVisibility = () => {
    const allKeys = expandAllNodes(incomeTableDataWithTotal.value)
    const hasAllExpanded = Object.keys(expandedIncomeKeys.value).length === Object.keys(allKeys).length
    
    if (hasAllExpanded) {
        // Collapse to default view (only Total expanded)
        expandedIncomeKeys.value = { total: true }
    } else {
        // Expand all nodes
        expandedIncomeKeys.value = allKeys
    }
}

const toggleExpenseVisibility = () => {
    const allKeys = expandAllNodes(expenseTableDataWithTotal.value)
    const hasAllExpanded = Object.keys(expandedExpenseKeys.value).length === Object.keys(allKeys).length
    
    if (hasAllExpanded) {
        // Collapse to default view (only Total expanded)
        expandedExpenseKeys.value = { total: true }
    } else {
        // Expand all nodes
        expandedExpenseKeys.value = allKeys
    }
}

const isIncomeFullyExpanded = computed(() => {
    const allKeys = expandAllNodes(incomeTableDataWithTotal.value)
    return Object.keys(expandedIncomeKeys.value).length === Object.keys(allKeys).length
})

const isExpenseFullyExpanded = computed(() => {
    const allKeys = expandAllNodes(expenseTableDataWithTotal.value)
    return Object.keys(expandedExpenseKeys.value).length === Object.keys(allKeys).length
})

const collectNodeIds = (node) => {
    const ids = []
    if (node.key && node.key !== 'total') {
        ids.push(Number(node.key))
    }
    if (node.children) {
        for (const child of node.children) {
            ids.push(...collectNodeIds(child))
        }
    }
    return ids
}

const viewEntries = (node) => {
    const ids = collectNodeIds(node)
    if (ids.length === 0) return
    router.push({
        path: '/entries',
        query: {
            from: formatDateForAPI(startDate.value),
            to: formatDateForAPI(endDate.value),
            page: '1',
            limit: '25',
            categoryIds: ids.join(',')
        }
    })
}

watch([startDate, endDate], () => {
    if (startDate.value && endDate.value) fetchReportData()
})

watch(incomeNodes, (newNodes) => {
    incomeTableData.value = sortNodesByValue(filterZeroNodes(newNodes))
})
watch(expenseNodes, (newNodes) => {
    expenseTableData.value = sortNodesByValue(filterZeroNodes(newNodes))
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
                                :icon="isIncomeFullyExpanded ? 'pi pi-minus' : 'pi pi-plus'"
                                :label="isIncomeFullyExpanded ? 'Collapse' : 'Expand'"
                                class="p-button-sm p-button-text"
                                @click="toggleIncomeVisibility"
                            />
                        </div>

                        <TreeTable
                            :value="incomeTableDataWithTotal"
                            v-model:expandedKeys="expandedIncomeKeys"
                            class="report-table"
                        >
                            <Column field="name" header="Name" :expander="true">
                                <template #body="slotProps">
                                    <span class="category-name">
                                        <i :class="['pi', slotProps.node.data.icon || 'pi-tag']"></i>
                                        {{ slotProps.node.data.name }}
                                        <Button
                                            v-if="slotProps.node.key !== 'total'"
                                            icon="pi pi-list"
                                            class="view-entries-btn"
                                            text
                                            rounded
                                            size="small"
                                            @click.stop="viewEntries(slotProps.node)"
                                            v-tooltip.top="'View entries'"
                                        />
                                    </span>
                                </template>
                            </Column>
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
                                        :class="{ 'bold-total': slotProps.node.key === 'total' }"
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
                                :icon="isExpenseFullyExpanded ? 'pi pi-minus' : 'pi pi-plus'"
                                :label="isExpenseFullyExpanded ? 'Collapse' : 'Expand'"
                                class="p-button-sm p-button-text"
                                @click="toggleExpenseVisibility"
                            />
                        </div>

                        <TreeTable
                            :value="expenseTableDataWithTotal"
                            v-model:expandedKeys="expandedExpenseKeys"
                            class="report-table"
                        >
                            <Column field="name" header="Name" :expander="true">
                                <template #body="slotProps">
                                    <span class="category-name">
                                        <i :class="['pi', slotProps.node.data.icon || 'pi-tag']"></i>
                                        {{ slotProps.node.data.name }}
                                        <Button
                                            v-if="slotProps.node.key !== 'total'"
                                            icon="pi pi-list"
                                            class="view-entries-btn"
                                            text
                                            rounded
                                            size="small"
                                            @click.stop="viewEntries(slotProps.node)"
                                            v-tooltip.top="'View entries'"
                                        />
                                    </span>
                                </template>
                            </Column>
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
                                        :class="{ 'bold-total': slotProps.node.key === 'total' }"
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

.empty-value {
    color: var(--c-text-color-secondary);
}

.bold-total {
    font-weight: bold;
}

.category-name {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
}

.view-entries-btn {
    opacity: 0.4;
    transition: opacity 0.2s;
    width: 1.75rem !important;
    height: 1.75rem !important;
    padding: 0 !important;
}

.category-name:hover .view-entries-btn {
    opacity: 1;
}
</style>
