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

const entries = ref([])
const loading = ref(false)
const visible = ref(false)
const newEntry = ref({ name: '', description: '' })

const fetchEntries = async () => {
    loading.value = true
    try {
        // TODO: Replace with actual API call
        entries.value = [
            { id: 1, name: 'Salary', description: 'Monthly salary income' },
            { id: 2, name: 'Rent', description: 'Monthly rent payment' },
            { id: 3, name: 'Groceries', description: 'Weekly grocery shopping' }
        ]
    } catch (error) {
        console.error('Failed to fetch entries:', error)
    } finally {
        loading.value = false
    }
}

const saveEntry = async () => {
    try {
        // TODO: Replace with actual API call
        const newId = Math.max(...entries.value.map(e => e.id)) + 1
        entries.value.push({
            id: newId,
            name: newEntry.value.name,
            description: newEntry.value.description
        })
        visible.value = false
        newEntry.value = { name: '', description: '' }
    } catch (error) {
        console.error('Failed to add entry:', error)
    }
}

onMounted(() => {
    fetchEntries()
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
                        <div class="entries-view">
                            <div class="header">
                                <h1>Entries</h1>
                                <Button label="Add Entry" icon="pi pi-plus" @click="visible = true" />
                            </div>

                            <DataTable
                                :value="entries"
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

                            <Dialog
                                v-model:visible="visible"
                                modal
                                header="Add New Entry"
                                :style="{ width: '450px' }"
                            >
                                <div class="p-fluid">
                                    <div class="field">
                                        <label for="name">Name</label>
                                        <InputText id="name" v-model="newEntry.name" />
                                    </div>
                                    <div class="field">
                                        <label for="description">Description</label>
                                        <InputText id="description" v-model="newEntry.description" />
                                    </div>
                                </div>
                                <template #footer>
                                    <Button label="Cancel" icon="pi pi-times" text @click="visible = false" />
                                    <Button label="Save" icon="pi pi-check" @click="saveEntry" />
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
.entries-view {
    padding: 2rem;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
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