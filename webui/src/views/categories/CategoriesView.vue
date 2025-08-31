<script setup lang="ts">
import { ref, computed } from 'vue'
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import TreeTable from 'primevue/treetable'
import ConfirmDialog from '@/components/common/confirmDialog.vue'
import { useCategories } from '@/composables/useCategories'
import { buildTreeForTable } from '@/utils/convertToTree'
import { UpdateIncomeCategoryDTO } from '@/types/category'

const {
    incomeCategories,
    createIncomeCategory,
    updateIncomeMutation,
    deleteIncomeMutation,
    expenseCategories,
    createExpenseMutation,
    updateExpenseMutation,
    deleteExpenseMutation,
} = useCategories()

// compute the tree data
const IncomeTreeData = computed(() => {
    if (!incomeCategories.data) return []
    return buildTreeForTable(incomeCategories.data.value)
})

const ExpenseTreeData = computed(() => {
    if (!expenseCategories.data) return []
    return buildTreeForTable(expenseCategories.data.value)
})

const categories = ref([])
const loading = ref(false)


/* --- Create and Edit Category--- */
const categoryDialogVisible = ref(false)
const categoryData = ref({ name: '', description: '', parentId: null })
const resetCategoryData = () => {
    categoryData.value = {  name: '', description: '', parentId: null };
};

// handle click Add/edit button click
const handleAddEditIncome = (item, action, type) => {
    if (action === 'add') {
        if (item != null) {
            categoryData.value.parentId = item.id
        }
        categoryData.value.action = 'add'
    } else if (action === 'edit') {
        categoryData.value.id = item.id
        categoryData.value.name = item.name
        categoryData.value.description = item.description
        categoryData.value.action = 'edit'
    }
    categoryData.value.type = type
    console.log('categoryData', categoryData.value)
    categoryDialogVisible.value = true
}

// submit handler
const saveCategory = () => {
    if (!categoryData.value.name) return

    console.log(categoryData.value)
    // CREATE
    if (categoryData.value.action === 'add') {
        const dto: CreateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined,
            parentId: categoryData.value.parentId || undefined
        }

        if (categoryData.value.type === 'income') {
            createIncomeCategory.mutate(dto, {
                onSuccess: () => {
                    categoryDialogVisible.value = false

                },
                onError: (err) => console.error('Failed to add income category', err),
                onSettled: () => {
                    resetCategoryData();
                },
            })
        }else if (categoryData.value.type === 'expense') {
            createExpenseMutation.mutate(dto, {
                onSuccess: () => {
                    categoryDialogVisible.value = false
                    categoryData.value = ref({ name: '', description: '', parentId: null })
                },
                onError: (err) => console.error('Failed to add income category', err),
                onSettled: () => {
                    resetCategoryData();
                },
            })
        }
    }

    // UPDATE
    if (categoryData.value.action === 'edit') {
        const dto: UpdateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined
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
                        resetCategoryData();
                    },
                }
            )
        }else if (categoryData.value.type === 'expense') {
            updateExpenseMutation.mutate(
                { id: categoryData.value.id, payload: dto },
                {
                    onSuccess: () => {
                        categoryDialogVisible.value = false
                    },
                    onError: (err) => console.error('Failed to add income category', err),
                    onSettled: () => {
                        resetCategoryData();
                    },
                }
            )
        }
    }
}

