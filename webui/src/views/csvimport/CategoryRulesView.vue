<script setup>
import { ref, computed, onMounted } from 'vue'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Card from 'primevue/card'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import { useToast } from 'primevue/usetoast'
import { useRouter } from 'vue-router'
import { getCategoryRules, createCategoryRule, updateCategoryRule, deleteCategoryRule } from '@/lib/api/CsvImport'
import { useCategoryTree } from '@/composables/useCategoryTree'
import { useCategoryUtils } from '@/utils/categoryUtils'

const toast = useToast()
const router = useRouter()
const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()
const { getCategoryName } = useCategoryUtils()

// State
const rules = ref([])
const isLoading = ref(false)
const showRuleDialog = ref(false)
const editingRule = ref(null)
const isSaving = ref(false)

// Form state
const formPattern = ref('')
const formIsRegex = ref(false)
const formCategoryId = ref(null)
const formPosition = ref(0)

// Flatten tree nodes into a flat list for the dropdown
const flattenNodes = (nodes, prefix = '') => {
    const result = []
    if (!nodes || !Array.isArray(nodes)) return result
    for (const node of nodes) {
        const path = prefix ? `${prefix} > ${node.data?.name || node.label}` : (node.data?.name || node.label)
        const id = node.data?.id ?? parseInt(node.key, 10)
        if (id) {
            result.push({ label: path, value: id })
        }
        if (node.children?.length) {
            result.push(...flattenNodes(node.children, path))
        }
    }
    return result
}

const categoryOptions = computed(() => {
    const income = flattenNodes(IncomeTreeData.value).map(c => ({ ...c, label: `[Income] ${c.label}` }))
    const expense = flattenNodes(ExpenseTreeData.value).map(c => ({ ...c, label: `[Expense] ${c.label}` }))
    return [...expense, ...income]
})

// Resolve category name for display
const resolveCategoryName = (categoryId) => {
    // Try expense first, then income
    let name = getCategoryName(categoryId, 'expense')
    if (name === 'Unknown') {
        name = getCategoryName(categoryId, 'income')
    }
    return name
}

// Load rules
const loadRules = async () => {
    isLoading.value = true
    try {
        rules.value = await getCategoryRules()
        // Sort by position
        rules.value.sort((a, b) => a.position - b.position)
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load category rules: ' + error.message, life: 3000 })
    } finally {
        isLoading.value = false
    }
}

// Reset form
const resetForm = () => {
    formPattern.value = ''
    formIsRegex.value = false
    formCategoryId.value = null
    formPosition.value = rules.value.length
}

// Open create dialog
const openCreateDialog = () => {
    editingRule.value = null
    resetForm()
    showRuleDialog.value = true
}

// Open edit dialog
const openEditDialog = (rule) => {
    editingRule.value = rule
    formPattern.value = rule.pattern
    formIsRegex.value = rule.isRegex
    formCategoryId.value = rule.categoryId
    formPosition.value = rule.position
    showRuleDialog.value = true
}

// Save rule
const handleSaveRule = async () => {
    if (!formPattern.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Pattern is required', life: 3000 })
        return
    }
    if (!formCategoryId.value) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Category is required', life: 3000 })
        return
    }

    const payload = {
        pattern: formPattern.value.trim(),
        isRegex: formIsRegex.value,
        categoryId: formCategoryId.value,
        position: formPosition.value ?? 0
    }

    isSaving.value = true
    try {
        if (editingRule.value) {
            await updateCategoryRule(editingRule.value.id, payload)
            toast.add({ severity: 'success', summary: 'Success', detail: 'Rule updated successfully', life: 3000 })
        } else {
            await createCategoryRule(payload)
            toast.add({ severity: 'success', summary: 'Success', detail: 'Rule created successfully', life: 3000 })
        }
        showRuleDialog.value = false
        await loadRules()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save rule: ' + error.message, life: 3000 })
    } finally {
        isSaving.value = false
    }
}

