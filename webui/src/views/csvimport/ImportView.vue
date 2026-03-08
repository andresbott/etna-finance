<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'

import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Card from 'primevue/card'
import Message from 'primevue/message'
import Checkbox from 'primevue/checkbox'

import { parseCSV, submitImport } from '@/lib/api/CsvImport'
import { useAccounts } from '@/composables/useAccounts'
import { useCategoryUtils } from '@/utils/categoryUtils'
import { useDateFormat } from '@/composables/useDateFormat'
import { getEntryTypeIcon } from '@/utils/entryDisplay'

/* --- Route & Navigation --- */
const route = useRoute()
const router = useRouter()
const toast = useToast()
const accountId = computed(() => route.params.accountId)

/* --- Accounts --- */
const { accounts } = useAccounts()

const account = computed(() => {
    if (!accountId.value || !accounts?.value) return null
    for (const provider of accounts.value) {
        if (provider.accounts) {
            for (const acct of provider.accounts) {
                if (String(acct.id) === String(accountId.value)) {
                    return acct
                }
            }
        }
    }
    return null
})

const accountName = computed(() => account.value?.name ?? 'Loading...')
const accountCurrency = computed(() => account.value?.currency ?? '')
const accountTitle = computed(() => {
    if (accountCurrency.value) {
        return `${accountName.value} (${accountCurrency.value})`
    }
    return accountName.value
})

/* --- Utils --- */
const { getCategoryPath } = useCategoryUtils()
const { formatDate } = useDateFormat()
const formatAmount = (n) =>
    n != null && !Number.isNaN(n)
        ? n.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
        : '0.00'

/* --- Upload State --- */
const selectedFile = ref(null)
const isParsing = ref(false)
const parseError = ref('')

/* --- Preview State --- */
const parsedRows = ref(null) // null = upload state, array = preview state
const checkedRows = ref({})  // rowNumber -> boolean

const isPreview = computed(() => parsedRows.value !== null)

/* --- Summary --- */
const summary = computed(() => {
    if (!parsedRows.value) return { newCount: 0, duplicateCount: 0, errorCount: 0 }
    let newCount = 0
    let duplicateCount = 0
    let errorCount = 0
    for (const row of parsedRows.value) {
        if (row.error) errorCount++
        else if (row.isDuplicate) duplicateCount++
        else newCount++
    }
    return { newCount, duplicateCount, errorCount }
})

/* --- Transformed entries for the table --- */
const tableEntries = computed(() => {
    if (!parsedRows.value) return []
    return parsedRows.value.map((row) => ({
        id: `import-${row.rowNumber}`,
        rowNumber: row.rowNumber,
        type: row.type,
        description: row.description,
        date: row.date,
        Amount: Math.abs(row.amount),
        accountId: accountId.value,
        categoryId: row.categoryId || null,
        isImportRow: true,
        isDuplicate: row.isDuplicate,
        importError: row.error
    }))
})

/* --- Row class helper --- */
const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'duplicate-row': data.isDuplicate,
    'error-row': !!data.importError
})

/* --- File handling --- */
const onFileChange = (event) => {
    const files = event.target.files
    if (files && files.length > 0) {
        selectedFile.value = files[0]
    }
}

/* --- Parse --- */
const handleParse = async () => {
    if (!selectedFile.value) return
    isParsing.value = true
    parseError.value = ''
    try {
        const result = await parseCSV(Number(accountId.value), selectedFile.value)
        parsedRows.value = result.rows
        // Initialize checked state: checked by default, unchecked for duplicates and errors
        const checked = {}
        for (const row of result.rows) {
            checked[row.rowNumber] = !row.isDuplicate && !row.error
        }
        checkedRows.value = checked
    } catch (err) {
        parseError.value = err?.response?.data?.message || err?.message || 'Failed to parse CSV'
    } finally {
        isParsing.value = false
    }
}

/* --- Import --- */
const isSubmitting = ref(false)

const handleImport = async () => {
    if (!parsedRows.value) return
    const selectedRows = parsedRows.value.filter(
        (row) => checkedRows.value[row.rowNumber] && !row.error
    )
    if (selectedRows.length === 0) {
        toast.add({ severity: 'warn', summary: 'No rows selected', detail: 'Select at least one row to import.', life: 3000 })
        return
    }
    isSubmitting.value = true
    try {
        const result = await submitImport(Number(accountId.value), selectedRows)
        toast.add({
            severity: 'success',
            summary: 'Import complete',
            detail: `${result.created} transactions imported successfully.`,
            life: 4000
        })
        router.push(`/entries/${accountId.value}`)
    } catch (err) {
        toast.add({
            severity: 'error',
            summary: 'Import failed',
            detail: err?.response?.data?.message || err?.message || 'Failed to import transactions.',
            life: 5000
        })
    } finally {
        isSubmitting.value = false
    }
}

/* --- Cancel --- */
const handleCancel = () => {
    if (isPreview.value) {
        // Go back to upload state
        parsedRows.value = null
        checkedRows.value = {}
        selectedFile.value = null
        parseError.value = ''
    } else {
        router.push(`/entries/${accountId.value}`)
    }
}

const handleBack = () => {
    router.push(`/entries/${accountId.value}`)
}

/* --- Checked rows count --- */
const checkedCount = computed(() => {
    if (!parsedRows.value) return 0
    return parsedRows.value.filter((r) => checkedRows.value[r.rowNumber] && !r.error).length
})
</script>