/* --- Delete Category --- */
const confirmDeleteDialog = ref(false)
const selectedItem = ref(null)
// handle click on delete icon
const confirmDelete = (item, type) => {
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
    }else if (selectedItem.value.type === 'expense') {
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
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <HorizontalLayout
                :fullHeight="true"
                :centerContent="true"
                :verticalCenterContent="false"
            >
                <template #default>
                    <Placeholder :width="'960px'" :height="'auto'">
                        <div class="categories-view">
                            <h1>Categories</h1>
                            <TabView>
                                <TabPanel header="Expense Categories">
                                    <TreeTable :value="ExpenseTreeData">
                                        <Column field="name" header="Name" expander></Column>
                                        <Column field="description" header="Description"></Column>
                                        <Column>
                                            <template #header>
                                                <div
                                                    style="
                                                        display: flex;
                                                        align-items: center;
                                                        justify-content: space-between;
                                                    "
                                                >
                                                    <Button
                                                        label="Add new parent category"
                                                        icon="pi pi-plus"
                                                        @click="handleAddEditIncome(null, 'add', 'expense')"
                                                    />
                                                </div>
                                            </template>
                                            <template #body="slotProps">
                                                <div class="actions">
                                                    <Button
                                                        icon="pi pi-plus"
                                                        text
                                                        rounded
                                                        @click="
                                                            handleAddEditIncome(
                                                                slotProps.node.data,
                                                                'add',
                                                                'expense'
                                                            )
                                                        "
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-pencil"
                                                        text
                                                        rounded
                                                        @click="
                                                            handleAddEditIncome(
                                                                slotProps.node.data,
                                                                'edit',
                                                                'expense'
                                                            )
                                                        "
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-trash"
                                                        text
                                                        rounded
                                                        severity="danger"
                                                        @click="confirmDelete(slotProps.node.data,'expense')"
                                                        class="action-button"
                                                    />
                                                </div>
                                            </template>
                                        </Column>
                                    </TreeTable>
                                </TabPanel>
                                <TabPanel header="Income Categories">
                                    <TreeTable :value="IncomeTreeData">
                                        <Column field="name" header="Name" expander></Column>
                                        <Column field="description" header="Description"></Column>
                                        <Column>
                                            <template #header>
                                                <div
                                                    style="
                                                        display: flex;
                                                        align-items: center;
                                                        justify-content: space-between;
                                                    "
                                                >
                                                    <Button
                                                        label="Add new parent category"
                                                        icon="pi pi-plus"
                                                        @click="handleAddEditIncome(null, 'add','income')"
                                                    />
                                                </div>
                                            </template>
                                            <template #body="slotProps">
                                                <div class="actions">
                                                    <Button
                                                        icon="pi pi-plus"
                                                        text
                                                        rounded
                                                        @click="
                                                            handleAddEditIncome(
                                                                slotProps.node.data,
                                                                'add',
                                                                'income'
                                                            )
                                                        "
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-pencil"
                                                        text
                                                        rounded
                                                        @click="
                                                            handleAddEditIncome(
                                                                slotProps.node.data,
                                                                'edit',
                                                                'income'
                                                            )
                                                        "
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-trash"
                                                        text
                                                        rounded
                                                        severity="danger"
                                                        @click="confirmDelete(slotProps.node.data,'income')"
                                                        class="action-button"
                                                    />
                                                </div>
                                            </template>
                                        </Column>
                                    </TreeTable>
                                </TabPanel>
                            </TabView>

                            <!-- Category Dialog -->
                            <Dialog
                                v-model:visible="categoryDialogVisible"
                                modal
                                :closable="true"
                                :draggable="false"
                                @hide="resetCategoryData"
                                header="Income Category"
                            >
                                <div class="p-fluid">
                                    <div class="field">
                                        <label for="income-name">Name</label>
                                        <InputText id="income-name" v-model="categoryData.name" />
                                    </div>
                                    <div class="field">
                                        <label for="income-description">Description</label>
                                        <InputText
                                            id="income-description"
                                            v-model="categoryData.description"
                                        />
                                    </div>
                                </div>
                                <template #footer>
                                    <Button
                                        label="Save"
                                        icon="pi pi-check"
                                        @click="saveCategory"
                                    />
                                    <Button
                                        label="Cancel"
                                        icon="pi pi-times"
                                        text
                                        @click="categoryDialogVisible = false"
                                    />
                                </template>
                            </Dialog>

                            <!-- Confirm Dialog -->
                            <ConfirmDialog
                                v-if="selectedItem"
                                v-model:visible="confirmDeleteDialog"
                                :name="selectedItem.name"
                                title="Delete Category"
                                message="Are you sure you want to delete this category?"
                                :onConfirm="deleteCategory"
                            />
                        </div>
                    </Placeholder>
                </template>
            </HorizontalLayout>
        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>
</template>

<style scoped>
.categories-view {
    padding: 2rem;
}

.tab-header {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 1rem;
}

.actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-start;
}

.action-button {
    padding: 0.25rem;
}

.field {
    margin-bottom: 1rem;
}

.field label {
    display: block;
    margin-bottom: 0.5rem;
}
</style>
