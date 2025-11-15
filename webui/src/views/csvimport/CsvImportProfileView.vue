<script setup>
import { ref, onMounted } from 'vue'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import TopBar from '@/views/topbar.vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import Message from 'primevue/message'
import CsvHeaderEditor from '@/components/CsvHeaderEditor.vue'
import { useToast } from 'primevue/usetoast'

const toast = useToast()

// State
const profiles = ref([])
const isLoading = ref(false)
const showProfileDialog = ref(false)
const editingProfile = ref(null)
const profileName = ref('')
const profileDescription = ref('')
const profileHeaders = ref([])

// Initial sample headers
const getDefaultHeaders = () => [
    { id: 1, name: 'Date', mappedTo: 'date', example: '2025-11-01' },
    { id: 2, name: 'Description', mappedTo: 'description', example: 'Grocery Store' },
    { id: 3, name: 'Amount', mappedTo: 'amount', example: '-45.50' },
    { id: 4, name: 'Category', mappedTo: 'category', example: 'Food' }
]

// Load profiles from storage/API
const loadProfiles = async () => {
    isLoading.value = true
    try {
        // TODO: Implement API call to fetch CSV import profiles
        await new Promise(resolve => setTimeout(resolve, 500)) // Simulated delay
        
        // Simulated data - replace with actual API call
        profiles.value = [
            {
                id: 1,
                name: 'Bank Statement Import',
                description: 'Standard bank CSV format',
                dateFormat: 'DD/MM/YYYY',
                headers: getDefaultHeaders(),
                createdAt: new Date('2025-10-15'),
                updatedAt: new Date('2025-10-20')
            },
            {
                id: 2,
                name: 'Credit Card Import',
                description: 'Credit card transaction format',
                dateFormat: 'YYYY-MM-DD',
                headers: [
                    { id: 1, name: 'Transaction Date', mappedTo: 'date', example: '2025-11-01' },
                    { id: 2, name: 'Merchant', mappedTo: 'description', example: 'Amazon' },
                    { id: 3, name: 'Debit', mappedTo: 'amount', example: '99.99' },
                    { id: 4, name: 'Reference', mappedTo: 'reference', example: 'REF123' }
                ],
                createdAt: new Date('2025-09-10'),
                updatedAt: new Date('2025-10-01')
            }
        ]
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

// Open create dialog
const openCreateDialog = () => {
    editingProfile.value = null
    profileName.value = ''
    profileDescription.value = ''
    profileHeaders.value = getDefaultHeaders()
    showProfileDialog.value = true
}

// Open edit dialog
const openEditDialog = (profile) => {
    editingProfile.value = profile
    profileName.value = profile.name
    profileDescription.value = profile.description
    profileHeaders.value = [...profile.headers]
    showProfileDialog.value = true
}

// Save profile
const handleSaveProfile = async (headerData) => {
    if (!profileName.value.trim()) {
        toast.add({
            severity: 'warn',
            summary: 'Validation Error',
            detail: 'Profile name is required',
            life: 3000
        })
        return
    }

    try {
        // TODO: Implement API call to save profile
        await new Promise(resolve => setTimeout(resolve, 500)) // Simulated delay
        
        if (editingProfile.value) {
            // Update existing profile
            const index = profiles.value.findIndex(p => p.id === editingProfile.value.id)
            if (index !== -1) {
                profiles.value[index] = {
                    ...profiles.value[index],
                    name: profileName.value,
                    description: profileDescription.value,
                    headers: headerData.headers,
                    dateFormat: headerData.dateFormat,
                    updatedAt: new Date()
                }
            }
            toast.add({
                severity: 'success',
                summary: 'Success',
                detail: 'Profile updated successfully',
                life: 3000
            })
        } else {
            // Create new profile
            const newProfile = {
                id: Date.now(),
                name: profileName.value,
                description: profileDescription.value,
                headers: headerData.headers,
                dateFormat: headerData.dateFormat,
                createdAt: new Date(),
                updatedAt: new Date()
            }
            profiles.value.push(newProfile)
            toast.add({
                severity: 'success',
                summary: 'Success',
                detail: 'Profile created successfully',
                life: 3000
            })
        }
        
        showProfileDialog.value = false
    } catch (error) {
        toast.add({
            severity: 'error',
            summary: 'Error',
            detail: 'Failed to save profile: ' + error.message,
            life: 3000
        })
    }
}

// Delete profile
const handleDeleteProfile = async (profile) => {
    if (!confirm(`Are you sure you want to delete the profile "${profile.name}"?`)) {
        return
    }

    try {
        // TODO: Implement API call to delete profile
        await new Promise(resolve => setTimeout(resolve, 500)) // Simulated delay
        
        const index = profiles.value.findIndex(p => p.id === profile.id)
        if (index !== -1) {
            profiles.value.splice(index, 1)
        }
        
        toast.add({
            severity: 'success',
            summary: 'Success',
            detail: 'Profile deleted successfully',
            life: 3000
        })
    } catch (error) {
        toast.add({
            severity: 'error',
            summary: 'Error',
            detail: 'Failed to delete profile: ' + error.message,
            life: 3000
        })
    }
}

// Duplicate profile
const handleDuplicateProfile = (profile) => {
    editingProfile.value = null
    profileName.value = `${profile.name} (Copy)`
    profileDescription.value = profile.description
    profileHeaders.value = [...profile.headers]
    showProfileDialog.value = true
}

// Format date
const formatDate = (date) => {
    return new Date(date).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    })
}

