<script setup>
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import SecurityDialog from './dialogs/SecurityDialog.vue'
import ConfirmDialog from '@/components/common/confirmDialog.vue'
import { useSecurities } from '@/composables/useSecurities'

const {
    securities: securitiesData,
    isLoading,
    createSecurity,
    updateSecurity,
    deleteSecurity,
    isCreating,
    isUpdating
} = useSecurities()

const isSaving = computed(() => isCreating.value || isUpdating.value)

const securities = computed(() => securitiesData.value ?? [])

const dialogVisible = ref(false)
const isEdit = ref(false)
const selectedSecurity = ref(null)

const deleteDialogVisible = ref(false)
const securityToDelete = ref(null)

const openAdd = () => {
    selectedSecurity.value = null
    isEdit.value = false
    dialogVisible.value = true
}

const openEdit = (security) => {
    selectedSecurity.value = { ...security }
    isEdit.value = true
    dialogVisible.value = true
}

const saveSecurity = async (payload) => {
    try {
        if (payload.id) {
            await updateSecurity({
                id: payload.id,
                payload: {
                    symbol: payload.symbol,
                    name: payload.name,
                    currency: payload.currency
                }
            })
        } else {
            await createSecurity({
                symbol: payload.symbol,
                name: payload.name,
                currency: payload.currency
            })
        }
        dialogVisible.value = false
    } catch (err) {
        console.error('Failed to save security:', err)
    }
}

const showDeleteDialog = (security) => {
    securityToDelete.value = security
    deleteDialogVisible.value = true
}

const confirmDelete = async () => {
    if (!securityToDelete.value) return
    try {
        await deleteSecurity(securityToDelete.value.id)
        deleteDialogVisible.value = false
        securityToDelete.value = null
    } catch (err) {
        console.error('Failed to delete security:', err)
    }
}
</script>

<template>
    <div class="main-app-content">
        <div class="p-3 securities-view">
                <div class="header">
                    <h1>Security Setup</h1>
                    <Button
                        label="Add Security"
                        icon="pi pi-plus"
                        @click="openAdd"
                    />
                </div>
                <p class="text-color-secondary mt-0 mb-3">
                    Configure securities such as stocks, ETFs, and other tradable instruments.
                </p>

                <Card>
                    <template #content>
                        <DataTable
                            :value="securities"
                            :loading="isLoading"
                            stripedRows
                            class="p-datatable-sm"
                            responsiveLayout="scroll"
                            style="width: 100%"
                        >
                            <Column field="symbol" header="Symbol" />
                            <Column field="name" header="Name" />
                            <Column field="currency" header="Currency" />
                            <Column header="Actions" style="width: 150px">
                                <template #body="{ data }">
                                    <div class="actions">
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            rounded
                                            severity="secondary"
                                            size="small"
                                            @click="openEdit(data)"
                                            v-tooltip.top="'Edit'"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            text
                                            rounded
                                            severity="danger"
                                            size="small"
                                            @click="showDeleteDialog(data)"
                                            v-tooltip.top="'Delete'"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>
        </div>
    </div>

    <SecurityDialog
        v-model:visible="dialogVisible"
        :is-edit="isEdit"
        :security="selectedSecurity"
        :loading="isSaving"
        @save="saveSecurity"
    />

    <ConfirmDialog
        v-model:visible="deleteDialogVisible"
        :name="securityToDelete?.name"
        title="Delete Security"
        message="Are you sure you want to delete this security?"
        :on-confirm="confirmDelete"
    />
</template>

<style scoped>
.securities-view {
    width: 100%;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
}

.actions {
    display: flex;
    gap: 0.25rem;
}
</style>
