<script setup>
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import RadioButton from 'primevue/radiobutton'
import Tag from 'primevue/tag'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import FileInput from '@/components/common/FileInput.vue'
import { useToast } from 'primevue/usetoast'
import { getProfiles, previewCSV } from '@/lib/api/CsvImport'
import { useCsvProfileSave } from '@/composables/useCsvProfileSave'

const props = defineProps({
    visible: { type: Boolean, default: false },
    accountId: { type: Number, required: true },
    profileId: { type: Number, default: 0 }, // 0 => create mode
})
const emit = defineEmits(['update:visible'])

const toast = useToast()
const { saveProfile } = useCsvProfileSave()

const isSaving = ref(false)

// Form state
const formName = ref('')
const formCsvSeparator = ref(',')
const formSkipRows = ref(0)
const formDateColumn = ref('')
const formDateFormat = ref('2006-01-02')
const formDescriptionColumn = ref('')
const formAmountColumn = ref('')
const formAmountMode = ref('single')
const formCreditColumn = ref('')
const formDebitColumn = ref('')

// Tab & file/preview state
const activeTab = ref('settings')
const sampleFile = ref(null)
const detectedHeaders = ref([])
const previewRows = ref([])
const previewTotalRows = ref(0)
const isLoadingFile = ref(false)
const isLoadingPreview = ref(false)

const isEdit = computed(() => props.profileId > 0)
const hasHeaders = computed(() => detectedHeaders.value.length > 0)
const headerOptions = computed(() => detectedHeaders.value.map(h => ({ label: h, value: h })))

const separatorOptions = [
    { label: 'Comma (,)', value: ',' },
    { label: 'Semicolon (;)', value: ';' },
    { label: 'Tab', value: '\t' }
]

const dateFormatOptions = [
    { label: '2006-01-02 (YYYY-MM-DD)', value: '2006-01-02' },
    { label: '02/01/2006 (DD/MM/YYYY)', value: '02/01/2006' },
    { label: '01/02/2006 (MM/DD/YYYY)', value: '01/02/2006' },
    { label: '02.01.2006 (DD.MM.YYYY)', value: '02.01.2006' }
]

const resetForm = () => {
    formName.value = ''
    formCsvSeparator.value = ','
    formSkipRows.value = 0
    formDateColumn.value = ''
    formDateFormat.value = '2006-01-02'
    formDescriptionColumn.value = ''
    formAmountColumn.value = ''
    formAmountMode.value = 'single'
    formCreditColumn.value = ''
    formDebitColumn.value = ''
    sampleFile.value = null
    detectedHeaders.value = []
    previewRows.value = []
    previewTotalRows.value = 0
    activeTab.value = 'settings'
}

const populateFromProfile = (profile) => {
    formName.value = profile.name
    formCsvSeparator.value = profile.csvSeparator
    formSkipRows.value = profile.skipRows
    formDateColumn.value = profile.dateColumn
    formDateFormat.value = profile.dateFormat
    formDescriptionColumn.value = profile.descriptionColumn
    formAmountColumn.value = profile.amountColumn || ''
    formAmountMode.value = profile.amountMode || 'single'
    formCreditColumn.value = profile.creditColumn || ''
    formDebitColumn.value = profile.debitColumn || ''
    sampleFile.value = null
    detectedHeaders.value = []
    previewRows.value = []
}

// Load form contents whenever the dialog opens.
watch(() => props.visible, async (v) => {
    if (!v) return
    resetForm()
    if (isEdit.value) {
        try {
            const profiles = await getProfiles()
            const profile = profiles.find(p => p.id === props.profileId)
            if (profile) populateFromProfile(profile)
        } catch (error) {
            toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load profile: ' + error.message, life: 3000 })
        }
    }
})

