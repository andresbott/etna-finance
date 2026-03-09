<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import Card from 'primevue/card'
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
import Checkbox from 'primevue/checkbox'
import { useToast } from 'primevue/usetoast'
import { getProfiles, createProfile, updateProfile, deleteProfile, previewCSV, getCategoryRules, createCategoryRule, updateCategoryRule, deleteCategoryRule } from '@/lib/api/CsvImport'
import { useCategoryTree } from '@/composables/useCategoryTree'
import { useCategoryUtils } from '@/utils/categoryUtils'

const toast = useToast()
const router = useRouter()
const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()
const { getCategoryName } = useCategoryUtils()

// State
const profiles = ref([])
const isLoading = ref(false)
const showProfileDialog = ref(false)
const editingProfile = ref(null)
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

// Whether we have headers loaded (to show Select vs InputText)
const hasHeaders = computed(() => detectedHeaders.value.length > 0)

// Header options for Select dropdowns
const headerOptions = computed(() =>
    detectedHeaders.value.map(h => ({ label: h, value: h }))
)

// Options
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

// Load profiles
const loadProfiles = async () => {
    isLoading.value = true
    try {
        profiles.value = await getProfiles()
    } catch (error) {
        toast.add({
            severity: 'error',
            summary: 'Error',
            detail: 'Failed to load CSV profiles: ' + error.message,
            life: 3000
        })
    } finally {
        isLoading.value = false
    }
}

// Reset form
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

// Open create dialog
const openCreateDialog = () => {
    editingProfile.value = null
    resetForm()
    showProfileDialog.value = true
}

// Open edit dialog
const openEditDialog = (profile) => {
    editingProfile.value = profile
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
    showProfileDialog.value = true
}

// File selection — triggers auto-detection
watch(sampleFile, async (file) => {
    if (!file) return

    isLoadingFile.value = true
    try {
        // Send with empty separator to trigger auto-detection
        const result = await previewCSV(file, {})
        detectedHeaders.value = result.headers || []
        previewTotalRows.value = result.totalRows

        // Apply detected settings
        if (result.detectedSeparator) {
            formCsvSeparator.value = result.detectedSeparator
        }
        if (result.detectedSkipRows !== undefined) {
            formSkipRows.value = result.detectedSkipRows
        }
        if (result.detectedDateFormat) {
            formDateFormat.value = result.detectedDateFormat
        }

        // Apply detected column mappings
        if (result.detectedColumns) {
            const cols = result.detectedColumns
            if (cols.dateColumn) {
                formDateColumn.value = cols.dateColumn
            }
            if (cols.descriptionColumn) {
                formDescriptionColumn.value = cols.descriptionColumn
            }
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
            if (result.detectedDateFormat) {
                formDateFormat.value = result.detectedDateFormat
            }
        } catch (e) {
            toast.add({ severity: 'error', summary: 'Preview Error', detail: e.message, life: 3000 })
        } finally {
            isLoadingPreview.value = false
        }
    }, 500)
}

// Re-detect when separator or skip rows change manually
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
        // Clear column selections if headers changed
        previewRows.value = []
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to reload headers: ' + error.message, life: 3000 })
    } finally {
        isLoadingFile.value = false
    }
}

// When the date column changes, re-detect the date format
watch(formDateColumn, async (newVal) => {
    if (!newVal || !sampleFile.value) return
    try {
        const result = await previewCSV(sampleFile.value, {
            csvSeparator: formCsvSeparator.value,
            skipRows: formSkipRows.value ?? 0,
            dateColumn: newVal,
        })
        if (result.detectedDateFormat) {
            formDateFormat.value = result.detectedDateFormat
        }
    } catch (_) {
        // Date format detection is best-effort
    }
})

watch([formDateColumn, formDateFormat, formDescriptionColumn, formAmountMode,
       formAmountColumn, formCreditColumn, formDebitColumn], refreshPreview)

// Save profile
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
        if (editingProfile.value) {
            await updateProfile(editingProfile.value.id, payload)
            toast.add({ severity: 'success', summary: 'Success', detail: 'Profile updated successfully', life: 3000 })
        } else {
            await createProfile(payload)
            toast.add({ severity: 'success', summary: 'Success', detail: 'Profile created successfully', life: 3000 })
        }
        showProfileDialog.value = false
        await loadProfiles()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save profile: ' + error.message, life: 3000 })
    } finally {
        isSaving.value = false
    }
}

