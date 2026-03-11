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
import CategorySelect from '@/components/common/CategorySelect.vue'
import Checkbox from 'primevue/checkbox'
import Divider from 'primevue/divider'
import { useToast } from 'primevue/usetoast'
import { getProfiles, createProfile, updateProfile, deleteProfile, previewCSV, getCategoryRuleGroups, createCategoryRuleGroup, updateCategoryRuleGroup, deleteCategoryRuleGroup, createCategoryRulePattern, updateCategoryRulePattern, deleteCategoryRulePattern } from '@/lib/api/CsvImport'
import { useCategoryUtils } from '@/utils/categoryUtils'

const toast = useToast()
const router = useRouter()
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

// ============ Category Rule Groups ============

const categoryRuleGroups = ref([])
const isLoadingRules = ref(false)
const showGroupDialog = ref(false)
const editingGroup = ref(null)
const isSavingRule = ref(false)

// Group form state
const formGroupName = ref('')
const formGroupCategoryId = ref(null)
const formGroupPriority = ref(0)
const formGroupPatterns = ref([])

// New pattern inline form
const newPatternValue = ref('')
const newPatternIsRegex = ref(false)


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
        categoryRuleGroups.value = await getCategoryRuleGroups()
        categoryRuleGroups.value.sort((a, b) => a.priority - b.priority)
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load category rule groups: ' + error.message, life: 3000 })
    } finally {
        isLoadingRules.value = false
    }
}

const openCreateGroupDialog = () => {
    editingGroup.value = null
    formGroupName.value = ''
    formGroupCategoryId.value = null
    formGroupPriority.value = categoryRuleGroups.value.length
    formGroupPatterns.value = []
    newPatternValue.value = ''
    newPatternIsRegex.value = false
    showGroupDialog.value = true
}

const openEditGroupDialog = (group) => {
    editingGroup.value = group
    formGroupName.value = group.name
    formGroupCategoryId.value = group.categoryId
    formGroupPriority.value = group.priority
    formGroupPatterns.value = (group.patterns || []).map(p => ({ ...p }))
    newPatternValue.value = ''
    newPatternIsRegex.value = false
    showGroupDialog.value = true
}

const addPattern = () => {
    if (!newPatternValue.value.trim()) return
    formGroupPatterns.value.push({
        id: null,
        pattern: newPatternValue.value.trim(),
        isRegex: newPatternIsRegex.value,
    })
    newPatternValue.value = ''
    newPatternIsRegex.value = false
}

const removePattern = (index) => {
    formGroupPatterns.value.splice(index, 1)
}

const handleSaveGroup = async () => {
    if (!formGroupName.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Name is required', life: 3000 })
        return
    }
    if (!formGroupCategoryId.value) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Category is required', life: 3000 })
        return
    }

    isSavingRule.value = true
    try {
        if (editingGroup.value) {
            await updateCategoryRuleGroup(editingGroup.value.id, {
                name: formGroupName.value.trim(),
                categoryId: formGroupCategoryId.value,
                priority: formGroupPriority.value ?? 0,
                patterns: editingGroup.value.patterns || [],
            })

            // Sync patterns: delete removed, update changed, create new
            const oldPatterns = editingGroup.value.patterns || []
            const newPatterns = formGroupPatterns.value
            const newIds = new Set(newPatterns.filter(p => p.id).map(p => p.id))

            for (const old of oldPatterns) {
                if (!newIds.has(old.id)) {
                    await deleteCategoryRulePattern(editingGroup.value.id, old.id)
                }
            }
            for (const p of newPatterns) {
                if (p.id) {
                    const old = oldPatterns.find(o => o.id === p.id)
                    if (old && (old.pattern !== p.pattern || old.isRegex !== p.isRegex)) {
                        await updateCategoryRulePattern(editingGroup.value.id, p.id, { pattern: p.pattern, isRegex: p.isRegex })
                    }
                } else {
                    await createCategoryRulePattern(editingGroup.value.id, { pattern: p.pattern, isRegex: p.isRegex })
                }
            }

            toast.add({ severity: 'success', summary: 'Success', detail: 'Group updated successfully', life: 3000 })
        } else {
            const created = await createCategoryRuleGroup({
                name: formGroupName.value.trim(),
                categoryId: formGroupCategoryId.value,
                priority: formGroupPriority.value ?? 0,
                patterns: [],
            })
            for (const p of formGroupPatterns.value) {
                await createCategoryRulePattern(created.id, { pattern: p.pattern, isRegex: p.isRegex })
            }
            toast.add({ severity: 'success', summary: 'Success', detail: 'Group created successfully', life: 3000 })
        }
        showGroupDialog.value = false
        await loadRules()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save group: ' + error.message, life: 3000 })
    } finally {
        isSavingRule.value = false
    }
}