// File selection — triggers auto-detection
watch(sampleFile, async (file) => {
    if (!file) return
    isLoadingFile.value = true
    try {
        const result = await previewCSV(file, {})
        detectedHeaders.value = result.headers || []
        previewTotalRows.value = result.totalRows
        if (result.detectedSeparator) formCsvSeparator.value = result.detectedSeparator
        if (result.detectedSkipRows !== undefined) formSkipRows.value = result.detectedSkipRows
        if (result.detectedDateFormat) formDateFormat.value = result.detectedDateFormat
        if (result.detectedColumns) {
            const cols = result.detectedColumns
            if (cols.dateColumn) formDateColumn.value = cols.dateColumn
            if (cols.descriptionColumn) formDescriptionColumn.value = cols.descriptionColumn
            if (cols.amountMode === 'split') {
                formAmountMode.value = 'split'
                if (cols.creditColumn) formCreditColumn.value = cols.creditColumn
                if (cols.debitColumn) formDebitColumn.value = cols.debitColumn
            } else if (cols.amountColumn) {
                formAmountMode.value = 'single'
                formAmountColumn.value = cols.amountColumn
            }
        }
        if (detectedHeaders.value.length === 0) {
            toast.add({ severity: 'warn', summary: 'No headers', detail: 'No headers detected in the file.', life: 3000 })
        }
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to analyze CSV: ' + error.message, life: 3000 })
    } finally {
        isLoadingFile.value = false
    }
})

// Preview refresh with debounce
let previewTimeout = null
const refreshPreview = () => {
    if (previewTimeout) clearTimeout(previewTimeout)
    previewTimeout = setTimeout(async () => {
        if (!sampleFile.value) return
        if (!formDateColumn.value && !formDescriptionColumn.value) return
        isLoadingPreview.value = true
        try {
            const result = await previewCSV(sampleFile.value, {
                csvSeparator: formCsvSeparator.value,
                skipRows: formSkipRows.value ?? 0,
                dateColumn: formDateColumn.value,
                dateFormat: formDateFormat.value,
                descriptionColumn: formDescriptionColumn.value,
                amountMode: formAmountMode.value,
                amountColumn: formAmountMode.value === 'single' ? formAmountColumn.value : undefined,
                creditColumn: formAmountMode.value === 'split' ? formCreditColumn.value : undefined,
                debitColumn: formAmountMode.value === 'split' ? formDebitColumn.value : undefined,
            })
            previewRows.value = result.rows || []
            previewTotalRows.value = result.totalRows
            if (result.detectedDateFormat) formDateFormat.value = result.detectedDateFormat
        } catch (e) {
            toast.add({ severity: 'error', summary: 'Preview Error', detail: e.message, life: 3000 })
        } finally {
            isLoadingPreview.value = false
        }
    }, 500)
}

const onSettingsChange = async () => {
    if (!sampleFile.value) return
    isLoadingFile.value = true
    try {
        const result = await previewCSV(sampleFile.value, {
            csvSeparator: formCsvSeparator.value,
            skipRows: formSkipRows.value ?? 0,
        })
        detectedHeaders.value = result.headers || []
        previewTotalRows.value = result.totalRows
        previewRows.value = []
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to reload headers: ' + error.message, life: 3000 })
    } finally {
        isLoadingFile.value = false
    }
}

watch(formDateColumn, async (newVal) => {
    if (!newVal || !sampleFile.value) return
    try {
        const result = await previewCSV(sampleFile.value, {
            csvSeparator: formCsvSeparator.value,
            skipRows: formSkipRows.value ?? 0,
            dateColumn: newVal,
        })
        if (result.detectedDateFormat) formDateFormat.value = result.detectedDateFormat
    } catch (_) {
        // Date format detection is best-effort
    }
})

watch([formDateColumn, formDateFormat, formDescriptionColumn, formAmountMode,
       formAmountColumn, formCreditColumn, formDebitColumn], refreshPreview)

