<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'

import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Card from 'primevue/card'
import Checkbox from 'primevue/checkbox'
import ProgressSpinner from 'primevue/progressspinner'

import { reapplyPreview, reapplySubmit } from '@/lib/api/CsvImport'
import { useDateFormat } from '@/composables/useDateFormat'
import { getEntryTypeIcon } from '@/utils/entryDisplay'
import { getApiErrorMessage } from '@/utils/apiError'

const router = useRouter()
const toast = useToast()
const { formatDate } = useDateFormat()

const formatAmount = (n) =>
    n != null && !Number.isNaN(n)
        ? n.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
        : '0.00'

/* --- State --- */
const rows = ref(null)
const isLoading = ref(false)
const isSubmitting = ref(false)
const checkedRows = ref({})

/* --- Summary --- */
const totalCount = computed(() => {
    if (!rows.value) return 0
    return rows.value.length
})

/* --- Checked count --- */
const checkedCount = computed(() => {
    if (!rows.value) return 0
    return rows.value.filter((r) => checkedRows.value[r.transactionId]).length
})

/* --- Row class --- */
const getRowClass = (data) => ({
    'expense-row': data.transactionType === 'expense',
    'income-row': data.transactionType === 'income'
})

/* --- Load preview --- */
const loadPreview = async () => {
    isLoading.value = true
    try {
        const result = await reapplyPreview()
        rows.value = result
        const checked = {}
        for (const row of result) {
            checked[row.transactionId] = row.changed
        }
        checkedRows.value = checked
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load preview: ' + getApiErrorMessage(err), life: 5000 })
        rows.value = []
    } finally {
        isLoading.value = false
    }
}

/* --- Submit --- */
const handleSubmit = async () => {
    if (!rows.value) return
    const selected = rows.value
        .filter((r) => checkedRows.value[r.transactionId])
        .map((r) => ({
            transactionId: r.transactionId,
            transactionType: r.transactionType,
            newCategoryId: r.newCategoryId
        }))

    if (selected.length === 0) {
        toast.add({ severity: 'warn', summary: 'No rows selected', detail: 'Select at least one transaction to update.', life: 3000 })
        return
    }

    isSubmitting.value = true
    try {
        const result = await reapplySubmit(selected)
        toast.add({
            severity: 'success',
            summary: 'Categories updated',
            detail: `${result.updated} transactions updated successfully.`,
            life: 4000
        })
        router.push('/setup/csv-profiles')
    } catch (err) {
        toast.add({
            severity: 'error',
            summary: 'Update failed',
            detail: getApiErrorMessage(err),
            life: 5000
        })
    } finally {
        isSubmitting.value = false
    }
}

/* --- Navigation --- */
const handleBack = () => {
    router.push('/setup/csv-profiles')
}

onMounted(() => {
    loadPreview()
})
</script>

