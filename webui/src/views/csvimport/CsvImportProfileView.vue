<script setup>
import { ref, onMounted } from 'vue'
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
import { useToast } from 'primevue/usetoast'
import { getProfiles, createProfile, updateProfile, deleteProfile } from '@/lib/api/CsvImport'

const toast = useToast()

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
    formAmountColumn.value = profile.amountColumn
    showProfileDialog.value = true
}

// Save profile
const handleSaveProfile = async () => {
    if (!formName.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Profile name is required', life: 3000 })
        return
    }
    if (!formDateColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Date column is required', life: 3000 })
        return
    }
    if (!formDescriptionColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Description column is required', life: 3000 })
        return
    }
    if (!formAmountColumn.value.trim()) {
        toast.add({ severity: 'warn', summary: 'Validation Error', detail: 'Amount column is required', life: 3000 })
        return
    }

    const payload = {
        name: formName.value.trim(),
        csvSeparator: formCsvSeparator.value,
        skipRows: formSkipRows.value ?? 0,
        dateColumn: formDateColumn.value.trim(),
        dateFormat: formDateFormat.value,
        descriptionColumn: formDescriptionColumn.value.trim(),
        amountColumn: formAmountColumn.value.trim()
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

onMounted(() => {
    loadProfiles()
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
                        <h1 class="text-2xl font-bold mb-2 text-color">CSV Import Profiles</h1>
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

                <!-- Profile Edit/Create Dialog -->
                <Dialog
                    v-model:visible="showProfileDialog"
                    :header="editingProfile ? 'Edit CSV Profile' : 'Create CSV Profile'"
                    :modal="true"
                    :closable="true"
                    class="entry-dialog entry-dialog--wide"
                >
                    <div class="profile-dialog-content">
                        <div class="field">
                            <label for="profileName">Name *</label>
                            <InputText
                                id="profileName"
                                v-model="formName"
                                placeholder="e.g., Bank Statement Import"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <label for="csvSeparator">CSV Separator</label>
                            <Select
                                id="csvSeparator"
                                v-model="formCsvSeparator"
                                :options="separatorOptions"
                                optionLabel="label"
                                optionValue="value"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <label for="skipRows">Skip Rows</label>
                            <InputNumber
                                id="skipRows"
                                v-model="formSkipRows"
                                :min="0"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <label for="dateColumn">Date Column *</label>
                            <InputText
                                id="dateColumn"
                                v-model="formDateColumn"
                                placeholder="CSV header name, e.g. Date"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
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

                        <div class="field">
                            <label for="descriptionColumn">Description Column *</label>
                            <InputText
                                id="descriptionColumn"
                                v-model="formDescriptionColumn"
                                placeholder="CSV header name, e.g. Description"
                                class="w-full"
                            />
                        </div>

                        <div class="field">
                            <label for="amountColumn">Amount Column *</label>
                            <InputText
                                id="amountColumn"
                                v-model="formAmountColumn"
                                placeholder="CSV header name, e.g. Amount"
                                class="w-full"
                            />
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

:deep(.p-card-content) {
    padding: 1.5rem;
}
</style>