const handleSaveProfile = async () => {
    if (!formName.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation', detail: 'Profile name is required', life: 3000 })
        return
    }
    if (!formDateColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation', detail: 'Date column is required', life: 3000 })
        return
    }
    if (!formDescriptionColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation', detail: 'Description column is required', life: 3000 })
        return
    }
    if (formAmountMode.value === 'single' && !formAmountColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation', detail: 'Amount column is required', life: 3000 })
        return
    }
    if (formAmountMode.value === 'split') {
        if (!formCreditColumn.value.trim()) {
            toast.add({ severity: 'warn', summary: 'Validation', detail: 'Credit column is required', life: 3000 })
            return
        }
        if (!formDebitColumn.value.trim()) {
            toast.add({ severity: 'warn', summary: 'Validation', detail: 'Debit column is required', life: 3000 })
            return
        }
    }

    const payload = {
        name: formName.value.trim(),
        csvSeparator: formCsvSeparator.value,
        skipRows: formSkipRows.value ?? 0,
        dateColumn: formDateColumn.value.trim(),
        dateFormat: formDateFormat.value,
        descriptionColumn: formDescriptionColumn.value.trim(),
        amountMode: formAmountMode.value,
        amountColumn: formAmountMode.value === 'single' ? formAmountColumn.value.trim() : '',
        creditColumn: formAmountMode.value === 'split' ? formCreditColumn.value.trim() : '',
        debitColumn: formAmountMode.value === 'split' ? formDebitColumn.value.trim() : '',
    }

    isSaving.value = true
    try {
        await saveProfile({ accountId: props.accountId, profileId: props.profileId, payload })
        toast.add({
            severity: 'success',
            summary: 'Success',
            detail: isEdit.value ? 'Profile updated successfully' : 'Profile created successfully',
            life: 3000,
        })
        emit('update:visible', false)
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save profile: ' + error.message, life: 3000 })
    } finally {
        isSaving.value = false
    }
}
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :header="isEdit ? 'Edit CSV Profile' : 'Create CSV Profile'"
        :modal="true"
        :closable="true"
        class="entry-dialog entry-dialog--wide"
    >
        <Tabs v-model:value="activeTab">
            <TabList>
                <Tab value="settings">Settings</Tab>
                <Tab value="preview" :disabled="previewRows.length === 0">
                    Preview
                    <Tag v-if="previewRows.length > 0" :value="String(previewRows.length)" severity="info" rounded class="ml-2" />
                </Tab>
            </TabList>
            <TabPanels>
                <TabPanel value="settings">
                    <div class="profile-dialog-content">
                        <div class="field">
                            <label for="profileName">Profile Name *</label>
                            <InputText id="profileName" v-model="formName" placeholder="e.g., Bank Statement Import" class="w-full" />
                        </div>

                        <div class="field">
                            <label for="sampleFile">Sample CSV File</label>
                            <FileInput v-model="sampleFile" accept=".csv,.txt" label="Choose CSV file" />
                            <small class="text-color-secondary">Upload a sample to auto-detect separator, skip rows, and column headers</small>
                        </div>

                        <div v-if="isLoadingFile" class="flex align-items-center gap-2 text-color-secondary">
                            <i class="ti ti-loader-2 spin-icon"></i>
                            <span>Analyzing CSV file...</span>
                        </div>

                        <div class="settings-row">
                            <div class="field field--inline">
                                <label for="csvSeparator">Separator</label>
                                <Select id="csvSeparator" v-model="formCsvSeparator" :options="separatorOptions" optionLabel="label" optionValue="value" class="w-full" @change="onSettingsChange" />
                            </div>
                            <div class="field field--inline">
                                <label for="skipRows">Skip Rows</label>
                                <InputNumber id="skipRows" v-model="formSkipRows" :min="0" class="w-full" @input="onSettingsChange" />
                            </div>
                            <div class="field field--inline">
                                <label for="dateFormat">Date Format</label>
                                <Select id="dateFormat" v-model="formDateFormat" :options="dateFormatOptions" optionLabel="label" optionValue="value" class="w-full" />
                            </div>
                        </div>

                        <div v-if="hasHeaders || isEdit" class="column-mapping">
                            <h4 class="text-base font-semibold mb-2">Column Mapping</h4>

                            <div class="field">
                                <label for="dateColumn">Date Column *</label>
                                <Select v-if="hasHeaders" id="dateColumn" v-model="formDateColumn" :options="headerOptions" optionLabel="label" optionValue="value" placeholder="Select date column" class="w-full" />
                                <InputText v-else id="dateColumn" v-model="formDateColumn" placeholder="CSV header name, e.g. Date" class="w-full" />
                            </div>

                            <div class="field">
                                <label for="descriptionColumn">Description Column *</label>
                                <Select v-if="hasHeaders" id="descriptionColumn" v-model="formDescriptionColumn" :options="headerOptions" optionLabel="label" optionValue="value" placeholder="Select description column" class="w-full" />
                                <InputText v-else id="descriptionColumn" v-model="formDescriptionColumn" placeholder="CSV header name, e.g. Description" class="w-full" />
                            </div>

                            <div class="field">
                                <label>Amount Mode</label>
                                <div class="flex gap-3 align-items-center">
                                    <div class="flex align-items-center gap-1">
                                        <RadioButton v-model="formAmountMode" inputId="amountModeSingle" value="single" />
                                        <label for="amountModeSingle">Single column</label>
                                    </div>
                                    <div class="flex align-items-center gap-1">
                                        <RadioButton v-model="formAmountMode" inputId="amountModeSplit" value="split" />
                                        <label for="amountModeSplit">Split credit/debit</label>
                                    </div>
                                </div>
                            </div>

                            <div v-if="formAmountMode === 'single'" class="field">
                                <label for="amountColumn">Amount Column *</label>
                                <Select v-if="hasHeaders" id="amountColumn" v-model="formAmountColumn" :options="headerOptions" optionLabel="label" optionValue="value" placeholder="Select amount column" class="w-full" />
                                <InputText v-else id="amountColumn" v-model="formAmountColumn" placeholder="CSV header name, e.g. Amount" class="w-full" />
                            </div>

                            <div v-if="formAmountMode === 'split'" class="field">
                                <label for="creditColumn">Credit Column *</label>
                                <Select v-if="hasHeaders" id="creditColumn" v-model="formCreditColumn" :options="headerOptions" optionLabel="label" optionValue="value" placeholder="Select credit column" class="w-full" />
                                <InputText v-else id="creditColumn" v-model="formCreditColumn" placeholder="CSV header name, e.g. Credit" class="w-full" />
                            </div>

                            <div v-if="formAmountMode === 'split'" class="field">
                                <label for="debitColumn">Debit Column *</label>
                                <Select v-if="hasHeaders" id="debitColumn" v-model="formDebitColumn" :options="headerOptions" optionLabel="label" optionValue="value" placeholder="Select debit column" class="w-full" />
                                <InputText v-else id="debitColumn" v-model="formDebitColumn" placeholder="CSV header name, e.g. Debit" class="w-full" />
                            </div>
                        </div>

                        <div class="flex justify-content-end gap-2 mt-3">
                            <Button label="Cancel" severity="secondary" text @click="$emit('update:visible', false)" />
                            <Button :label="isEdit ? 'Update' : 'Create'" icon="ti ti-check" :loading="isSaving" @click="handleSaveProfile" />
                        </div>
                    </div>
                </TabPanel>
                <TabPanel value="preview">
                    <div v-if="previewRows.length > 0" class="mt-2">
                        <h4 class="text-base font-semibold mb-2">Preview ({{ previewRows.length }} of {{ previewTotalRows }} rows)</h4>
                        <DataTable :value="previewRows" :loading="isLoadingPreview" stripedRows size="small">
                            <Column field="rowNumber" header="#" style="width: 50px" />
                            <Column field="date" header="Date" />
                            <Column field="description" header="Description" />
                            <Column field="amount" header="Amount">
                                <template #body="{ data }">
                                    <span :class="data.amount >= 0 ? 'text-green-500' : 'text-red-500'">
                                        {{ data.amount >= 0 ? '+' : '' }}{{ data.amount.toFixed(2) }}
                                    </span>
                                </template>
                            </Column>
                            <Column field="type" header="Type" style="width: 80px">
                                <template #body="{ data }">
                                    <Tag :value="data.type" :severity="data.type === 'income' ? 'success' : 'danger'" />
                                </template>
                            </Column>
                            <Column field="error" header="Status">
                                <template #body="{ data }">
                                    <Tag v-if="data.error" :value="data.error" severity="danger" />
                                    <Tag v-else value="OK" severity="success" />
                                </template>
                            </Column>
                        </DataTable>
                    </div>
                    <div v-else class="empty-preview">
                        <i class="ti ti-table"></i>
                        <p>No preview available yet. Upload a CSV file and configure column mappings in the Settings tab.</p>
                    </div>
                </TabPanel>
            </TabPanels>
        </Tabs>
    </Dialog>
</template>

<style scoped lang="scss">
.profile-dialog-content {
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

.settings-row {
    display: flex;
    gap: 1rem;

    .field--inline {
        flex: 1;
    }
}

.column-mapping {
    border-top: 1px solid var(--surface-200);
    padding-top: 1rem;
}

.empty-preview {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-color-secondary);

    i {
        font-size: 3rem;
        margin-bottom: 1rem;
        opacity: 0.5;
    }

    p {
        font-size: 1rem;
    }
}
</style>