// Delete profile
const handleDeleteProfile = async (profile) => {
    if (!confirm(`Are you sure you want to delete the profile "${profile.name}"?`)) {
        return
    }

    try {
        await deleteProfile(profile.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Profile deleted successfully', life: 3000 })
        await loadProfiles()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete profile: ' + error.message, life: 3000 })
    }
}

// Display helpers
const getSeparatorLabel = (value) => {
    const opt = separatorOptions.find(o => o.value === value)
    return opt ? opt.label : value
}

// ============ Category Rules ============

const categoryRules = ref([])
const isLoadingRules = ref(false)
const showRuleDialog = ref(false)
const editingRule = ref(null)
const isSavingRule = ref(false)

// Rule form state
const formRulePattern = ref('')
const formRuleIsRegex = ref(false)
const formRuleCategoryId = ref(null)
const formRulePosition = ref(0)

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

const resolveCategoryName = (categoryId) => {
    let name = getCategoryName(categoryId, 'expense')
    if (name === 'Unknown') {
        name = getCategoryName(categoryId, 'income')
    }
    return name
}

const loadRules = async () => {
    isLoadingRules.value = true
    try {
        categoryRules.value = await getCategoryRules()
        categoryRules.value.sort((a, b) => a.position - b.position)
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load category rules: ' + error.message, life: 3000 })
    } finally {
        isLoadingRules.value = false
    }
}

const resetRuleForm = () => {
    formRulePattern.value = ''
    formRuleIsRegex.value = false
    formRuleCategoryId.value = null
    formRulePosition.value = categoryRules.value.length
}

const openCreateRuleDialog = () => {
    editingRule.value = null
    resetRuleForm()
    showRuleDialog.value = true
}

const openEditRuleDialog = (rule) => {
    editingRule.value = rule
    formRulePattern.value = rule.pattern
    formRuleIsRegex.value = rule.isRegex
    formRuleCategoryId.value = rule.categoryId
    formRulePosition.value = rule.position
    showRuleDialog.value = true
}

const handleSaveRule = async () => {
    if (!formRulePattern.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Pattern is required', life: 3000 })
        return
    }
    if (!formRuleCategoryId.value) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Category is required', life: 3000 })
        return
    }

    const payload = {
        pattern: formRulePattern.value.trim(),
        isRegex: formRuleIsRegex.value,
        categoryId: formRuleCategoryId.value,
        position: formRulePosition.value ?? 0
    }

    isSavingRule.value = true
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
        isSavingRule.value = false
    }
}

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
    loadProfiles()
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
                        <h1 class="text-2xl font-bold mb-2 text-color">CSV Import</h1>
                        <p class="text-color-secondary m-0 text-base">
                            Create and manage CSV import profiles to easily import transactions from different sources
                        </p>
                    </div>
                    <Button
                        label="New Profile"
                        icon="pi pi-plus"
                        @click="openCreateDialog"
                    />
                </div>

                <Card>
                    <template #content>
                        <DataTable
                            :value="profiles"
                            :loading="isLoading"
                            stripedRows
                            :paginator="profiles.length > 10"
                            :rows="10"
                            responsiveLayout="scroll"
                        >
                            <template #empty>
                                <div class="empty-state">
                                    <i class="pi pi-inbox"></i>
                                    <p>No CSV import profiles found</p>
                                    <Button
                                        label="Create Your First Profile"
                                        icon="pi pi-plus"
                                        @click="openCreateDialog"
                                        outlined
                                    />
                                </div>
                            </template>

                            <Column field="name" header="Profile Name" :sortable="true">
                                <template #body="{ data }">
                                    <div class="flex align-items-center gap-2 font-semibold">
                                        <i class="pi pi-file-import text-primary"></i>
                                        <span>{{ data.name }}</span>
                                    </div>
                                </template>
                            </Column>

                            <Column field="csvSeparator" header="CSV Separator">
                                <template #body="{ data }">
                                    <span class="separator-badge">{{ getSeparatorLabel(data.csvSeparator) }}</span>
                                </template>
                            </Column>

                            <Column field="dateFormat" header="Date Format" :sortable="true">
                                <template #body="{ data }">
                                    <span class="date-format">{{ data.dateFormat }}</span>
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
                                            v-tooltip.top="'Edit profile'"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            severity="danger"
                                            text
                                            rounded
                                            class="p-1"
                                            @click="handleDeleteProfile(data)"
                                            v-tooltip.top="'Delete profile'"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Category Matching Rules Section -->
                <div class="flex justify-content-between align-items-start mb-4 mt-5 gap-3">
                    <div>
                        <h2 class="text-xl font-bold mb-2 text-color">Category Matching Rules</h2>
                        <p class="text-color-secondary m-0 text-base">
                            Define rules to automatically assign categories to imported transactions based on description matching. Rules are evaluated in position order; the first match wins.
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
                            @click="openCreateRuleDialog"
                        />
                    </div>
                </div>

                <Card>
                    <template #content>
                        <DataTable
                            :value="categoryRules"
                            :loading="isLoadingRules"
                            stripedRows
                            :paginator="categoryRules.length > 10"
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
                                        @click="openCreateRuleDialog"
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
                                            @click="openEditRuleDialog(data)"
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
                    class="entry-dialog"
                >
                    <div class="rule-dialog-content">
                        <div class="field">
                            <label for="rulePattern">Pattern *</label>
                            <InputText
                                id="rulePattern"
                                v-model="formRulePattern"
                                placeholder="e.g., GROCERY or .*grocery.*"
                                class="w-full"
                            />
                            <small class="text-color-secondary">
                                Substring match is case-insensitive. Use regex for advanced patterns.
                            </small>
                        </div>

                        <div class="field">
                            <div class="flex align-items-center gap-2">
                                <Checkbox
                                    id="ruleIsRegex"
                                    v-model="formRuleIsRegex"
                                    :binary="true"
                                />
                                <label for="ruleIsRegex">Use Regular Expression</label>
                            </div>
                        </div>

                        <div class="field">
                            <label for="ruleCategory">Category *</label>
                            <Select
                                id="ruleCategory"
                                v-model="formRuleCategoryId"
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
                                v-model="formRulePosition"
                                :min="0"
                                class="w-full"
                            />
                            <small class="text-color-secondary">
                                Lower position = higher priority. First matching rule wins.
                            </small>
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
                                :loading="isSavingRule"
                                @click="handleSaveRule"
                            />
                        </div>
                    </div>
                </Dialog>

                <!-- Profile Edit/Create Dialog -->
                <Dialog
                    v-model:visible="showProfileDialog"
                    :header="editingProfile ? 'Edit CSV Profile' : 'Create CSV Profile'"
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
                                        <InputText
                                            id="profileName"
                                            v-model="formName"
                                            placeholder="e.g., Bank Statement Import"
                                            class="w-full"
                                        />
                                    </div>

                                    <div class="field">
                                        <label for="sampleFile">Sample CSV File</label>
                                        <FileInput
                                            v-model="sampleFile"
                                            accept=".csv,.txt"
                                            label="Choose CSV file"
                                        />
                                        <small class="text-color-secondary">Upload a sample to auto-detect separator, skip rows, and column headers</small>
                                    </div>

                                    <!-- Loading indicator for file analysis -->
                                    <div v-if="isLoadingFile" class="flex align-items-center gap-2 text-color-secondary">
                                        <i class="pi pi-spin pi-spinner"></i>
                                        <span>Analyzing CSV file...</span>
                                    </div>

                                    <!-- Detected/editable settings -->
                                    <div class="settings-row">
                                        <div class="field field--inline">
                                            <label for="csvSeparator">Separator</label>
                                            <Select
                                                id="csvSeparator"
                                                v-model="formCsvSeparator"
                                                :options="separatorOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                class="w-full"
                                                @change="onSettingsChange"
                                            />
                                        </div>

                                        <div class="field field--inline">
                                            <label for="skipRows">Skip Rows</label>
                                            <InputNumber
                                                id="skipRows"
                                                v-model="formSkipRows"
                                                :min="0"
                                                class="w-full"
                                                @input="onSettingsChange"
                                            />
                                        </div>

                                        <div class="field field--inline">
                                            <label for="dateFormat">Date Format</label>
                                            <Select
                                                id="dateFormat"
                                                v-model="formDateFormat"
                                                :options="dateFormatOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                class="w-full"
                                            />
                                        </div>
                                    </div>

                                    <!-- Column mapping — shown when headers are available or when editing -->
                                    <div v-if="hasHeaders || editingProfile" class="column-mapping">
                                        <h4 class="text-base font-semibold mb-2">Column Mapping</h4>

                                        <div class="field">
                                            <label for="dateColumn">Date Column *</label>
                                            <Select
                                                v-if="hasHeaders"
                                                id="dateColumn"
                                                v-model="formDateColumn"
                                                :options="headerOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                placeholder="Select date column"
                                                class="w-full"
                                            />
                                            <InputText
                                                v-else
                                                id="dateColumn"
                                                v-model="formDateColumn"
                                                placeholder="CSV header name, e.g. Date"
                                                class="w-full"
                                            />
                                        </div>

                                        <div class="field">
                                            <label for="descriptionColumn">Description Column *</label>
                                            <Select
                                                v-if="hasHeaders"
                                                id="descriptionColumn"
                                                v-model="formDescriptionColumn"
                                                :options="headerOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                placeholder="Select description column"
                                                class="w-full"
                                            />
                                            <InputText
                                                v-else
                                                id="descriptionColumn"
                                                v-model="formDescriptionColumn"
                                                placeholder="CSV header name, e.g. Description"
                                                class="w-full"
                                            />
                                        </div>

                                        <div class="field">
                                            <label>Amount Mode</label>
                                            <div class="flex gap-3 align-items-center">
                                                <div class="flex align-items-center gap-1">
                                                    <RadioButton
                                                        v-model="formAmountMode"
                                                        inputId="amountModeSingle"
                                                        value="single"
                                                    />
                                                    <label for="amountModeSingle">Single column</label>
                                                </div>
                                                <div class="flex align-items-center gap-1">
                                                    <RadioButton
                                                        v-model="formAmountMode"
                                                        inputId="amountModeSplit"
                                                        value="split"
                                                    />
                                                    <label for="amountModeSplit">Split credit/debit</label>
                                                </div>
                                            </div>
                                        </div>

                                        <div v-if="formAmountMode === 'single'" class="field">
                                            <label for="amountColumn">Amount Column *</label>
                                            <Select
                                                v-if="hasHeaders"
                                                id="amountColumn"
                                                v-model="formAmountColumn"
                                                :options="headerOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                placeholder="Select amount column"
                                                class="w-full"
                                            />
                                            <InputText
                                                v-else
                                                id="amountColumn"
                                                v-model="formAmountColumn"
                                                placeholder="CSV header name, e.g. Amount"
                                                class="w-full"
                                            />
                                        </div>

                                        <div v-if="formAmountMode === 'split'" class="field">
                                            <label for="creditColumn">Credit Column *</label>
                                            <Select
                                                v-if="hasHeaders"
                                                id="creditColumn"
                                                v-model="formCreditColumn"
                                                :options="headerOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                placeholder="Select credit column"
                                                class="w-full"
                                            />
                                            <InputText
                                                v-else
                                                id="creditColumn"
                                                v-model="formCreditColumn"
                                                placeholder="CSV header name, e.g. Credit"
                                                class="w-full"
                                            />
                                        </div>

                                        <div v-if="formAmountMode === 'split'" class="field">
                                            <label for="debitColumn">Debit Column *</label>
                                            <Select
                                                v-if="hasHeaders"
                                                id="debitColumn"
                                                v-model="formDebitColumn"
                                                :options="headerOptions"
                                                optionLabel="label"
                                                optionValue="value"
                                                placeholder="Select debit column"
                                                class="w-full"
                                            />
                                            <InputText
                                                v-else
                                                id="debitColumn"
                                                v-model="formDebitColumn"
                                                placeholder="CSV header name, e.g. Debit"
                                                class="w-full"
                                            />
                                        </div>
                                    </div>

                                    <div class="flex justify-content-end gap-2 mt-3">
                                        <Button
                                            label="Cancel"
                                            severity="secondary"
                                            text
                                            @click="showProfileDialog = false"
                                        />
                                        <Button
                                            :label="editingProfile ? 'Update' : 'Create'"
                                            icon="pi pi-check"
                                            :loading="isSaving"
                                            @click="handleSaveProfile"
                                        />
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
                                    <i class="pi pi-table"></i>
                                    <p>No preview available yet. Upload a CSV file and configure column mappings in the Settings tab.</p>
                                </div>
                            </TabPanel>
                        </TabPanels>
                    </Tabs>
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

.date-format {
    font-family: monospace;
    background-color: var(--surface-100);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.9rem;
}

.separator-badge {
    font-family: monospace;
    background-color: var(--surface-100);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.9rem;
}

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

:deep(.p-card-content) {
    padding: 1.5rem;
}
</style>
