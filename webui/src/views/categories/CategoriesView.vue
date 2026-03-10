<script setup lang="ts">
import { ref, watch } from 'vue'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Column from 'primevue/column'
import Button from 'primevue/button'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import TreeTable from 'primevue/treetable'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useCategories } from '@/composables/useCategories'
import { CreateIncomeCategoryDTO, UpdateIncomeCategoryDTO } from '@/types/category'
import CategoryDialog from './dialogs/CategoryDialog.vue'
import { useCategoryTree } from '@/composables/useCategoryTree'
import type { TreeTableExpandedKeys } from 'primevue/treetable'
import type { TreeNode } from 'primevue/treenode'

const {
    createIncomeCategory,
    updateIncomeMutation,
    deleteIncomeMutation,
    createExpenseMutation,
    updateExpenseMutation,
    deleteExpenseMutation
} = useCategories()

// compute the tree data
const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()

const expandedIncomeKeys = ref<TreeTableExpandedKeys>({})

watch(
    IncomeTreeData,
    (newNodes) => {
        if (newNodes && newNodes.length > 0) {
            expandedIncomeKeys.value = expandAll(newNodes)
        }
    },
    { immediate: true }
)

const expandedExpenseKeys = ref<TreeTableExpandedKeys>({})

watch(
    ExpenseTreeData,
    (newNodes) => {
        if (newNodes && newNodes.length > 0) {
            expandedExpenseKeys.value = expandAll(newNodes)
        }
    },
    { immediate: true }
)

function expandAll(nodes: TreeNode[]): TreeTableExpandedKeys {
    const expanded: TreeTableExpandedKeys = {}
    for (const node of nodes) {
        if (node.children && node.children.length) {
            expanded[node.key as string] = true
            Object.assign(expanded, expandAll(node.children))
        }
    }
    return expanded
}

interface Category {
    id: number | null
    parentId?: number | null
    name: string
    description: string
    type?: string
    action?: string
    icon?: string
}

/* --- Create and Edit Category--- */
const categoryDialogVisible = ref(false)
const categoryData = ref<Category>({ id: null, name: '', description: '', parentId: 0, icon: 'pi-tag' })
const resetCategoryData = () => {
    categoryData.value = { id: null, name: '', description: '', parentId: 0, icon: 'pi-tag' }
}

// handle click Add/edit button click
const handleAddEditIncome = (item: Category | null, action: string, type: string) => {
    if (action === 'add') {
        if (item != null) {
            categoryData.value.parentId = item.id
        }
        categoryData.value.action = 'add'
    } else if (action === 'edit') {
        if (item == null) {
            console.error('Something went wrong, item cannot be null for edit actions')
            return
        }
        categoryData.value.parentId = item.parentId ? item.parentId : 0
        categoryData.value.id = item.id
        categoryData.value.name = item.name
        categoryData.value.description = item.description
        categoryData.value.icon = item.icon || 'pi-tag'
        categoryData.value.action = 'edit'
    }
    categoryData.value.type = type
    categoryDialogVisible.value = true
}

// submit handler
const saveCategory = () => {
    if (!categoryData.value.name) return

    // CREATE
    if (categoryData.value.action === 'add') {
        const dto: CreateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined,
            parentId: categoryData.value.parentId || undefined,
            icon: categoryData.value.icon || 'pi-tag'
        }

        if (categoryData.value.type === 'income') {
            createIncomeCategory.mutate(dto, {
                onSuccess: () => {
                    categoryDialogVisible.value = false
                },
                onError: (err) => console.error('Failed to add income category', err),
                onSettled: () => {
                    resetCategoryData()
                }
            })
        } else if (categoryData.value.type === 'expense') {
            createExpenseMutation.mutate(dto, {
                onSuccess: () => {
                    categoryDialogVisible.value = false
                },
                onError: (err) => console.error('Failed to add income category', err),
                onSettled: () => {
                    resetCategoryData()
                }
            })
        }
    }

    // UPDATE
    if (categoryData.value.action === 'edit') {
        if (!categoryData.value.id) {
            console.error('Something went wrong and id is null')
            return
        }

        // TODO: Add updated categoryData->parentId to payload request
        const dto: UpdateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined,
            parentId: categoryData.value.parentId,
            icon: categoryData.value.icon || 'pi-tag'
        }

        if (categoryData.value.type === 'income') {
            updateIncomeMutation.mutate(
                { id: categoryData.value.id, payload: dto },
                {
                    onSuccess: () => {
                        categoryDialogVisible.value = false
                    },
                    onError: (err) => console.error('Failed to add income category', err),
                    onSettled: () => {
                        resetCategoryData()
                    }
                }
            )
        } else if (categoryData.value.type === 'expense') {
            updateExpenseMutation.mutate(
                { id: categoryData.value.id, payload: dto },
                {
                    onSuccess: () => {
                        categoryDialogVisible.value = false
                    },
                    onError: (err) => console.error('Failed to add income category', err),
                    onSettled: () => {
                        resetCategoryData()
                    }
                }
            )
        }
    }
}

/* --- Delete Category --- */