// Delete rule
const handleDeleteRule = async (rule) => {
    if (!confirm(`Are you sure you want to delete the rule "${rule.pattern}"?`)) {
        return
    }

    try {
        await deleteCategoryRule(rule.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Rule deleted successfully', life: 3000 })
        await loadRules()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete rule: ' + error.message, life: 3000 })
    }
}

onMounted(() => {
    loadRules()
})
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>

        </template>
        <template #default>
            <div class="view-container">
                <div class="flex justify-content-between align-items-start mb-4 gap-3">
                    <div>
                        <h1 class="text-2xl font-bold mb-2 text-color">Category Matching Rules</h1>
                        <p class="text-color-secondary m-0 text-base">
                            Define rules to automatically assign categories to imported transactions based on description matching
                        </p>
                    </div>
                    <div class="flex gap-2">
                        <Button
                            label="Re-apply Rules"
                            icon="pi pi-sync"
                            severity="secondary"
                            @click="router.push('/setup/reapply-rules')"
                        />
                        <Button
                            label="New Rule"
                            icon="pi pi-plus"
                            @click="openCreateDialog"
                        />
                    </div>
                </div>

                <Card>
                    <template #content>
                        <DataTable
                            :value="rules"
                            :loading="isLoading"
                            stripedRows
                            :paginator="rules.length > 10"
                            :rows="10"
                            responsiveLayout="scroll"
                        >
                            <template #empty>
                                <div class="empty-state">
                                    <i class="pi pi-inbox"></i>
                                    <p>No category matching rules found</p>
                                    <Button
                                        label="Create Your First Rule"
                                        icon="pi pi-plus"
                                        @click="openCreateDialog"
                                        outlined
                                    />
                                </div>
                            </template>

                            <Column field="position" header="Position" :sortable="true" style="width: 100px">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.position }}</span>
                                </template>
                            </Column>

                            <Column field="pattern" header="Pattern">
                                <template #body="{ data }">
                                    <span class="pattern-text">{{ data.pattern }}</span>
                                </template>
                            </Column>

                            <Column field="isRegex" header="Type" style="width: 120px">
                                <template #body="{ data }">
                                    <Tag
                                        :value="data.isRegex ? 'Regex' : 'Substring'"
                                        :severity="data.isRegex ? 'warn' : 'info'"
                                    />
                                </template>
                            </Column>

                            <Column field="categoryId" header="Category">
                                <template #body="{ data }">
                                    {{ resolveCategoryName(data.categoryId) }}
                                </template>
                            </Column>

                            <Column header="Actions" :exportable="false" style="width: 100px">
                                <template #body="{ data }">
                                    <div class="flex gap-1 justify-content-center">
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            rounded
                                            class="p-1"
                                            @click="openEditDialog(data)"
                                            v-tooltip.top="'Edit rule'"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            severity="danger"
                                            text
                                            rounded
                                            class="p-1"
                                            @click="handleDeleteRule(data)"
                                            v-tooltip.top="'Delete rule'"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Rule Edit/Create Dialog -->
                <Dialog
                    v-model:visible="showRuleDialog"
                    :header="editingRule ? 'Edit Category Rule' : 'Create Category Rule'"
                    :modal="true"
                    :closable="true"
                    class="entry-dialog entry-dialog--wide"
                >
                    <div class="rule-dialog-content">
                        <div class="field">
                            <label for="rulePattern">Pattern *</label>
                            <InputText
                                id="rulePattern"
                                v-model="formPattern"
                                placeholder="e.g., GROCERY or .*grocery.*"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <div class="flex align-items-center gap-2">
                                <Checkbox
                                    id="ruleIsRegex"
                                    v-model="formIsRegex"
                                    :binary="true"
                                />
                                <label for="ruleIsRegex">Is Regex</label>
                            </div>
                        </div>

                        <div class="field">
                            <label for="ruleCategory">Category *</label>
                            <Select
                                id="ruleCategory"
                                v-model="formCategoryId"
                                :options="categoryOptions"
                                optionLabel="label"
                                optionValue="value"
                                placeholder="Select a category"
                                filter
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <label for="rulePosition">Position</label>
                            <InputNumber
                                id="rulePosition"
                                v-model="formPosition"
                                :min="0"
                                class="w-full"
                            />
                        </div>

                        <div class="flex justify-content-end gap-2 mt-3">
                            <Button
                                label="Cancel"
                                severity="secondary"
                                text
                                @click="showRuleDialog = false"
                            />
                            <Button
                                :label="editingRule ? 'Update' : 'Create'"
                                icon="pi pi-check"
                                :loading="isSaving"
                                @click="handleSaveRule"
                            />
                        </div>
                    </div>
                </Dialog>
            </div>
        </template>
    </VerticalLayout>
</template>

<style scoped lang="scss">
.empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-color-secondary);

    i {
        font-size: 3rem;
        margin-bottom: 1rem;
        opacity: 0.5;
    }

    p {
        margin-bottom: 1.5rem;
        font-size: 1.1rem;
    }
}

.pattern-text {
    font-family: monospace;
    background-color: var(--surface-100);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.9rem;
}

.rule-dialog-content {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;

    label {
        font-weight: 600;
        color: var(--text-color);
    }
}

:deep(.p-card-content) {
    padding: 1.5rem;
}
</style>