// Get mapped fields count
const getMappedFieldsCount = (headers) => {
    return headers.filter(h => h.mappedTo).length
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
            <div class="csv-profile-container">
                <div class="page-header">
                    <div>
                        <h1 class="page-title">CSV Import Profiles</h1>
                        <p class="page-description">
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
                                    <div class="profile-name">
                                        <i class="pi pi-file-import"></i>
                                        <span>{{ data.name }}</span>
                                    </div>
                                </template>
                            </Column>

                            <Column field="description" header="Description">
                                <template #body="{ data }">
                                    <span class="description">{{ data.description || 'No description' }}</span>
                                </template>
                            </Column>

                            <Column field="dateFormat" header="Date Format" :sortable="true">
                                <template #body="{ data }">
                                    <span class="date-format">{{ data.dateFormat }}</span>
                                </template>
                            </Column>

                            <Column header="Mapped Fields">
                                <template #body="{ data }">
                                    <span class="mapped-count">
                                        {{ getMappedFieldsCount(data.headers) }} / {{ data.headers.length }}
                                    </span>
                                </template>
                            </Column>

                            <Column field="updatedAt" header="Last Updated" :sortable="true">
                                <template #body="{ data }">
                                    {{ formatDate(data.updatedAt) }}
                                </template>
                            </Column>

                            <Column header="Actions" :exportable="false" style="width: 150px">
                                <template #body="{ data }">
                                    <div class="action-buttons">
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            rounded
                                            @click="openEditDialog(data)"
                                            v-tooltip.top="'Edit profile'"
                                        />
                                        <Button
                                            icon="pi pi-copy"
                                            text
                                            rounded
                                            severity="secondary"
                                            @click="handleDuplicateProfile(data)"
                                            v-tooltip.top="'Duplicate profile'"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            severity="danger"
                                            text
                                            rounded
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
                    :style="{ width: '90vw', maxWidth: '1200px' }"
                >
                    <div class="profile-dialog-content">
                        <div class="profile-info">
                            <div class="field">
                                <label for="profileName">Profile Name *</label>
                                <InputText
                                    id="profileName"
                                    v-model="profileName"
                                    placeholder="e.g., Bank Statement Import"
                                    class="w-full"
                                />
                            </div>

                            <div class="field">
                                <label for="profileDescription">Description</label>
                                <InputText
                                    id="profileDescription"
                                    v-model="profileDescription"
                                    placeholder="Brief description of this import profile"
                                    class="w-full"
                                />
                            </div>
                        </div>

                        <CsvHeaderEditor
                            :headers="profileHeaders"
                            @update:headers="profileHeaders = $event"
                            @save="handleSaveProfile"
                        />
                    </div>
                </Dialog>
            </div>
        </template>
    </VerticalLayout>
</template>

<style scoped lang="scss">
.csv-profile-container {
    padding: 2rem;
    max-width: 1400px;
    margin: 0 auto;
}

.page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 2rem;
    gap: 2rem;
}

.page-title {
    font-size: 2rem;
    font-weight: 700;
    margin-bottom: 0.5rem;
    color: var(--text-color);
}

.page-description {
    color: var(--text-color-secondary);
    margin: 0;
    font-size: 1rem;
}

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

.profile-name {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 600;

    i {
        color: var(--primary-color);
    }
}

.description {
    color: var(--text-color-secondary);
}

.date-format {
    font-family: monospace;
    background-color: var(--surface-100);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.9rem;
}

.mapped-count {
    font-weight: 600;
    color: var(--primary-color);
}

.action-buttons {
    display: flex;
    gap: 0.25rem;
    justify-content: center;
}

.profile-dialog-content {
    display: flex;
    flex-direction: column;
    gap: 2rem;
}

.profile-info {
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