<template>
    <div class="main-app-content">
        <div class="reapply-content">
            <!-- Header -->
            <div class="toolbar">
                <div class="toolbar-left">
                    <Button icon="pi pi-arrow-left" text rounded @click="handleBack" v-tooltip.bottom="'Back to rules'" class="mr-2" />
                    <h2 class="page-title">Re-apply Category Rules</h2>
                </div>
            </div>

            <!-- Loading -->
            <div v-if="isLoading" class="loading-section">
                <ProgressSpinner />
                <p>Analyzing transactions...</p>
            </div>

            <!-- Results -->
            <div v-else-if="rows" class="preview-section">
                <!-- Summary Bar -->
                <div class="summary-bar">
                    <span class="summary-item summary-changed">
                        <i class="pi pi-sync"></i> {{ totalCount }} to update
                    </span>
                </div>

                <!-- Preview Table -->
                <Card>
                    <template #content>
                        <DataTable
                            class="datatable-compact"
                            :value="rows"
                            stripedRows
                            style="width: 100%"
                            :rowClass="getRowClass"
                            :paginator="rows.length > 25"
                            :rows="25"
                        >
                            <template #empty>
                                <div class="empty-state">
                                    <i class="pi pi-check-circle"></i>
                                    <p>No transactions match any category rules</p>
                                </div>
                            </template>

                            <!-- Checkbox column -->
                            <Column header="" style="width: 2.5rem">
                                <template #body="{ data }">
                                    <Checkbox
                                        v-model="checkedRows[data.transactionId]"
                                        :binary="true"
                                    />
                                </template>
                            </Column>

                            <!-- Type icon -->
                            <Column header="" style="width: 2rem">
                                <template #body="{ data }">
                                    <i :class="getEntryTypeIcon(data.transactionType)" style="font-size: 0.8rem" />
                                </template>
                            </Column>

                            <!-- Description -->
                            <Column field="description" header="Description" bodyClass="description-cell" />

                            <!-- Date -->
                            <Column field="date" header="Date" style="width: 7rem">
                                <template #body="{ data }">
                                    {{ formatDate(data.date) }}
                                </template>
                            </Column>

                            <!-- Amount -->
                            <Column field="amount" header="Amount" bodyStyle="text-align: right" style="width: 6rem">
                                <template #body="{ data }">
                                    <div class="amount" :class="data.transactionType === 'expense' ? 'expense' : 'income'">
                                        <template v-if="data.transactionType === 'expense'">-</template>
                                        <template v-else>+</template>
                                        {{ formatAmount(data.amount) }}
                                    </div>
                                </template>
                            </Column>

                            <!-- Account -->
                            <Column field="accountName" header="Account" style="width: 8rem" bodyStyle="white-space: nowrap" />

                            <!-- Current Category -->
                            <Column header="Current" style="width: 8rem">
                                <template #body="{ data }">
                                    {{ data.currentCategoryName || '—' }}
                                </template>
                            </Column>

                            <!-- New Category -->
                            <Column header="New" style="width: 8rem">
                                <template #body="{ data }">
                                    <span :class="{ 'category-changed': data.changed }">
                                        {{ data.newCategoryName || '—' }}
                                    </span>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Action buttons -->
                <div class="preview-actions">
                    <Button
                        :label="`Apply Selected (${checkedCount})`"
                        icon="pi pi-check"
                        :loading="isSubmitting"
                        :disabled="checkedCount === 0"
                        @click="handleSubmit"
                    />
                    <Button
                        label="Cancel"
                        severity="secondary"
                        icon="pi pi-times"
                        @click="handleBack"
                    />
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.main-app-content {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.reapply-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: auto;
}

.toolbar {
    display: flex;
    align-items: center;
    padding: 1rem;
    background-color: var(--surface-ground);
    border-bottom: 1px solid var(--surface-border);
}

.toolbar-left {
    display: flex;
    align-items: center;
}

.page-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-700);
}

.loading-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    color: var(--text-color-secondary);
}

.preview-section {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.summary-bar {
    display: flex;
    gap: 1.5rem;
    padding: 0.75rem 1rem;
    background-color: var(--surface-card);
    border: 1px solid var(--surface-border);
    border-radius: var(--border-radius);
    font-weight: 600;
}

.summary-item {
    display: flex;
    align-items: center;
    gap: 0.4rem;
}

.summary-changed {
    color: var(--blue-600);
}

.preview-actions {
    display: flex;
    gap: 0.75rem;
    padding-top: 0.5rem;
}

.amount.expense {
    color: var(--red-500);
}

.amount.income {
    color: var(--green-500);
}

.category-changed {
    font-weight: 600;
    color: var(--blue-600);
}

.empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-color-secondary);
}

.empty-state i {
    font-size: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
}

:deep(.datatable-compact .p-datatable-tbody > tr > td) {
    padding-top: 0.75rem;
    padding-bottom: 0.75rem;
}

:deep(.description-cell) {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 1px;
}

</style>