interface SelectDeleteCategory {
    id: number
    name: string
    type: string
}

const confirmDeleteDialog = ref(false)
const selectedItem = ref<SelectDeleteCategory | null>(null)
// handle click on delete icon
const confirmDelete = (item: SelectDeleteCategory, type: string) => {
    selectedItem.value = item
    selectedItem.value.type = type
    confirmDeleteDialog.value = true
}
// handle click ok on confirm delete
const deleteCategory = () => {
    if (selectedItem.value === null) return
    if (selectedItem.value.type === 'income') {
        deleteIncomeMutation.mutate(selectedItem.value.id, {
            onSuccess: () => {
                confirmDeleteDialog.value = false
                selectedItem.value = null
            },
            onError: (err) => {
                console.error('Failed to delete category:', err)
            }
        })
    } else if (selectedItem.value.type === 'expense') {
        deleteExpenseMutation.mutate(selectedItem.value.id, {
            onSuccess: () => {
                confirmDeleteDialog.value = false
                selectedItem.value = null
            },
            onError: (err) => {
                console.error('Failed to delete category:', err)
            }
        })
    }
}
</script>

<template>
    <div class="main-app-content">
        <h1>Categories</h1>
        <TabView>
            <TabPanel header="Expense Categories" :value="1">
                <TreeTable :value="ExpenseTreeData" :expandedKeys="expandedExpenseKeys">
                    <Column field="name" header="Name" expander>
                        <template #body="slotProps">
                            <span class="inline-flex align-items-center gap-2">
                                <i :class="['pi', slotProps.node.data.icon || 'pi-tag']"></i>
                                {{ slotProps.node.data.name }}
                            </span>
                        </template>
                    </Column>
                    <Column field="description" header="Description"></Column>
                    <Column>
                        <template #header>
                            <div class="flex gap-1 justify-content-end w-full">
                                <Button
                                    label="Add new parent category"
                                    icon="pi pi-plus"
                                    @click="handleAddEditIncome(null, 'add', 'expense')"
                                />
                            </div>
                        </template>
                        <template #body="slotProps">
                            <div class="flex gap-1 justify-content-end w-full">
                                <Button
                                    icon="pi pi-plus"
                                    text
                                    rounded
                                    class="p-1"
                                    @click="
                                        handleAddEditIncome(slotProps.node.data, 'add', 'expense')
                                    "
                                />
                                <Button
                                    icon="pi pi-pencil"
                                    text
                                    rounded
                                    class="p-1"
                                    @click="
                                        handleAddEditIncome(slotProps.node.data, 'edit', 'expense')
                                    "
                                />
                                <Button
                                    icon="pi pi-trash"
                                    text
                                    rounded
                                    severity="danger"
                                    class="p-1"
                                    @click="confirmDelete(slotProps.node.data, 'expense')"
                                />
                            </div>
                        </template>
                    </Column>
                </TreeTable>
            </TabPanel>
            <TabPanel header="Income Categories" :value="2">
                <TreeTable :value="IncomeTreeData" :expandedKeys="expandedIncomeKeys">
                    <Column field="name" header="Name" expander>
                        <template #body="slotProps">
                            <span class="inline-flex align-items-center gap-2">
                                <i :class="['pi', slotProps.node.data.icon || 'pi-tag']"></i>
                                {{ slotProps.node.data.name }}
                            </span>
                        </template>
                    </Column>
                    <Column field="description" header="Description"></Column>
                    <Column>
                        <template #header>
                            <div class="flex gap-1 justify-content-end w-full">
                                <Button
                                    label="Add new parent category"
                                    icon="pi pi-plus"
                                    @click="handleAddEditIncome(null, 'add', 'income')"
                                />
                            </div>
                        </template>
                        <template #body="slotProps">
                            <div class="flex gap-1 justify-content-end w-full">
                                <Button
                                    icon="pi pi-plus"
                                    text
                                    rounded
                                    class="p-1"
                                    @click="
                                        handleAddEditIncome(slotProps.node.data, 'add', 'income')
                                    "
                                />
                                <Button
                                    icon="pi pi-pencil"
                                    text
                                    rounded
                                    class="p-1"
                                    @click="
                                        handleAddEditIncome(slotProps.node.data, 'edit', 'income')
                                    "
                                />
                                <Button
                                    icon="pi pi-trash"
                                    text
                                    rounded
                                    severity="danger"
                                    class="p-1"
                                    @click="confirmDelete(slotProps.node.data, 'income')"
                                />
                            </div>
                        </template>
                    </Column>
                </TreeTable>
            </TabPanel>
        </TabView>
    </div>

    <CategoryDialog
        v-model:visible="categoryDialogVisible"
        :categoryData="categoryData"
        @update:categoryData="categoryData = $event"
        @save="saveCategory"
        @reset="resetCategoryData"
    />

    <ConfirmDialog
        v-if="selectedItem"
        v-model:visible="confirmDeleteDialog"
        :name="selectedItem.name"
        title="Delete Category"
        message="Are you sure you want to delete this category?"
        @confirm="deleteCategory"
    />
</template>

