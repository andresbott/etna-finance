<script setup>
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InstrumentDialog from './dialogs/InstrumentDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useInstruments } from '@/composables/useInstruments'
import { useSettingsStore } from '@/store/settingsStore'
import { useToast } from 'primevue/usetoast'
import { getApiErrorMessage } from '@/utils/apiError'

const settingsStore = useSettingsStore()
const toast = useToast()
const defaultCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
const {
    instruments: instrumentsData,
    isLoading,
    createInstrument,
    updateInstrument,
    deleteInstrument,
    isCreatingInstrument,
    isUpdatingInstrument
} = useInstruments()

const isSaving = computed(() => isCreatingInstrument.value || isUpdatingInstrument.value)

const instruments = computed(() => instrumentsData.value ?? [])

const selectedInstrument = ref(null)
const instrumentDialogVisible = ref(false)
const isEditInstrument = ref(false)
const deleteInstrumentDialogVisible = ref(false)
const instrumentToDelete = ref(null)

const openNewInstrumentDialog = () => {
    selectedInstrument.value = {
        symbol: '',
        name: '',
        currency: defaultCurrency.value
    }
    isEditInstrument.value = false
    instrumentDialogVisible.value = true
}

const editInstrument = (inst) => {
    selectedInstrument.value = {
        id: inst.id,
        symbol: inst.symbol,
        name: inst.name,
        currency: inst.currency
    }
    isEditInstrument.value = true
    instrumentDialogVisible.value = true
}

const showDeleteInstrumentDialog = (inst) => {
    instrumentToDelete.value = inst
    deleteInstrumentDialogVisible.value = true
}

const confirmDeleteInstrument = async () => {
    if (!instrumentToDelete.value) return
    try {
        await deleteInstrument(instrumentToDelete.value.id)
        deleteInstrumentDialogVisible.value = false
        instrumentToDelete.value = null
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: getApiErrorMessage(err), life: 5000 })
        console.error('Failed to delete instrument:', err)
    }
}

const saveInstrument = async (payload) => {
    try {
        if (payload.id) {
            await updateInstrument({
                id: payload.id,
                payload: {
                    symbol: payload.symbol,
                    name: payload.name,
                    currency: payload.currency
                }
            })
        } else {
            await createInstrument({
                symbol: payload.symbol,
                name: payload.name,
                currency: payload.currency
            })
        }
        instrumentDialogVisible.value = false
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: getApiErrorMessage(err), life: 5000 })
        console.error('Failed to save instrument:', err)
    }
}
</script>

<template>
    <div class="main-app-content">
        <div class="view-container w-full">
            <div class="flex justify-content-between align-items-center mb-2">
                <h1 class="m-0">Investment Products</h1>
                <Button
                    label="Add Instrument"
                    icon="pi pi-plus"
                    @click="openNewInstrumentDialog"
                />
            </div>
            <p class="text-color-secondary mt-0 mb-3">
                Manage your investment products such as stocks, ETFs, forex, and commodities.
            </p>

            <Card v-if="!instruments.length && !isLoading">
                <template #content>
                    <div class="info-message">
                        No instruments yet. Add one to get started.
                    </div>
                </template>
            </Card>

            <Card v-else>
                <template #content>
                    <DataTable
                        :value="instruments"
                        :loading="isLoading"
                        data-key="id"
                        class="p-datatable-sm"
                    >
                        <Column field="symbol" header="Symbol" />
                        <Column field="name" header="Name" />
                        <Column field="currency" header="Currency" />
                        <Column header="Actions" class="actions-column">
                            <template #body="{ data }">
                                <div class="flex gap-1 justify-content-end">
                                    <Button
                                        icon="pi pi-pencil"
                                        text
                                        rounded
                                        class="p-1"
                                        @click="editInstrument(data)"
                                    />
                                    <Button
                                        icon="pi pi-trash"
                                        text
                                        rounded
                                        severity="danger"
                                        class="p-1"
                                        @click="showDeleteInstrumentDialog(data)"
                                    />
                                </div>
                            </template>
                        </Column>
                    </DataTable>
                </template>
            </Card>
        </div>
    </div>

    <InstrumentDialog
        v-if="selectedInstrument"
        v-model:visible="instrumentDialogVisible"
        :is-edit="isEditInstrument"
        :instrument="selectedInstrument"
        :loading="isSaving"
        @save="saveInstrument"
    />

    <ConfirmDialog
        v-model:visible="deleteInstrumentDialogVisible"
        :name="instrumentToDelete?.name"
        title="Delete investment instrument"
        message="Are you sure you want to delete this investment instrument?"
        @confirm="confirmDeleteInstrument"
    />
</template>

<style scoped>
.info-message {
    padding: 1rem;
    text-align: center;
    color: var(--p-text-muted-color);
}
</style>