const handleDeleteGroup = async (group) => {
    if (!confirm(`Are you sure you want to delete the group "${group.name}" and all its patterns?`)) {
        return
    }

    try {
        await deleteCategoryRuleGroup(group.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Group deleted successfully', life: 3000 })
        await loadRules()
    } catch (error) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete group: ' + error.message, life: 3000 })
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

                <!-- Category Matching Rule Groups Section -->
                <div class="flex justify-content-between align-items-start mb-4 mt-5 gap-3">
                    <div>
                        <h2 class="text-xl font-bold mb-2 text-color">Category Matching Rules</h2>
                        <p class="text-color-secondary m-0 text-base">
                            Define rule groups to automatically assign categories to imported transactions based on description matching. Groups are evaluated in priority order; the first match wins.
                        </p>
                    </div>
                    <div class="flex gap-2" style="white-space: nowrap; flex-shrink: 0">
                        <Button
                            label="Re-apply Rules"
                            icon="pi pi-sync"
                            severity="secondary"
                            @click="router.push('/setup/reapply-rules')"
                        />
                        <Button
                            label="New Group"
                            icon="pi pi-plus"
                            @click="openCreateGroupDialog"
                        />
                    </div>
                </div>

                <Card>
                    <template #content>
                        <DataTable
                            :value="categoryRuleGroups"
                            :loading="isLoadingRules"
                            dataKey="id"
                            stripedRows
                            :paginator="categoryRuleGroups.length > 10"
                            :rows="10"
                            responsiveLayout="scroll"
                        >
                            <template #empty>
                                <div class="empty-state">
                                    <i class="pi pi-inbox"></i>
                                    <p>No category matching rule groups found</p>
                                    <Button
                                        label="Create Your First Group"
                                        icon="pi pi-plus"
                                        @click="openCreateGroupDialog"
                                        outlined
                                    />
                                </div>
                            </template>

                            <Column field="priority" header="Priority" :sortable="true" style="width: 100px">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.priority }}</span>
                                </template>
                            </Column>

                            <Column field="name" header="Name">
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ data.name }}</span>
                                </template>
                            </Column>

                            <Column field="categoryId" header="Category">
                                <template #body="{ data }">
                                    {{ resolveCategoryName(data.categoryId) }}
                                </template>
                            </Column>

                            <Column header="Patterns" style="width: 120px">
                                <template #body="{ data }">
                                    <Tag :value="String(data.patterns?.length || 0)" severity="info" />
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
                                            @click="openEditGroupDialog(data)"
                                            v-tooltip.top="'Edit group'"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            severity="danger"
                                            text
                                            rounded
                                            class="p-1"
                                            @click="handleDeleteGroup(data)"
                                            v-tooltip.top="'Delete group'"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Group Edit/Create Dialog (with inline patterns) -->
                <Dialog
                    v-model:visible="showGroupDialog"
                    :header="editingGroup ? 'Edit Rule Group' : 'Create Rule Group'"
                    :modal="true"
                    :closable="true"
                    class="entry-dialog entry-dialog--wide"
                >
                    <div class="rule-dialog-content">
                        <div class="field">
                            <label for="groupName">Name *</label>
                            <InputText
                                id="groupName"
                                v-model="formGroupName"
                                placeholder="e.g., Amazon, Grocery Stores"
                                class="w-full"
                            />
                        </div>

                        <CategorySelect
                            v-model="formGroupCategoryId"
                            type="all"
                            label="Category *"
                        />

                        <div class="field">
                            <label for="groupPriority">Priority</label>
                            <InputNumber
                                id="groupPriority"
                                v-model="formGroupPriority"
                                :min="0"
                                class="w-full"
                            />
                            <small class="text-color-secondary">
                                Lower value = higher priority. First matching group wins.
                            </small>
                        </div>

                        <Divider />

                        <!-- Inline pattern management -->
                        <div class="patterns-section">
                            <label class="font-semibold text-color">Patterns</label>

                            <div class="patterns-list" v-if="formGroupPatterns.length > 0">
                                <div
                                    v-for="(p, index) in formGroupPatterns"
                                    :key="index"
                                    class="pattern-row"
                                >
                                    <span class="pattern-text">{{ p.pattern }}</span>
                                    <Tag
                                        :value="p.isRegex ? 'Regex' : 'Substring'"
                                        :severity="p.isRegex ? 'warn' : 'info'"
                                        class="flex-shrink-0"
                                    />
                                    <Button
                                        icon="pi pi-times"
                                        severity="danger"
                                        text
                                        rounded
                                        size="small"
                                        @click="removePattern(index)"
                                        v-tooltip.top="'Remove pattern'"
                                    />
                                </div>
                            </div>
                            <div v-else class="text-color-secondary text-sm">
                                No patterns yet. Add one below.
                            </div>

                            <div class="add-pattern-row">
                                <InputText
                                    v-model="newPatternValue"
                                    placeholder="e.g., AMAZON or .*amazon.*"
                                    class="flex-grow-1"
                                    size="small"
                                    @keydown.enter="addPattern"
                                />
                                <div class="flex align-items-center gap-2 flex-shrink-0">
                                    <Checkbox
                                        v-model="newPatternIsRegex"
                                        :binary="true"
                                        inputId="newPatternRegex"
                                    />
                                    <label for="newPatternRegex" class="text-sm white-space-nowrap">Regex</label>
                                </div>
                                <Button
                                    label="Add"
                                    icon="pi pi-plus"
                                    outlined
                                    @click="addPattern"
                                    :disabled="!newPatternValue.trim()"
                                />
                            </div>
                        </div>

                        <div class="flex justify-content-end gap-2 mt-3">
                            <Button
                                label="Cancel"
                                severity="secondary"
                                text
                                @click="showGroupDialog = false"
                            />
                            <Button
                                :label="editingGroup ? 'Update' : 'Create'"
                                icon="pi pi-check"
                                :loading="isSavingRule"
                                @click="handleSaveGroup"
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

.patterns-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.patterns-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.pattern-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.5rem;
    border-radius: 6px;
    background-color: var(--surface-50);
    border-left: 3px solid var(--primary-color);

    .pattern-text {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: monospace;
        font-size: 0.9rem;
        color: var(--primary-700, var(--primary-color));
    }
}

.add-pattern-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

:deep(.p-card-content) {
    padding: 1.5rem;
}
</style>