<template>
    <div class="main-app-content">
        <div class="import-content">
            <!-- Header -->
            <div class="toolbar">
                <div class="toolbar-left">
                    <Button icon="pi pi-arrow-left" text rounded @click="handleBack" v-tooltip.bottom="'Back to entries'" class="mr-2" />
                    <h2 class="account-title">
                        Import CSV — {{ accountTitle }}
                    </h2>
                </div>
            </div>

            <!-- Upload State -->
            <div v-if="!isPreview" class="upload-section">
                <Card>
                    <template #title>Upload CSV File</template>
                    <template #content>
                        <div class="upload-form">
                            <div class="file-input-wrapper">
                                <input
                                    type="file"
                                    accept=".csv"
                                    @change="onFileChange"
                                    class="file-input"
                                />
                            </div>

                            <div class="upload-actions">
                                <Button
                                    label="Parse"
                                    icon="pi pi-upload"
                                    :loading="isParsing"
                                    :disabled="!selectedFile"
                                    @click="handleParse"
                                />
                                <Button
                                    label="Cancel"
                                    severity="secondary"
                                    @click="handleBack"
                                />
                            </div>

                            <Message v-if="parseError" severity="error" :closable="false" class="mt-3">
                                {{ parseError }}
                            </Message>
                        </div>
                    </template>
                </Card>
            </div>

            <!-- Preview State -->
            <div v-else class="preview-section">
                <!-- Summary Bar -->
                <div class="summary-bar">
                    <span class="summary-item summary-new">
                        <i class="pi pi-check-circle"></i> {{ summary.newCount }} new
                    </span>
                    <span class="summary-item summary-duplicate">
                        <i class="pi pi-copy"></i> {{ summary.duplicateCount }} duplicates
                    </span>
                    <span class="summary-item summary-error">
                        <i class="pi pi-exclamation-triangle"></i> {{ summary.errorCount }} errors
                    </span>
                </div>

                <!-- Preview Table -->
                <Card>
                    <template #content>
                        <DataTable
                            class="datatable-compact"
                            :value="tableEntries"
                            stripedRows
                            style="width: 100%"
                            :rowClass="getRowClass"
                        >
                            <!-- Checkbox column -->
                            <Column header="" style="width: 50px">
                                <template #body="{ data }">
                                    <Checkbox
                                        v-if="!data.importError"
                                        v-model="checkedRows[data.rowNumber]"
                                        :binary="true"
                                    />
                                    <i v-else class="pi pi-exclamation-triangle" style="color: var(--red-500)" v-tooltip.bottom="data.importError" />
                                </template>
                            </Column>

                            <!-- Type icon -->
                            <Column header="" style="width: 40px">
                                <template #body="{ data }">
                                    <i :class="getEntryTypeIcon(data.type)" style="font-size: 0.8rem" />
                                </template>
                            </Column>

                            <!-- Description -->
                            <Column field="description" header="Description">
                                <template #body="{ data }">
                                    <span v-tooltip.bottom="data.categoryId ? `Category: ${getCategoryPath(data.categoryId, data.type)}` : ''">
                                        {{ data.description }}
                                    </span>
                                </template>
                            </Column>

                            <!-- Date -->
                            <Column field="date" header="Date">
                                <template #body="{ data }">
                                    {{ formatDate(data.date) }}
                                </template>
                            </Column>

                            <!-- Amount -->
                            <Column field="Amount" header="Amount" bodyStyle="text-align: right">
                                <template #body="{ data }">
                                    <div class="amount" :class="data.type === 'expense' ? 'expense' : 'income'">
                                        <template v-if="data.type === 'expense'">-</template>
                                        <template v-else>+</template>
                                        {{ formatAmount(data.Amount) }}
                                    </div>
                                </template>
                            </Column>

                            <!-- Category -->
                            <Column header="Category">
                                <template #body="{ data }">
                                    {{ data.categoryId ? getCategoryPath(data.categoryId, data.type) : '—' }}
                                </template>
                            </Column>

                            <!-- Status -->
                            <Column header="Status" style="width: 120px">
                                <template #body="{ data }">
                                    <span v-if="data.importError" class="status-badge status-error" v-tooltip.bottom="data.importError">
                                        Error
                                    </span>
                                    <span v-else-if="data.isDuplicate" class="status-badge status-duplicate">
                                        Duplicate
                                    </span>
                                    <span v-else class="status-badge status-new">
                                        New
                                    </span>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Action buttons -->
                <div class="preview-actions">
                    <Button
                        :label="`Import Selected (${checkedCount})`"
                        icon="pi pi-check"
                        :loading="isSubmitting"
                        :disabled="checkedCount === 0"
                        @click="handleImport"
                    />
                    <Button
                        label="Cancel"
                        severity="secondary"
                        icon="pi pi-times"
                        @click="handleCancel"
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

.import-content {
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

.account-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-700);
}

.upload-section {
    padding: 2rem;
    max-width: 600px;
}

.upload-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.file-input-wrapper {
    margin-bottom: 0.5rem;
}

.file-input {
    width: 100%;
}

.upload-actions {
    display: flex;
    gap: 0.75rem;
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

.summary-new {
    color: var(--green-600);
}

.summary-duplicate {
    color: var(--yellow-700);
}

.summary-error {
    color: var(--red-600);
}

.preview-actions {
    display: flex;
    gap: 0.75rem;
    padding-top: 0.5rem;
}

/* Amount styling (matches AccountEntriesTable) */
.amount.expense {
    color: var(--red-500);
}

.amount.income {
    color: var(--green-500);
}

/* Status badges */
.status-badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-size: 0.8rem;
    font-weight: 600;
}

.status-new {
    background-color: var(--green-100);
    color: var(--green-700);
}

.status-duplicate {
    background-color: var(--yellow-100);
    color: var(--yellow-700);
}

.status-error {
    background-color: var(--red-100);
    color: var(--red-700);
}

/* Row styling for duplicates and errors */
:deep(.duplicate-row) {
    opacity: 0.6;
}

:deep(.error-row) {
    background-color: var(--red-50) !important;
}
</style>
