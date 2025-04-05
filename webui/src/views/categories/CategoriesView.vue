<script setup>
import { ref, onMounted } from 'vue'
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

const categories = ref([])
const loading = ref(false)
const expenseDialogVisible = ref(false)
const incomeDialogVisible = ref(false)
const newExpenseCategory = ref({ name: '', description: '' })
const newIncomeCategory = ref({ name: '', description: '' })

const fetchCategories = async () => {
    loading.value = true
    try {
        // TODO: Replace with actual API call
        categories.value = [
            { id: 1, name: 'Food', description: 'Food and dining expenses', type: 'expense' },
            { id: 2, name: 'Transportation', description: 'Transportation costs', type: 'expense' },
            { id: 3, name: 'Entertainment', description: 'Entertainment expenses', type: 'expense' }
        ]
    } catch (error) {
        console.error('Failed to fetch categories:', error)
    } finally {
        loading.value = false
    }
}

const saveExpenseCategory = async () => {
    try {
        // TODO: Replace with actual API call
        const newId = Math.max(...categories.value.map(c => c.id)) + 1
        categories.value.push({
            id: newId,
            name: newExpenseCategory.value.name,
            description: newExpenseCategory.value.description,
            type: 'expense'
        })
        expenseDialogVisible.value = false
        newExpenseCategory.value = { name: '', description: '' }
    } catch (error) {
        console.error('Failed to add expense category:', error)
    }
}

const saveIncomeCategory = async () => {
    try {
        // TODO: Replace with actual API call
        const newId = Math.max(...categories.value.map(c => c.id)) + 1
        categories.value.push({
            id: newId,
            name: newIncomeCategory.value.name,
            description: newIncomeCategory.value.description,
            type: 'income'
        })
        incomeDialogVisible.value = false
        newIncomeCategory.value = { name: '', description: '' }
    } catch (error) {
        console.error('Failed to add income category:', error)
    }
}

onMounted(() => {
    fetchCategories()
})
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
                                    <div class="tab-header">
                                        <Button 
                                            label="Add Expense Category" 
                                            icon="pi pi-plus" 
                                            @click="expenseDialogVisible = true" 
                                        />
                                    </div>
                                    <DataTable
                                        :value="categories.filter(c => c.type === 'expense')"
                                        :loading="loading"
                                        stripedRows
                                        class="p-datatable-sm"
                                    >
                                        <Column field="name" header="Name" sortable />
                                        <Column field="description" header="Description" sortable />
                                        <Column header="Actions" style="width: 100px">
                                            <template #body="{ data }">
                                                <div class="actions">
                                                    <Button
                                                        icon="pi pi-pencil"
                                                        text
                                                        rounded
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-trash"
                                                        text
                                                        rounded
                                                        severity="danger"
                                                        class="action-button"
                                                    />
                                                </div>
                                            </template>
                                        </Column>
                                    </DataTable>
                                </TabPanel>
                                <TabPanel header="Income Categories">
                                    <div class="tab-header">
                                        <Button 
                                            label="Add Income Category" 
                                            icon="pi pi-plus" 
                                            @click="incomeDialogVisible = true" 
                                        />
                                    </div>
                                    <DataTable
                                        :value="categories.filter(c => c.type === 'income')"
                                        :loading="loading"
                                        stripedRows
                                        class="p-datatable-sm"
                                    >
                                        <Column field="name" header="Name" sortable />
                                        <Column field="description" header="Description" sortable />
                                        <Column header="Actions" style="width: 100px">
                                            <template #body="{ data }">
                                                <div class="actions">
                                                    <Button
                                                        icon="pi pi-pencil"
                                                        text
                                                        rounded
                                                        class="action-button"
                                                    />
                                                    <Button
                                                        icon="pi pi-trash"
                                                        text
                                                        rounded
                                                        severity="danger"
                                                        class="action-button"
                                                    />
                                                </div>
                                            </template>
                                        </Column>
                                    </DataTable>
                                </TabPanel>
                            </TabView>

                            <!-- Expense Category Dialog -->
                            <Dialog
                                v-model:visible="expenseDialogVisible"
                                modal
                                header="Add New Expense Category"
                                :style="{ width: '450px' }"
                            >
                                <div class="p-fluid">
                                    <div class="field">
                                        <label for="expense-name">Name</label>
                                        <InputText id="expense-name" v-model="newExpenseCategory.name" />
                                    </div>
                                    <div class="field">
                                        <label for="expense-description">Description</label>
                                        <InputText id="expense-description" v-model="newExpenseCategory.description" />
                                    </div>
                                </div>
                                <template #footer>
                                    <Button label="Cancel" icon="pi pi-times" text @click="expenseDialogVisible = false" />
                                    <Button label="Save" icon="pi pi-check" @click="saveExpenseCategory" />
                                </template>
                            </Dialog>

                            <!-- Income Category Dialog -->
                            <Dialog
                                v-model:visible="incomeDialogVisible"
                                modal
                                header="Add New Income Category"
                                :style="{ width: '450px' }"
                            >
                                <div class="p-fluid">
                                    <div class="field">
                                        <label for="income-name">Name</label>
                                        <InputText id="income-name" v-model="newIncomeCategory.name" />
                                    </div>
                                    <div class="field">
                                        <label for="income-description">Description</label>
                                        <InputText id="income-description" v-model="newIncomeCategory.description" />
                                    </div>
                                </div>
                                <template #footer>
                                    <Button label="Cancel" icon="pi pi-times" text @click="incomeDialogVisible = false" />
                                    <Button label="Save" icon="pi pi-check" @click="saveIncomeCategory" />
                                </template>
                            </Dialog>
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