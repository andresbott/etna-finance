<script setup>
import { ref, computed } from 'vue'
import { VerticalLayout } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import TopBar from '@/views/topbar.vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import FileUpload from 'primevue/fileupload'
import Message from 'primevue/message'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ConfirmDialog from '@/components/common/confirmDialog.vue'
import { useBackups } from '@/composables/useBackups'

const successMessage = ref('')
const errorMessage = ref('')
const deleteDialogVisible = ref(false)
const backupToDelete = ref(null)

// Use the composable
const {
    backupFiles,
    isLoading,
    createBackup,
    deleteBackup,
    downloadBackup,
    restoreBackup,
    isCreating,
    isDeleting,
    isDownloading,
    isRestoring
} = useBackups()

const handleBackup = async () => {
    successMessage.value = ''
    errorMessage.value = ''
    
    try {
        await createBackup()
        successMessage.value = 'Backup created successfully!'
    } catch (error) {
        errorMessage.value = 'Failed to create backup: ' + error.message
    }
}

const handleRestore = async (event) => {
    successMessage.value = ''
    errorMessage.value = ''
    
    try {
        const file = event.files[0]
        await restoreBackup(file)
        successMessage.value = 'Data restored successfully!'
    } catch (error) {
        errorMessage.value = 'Failed to restore data: ' + error.message
    }
}

const openDeleteDialog = (backup) => {
    backupToDelete.value = backup
    deleteDialogVisible.value = true
}

const handleDeleteBackup = async () => {
    try {
        await deleteBackup(backupToDelete.value.id)
        successMessage.value = `Backup "${backupToDelete.value.filename}" deleted successfully!`
    } catch (error) {
        errorMessage.value = 'Failed to delete backup: ' + error.message
    }
}

const handleDownloadBackup = async (backup) => {
    successMessage.value = ''
    errorMessage.value = ''
    
    try {
        await downloadBackup({ id: backup.id, filename: backup.filename })
        successMessage.value = `Backup "${backup.filename}" downloaded successfully!`
    } catch (error) {
        errorMessage.value = 'Failed to download backup: ' + error.message
    }
}

const formatDate = (filename) => {
    // Extract date from filename if it follows a pattern like "backup_2025-11-01_10-30-00.json"
    // For now, just return the filename as is
    // TODO: Implement proper date extraction or get date from API
    return filename
}

const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <div class="backup-restore-container">
                <h1 class="page-title">Backup & Restore</h1>
                
                <Message v-if="successMessage" severity="success" :closable="true" @close="successMessage = ''">
                    {{ successMessage }}
                </Message>
                
                <Message v-if="errorMessage" severity="error" :closable="true" @close="errorMessage = ''">
                    {{ errorMessage }}
                </Message>

                <Card>
                    <template #content>
                        <!-- Action Buttons -->
                        <div class="backup-actions">
                            <Button
                                label="Create Backup"
                                icon="pi pi-download"
                                @click="handleBackup"
                                :loading="isCreating"
                                :disabled="isRestoring"
                            />
                            <FileUpload
                                mode="basic"
                                accept=".json,.zip"
                                :maxFileSize="10000000"
                                :auto="true"
                                chooseLabel="Upload Backup"
                                chooseIcon="pi pi-upload"
                                @select="handleRestore"
                                :disabled="isCreating"
                            />
                        </div>

                        <!-- Backup Files Table -->
                        <div class="backup-table-container">
                            <DataTable
                                :value="backupFiles"
                                :loading="isLoading"
                                stripedRows
                                :paginator="backupFiles && backupFiles.length > 10"
                                :rows="10"
                                responsiveLayout="scroll"
                            >
                                <template #empty>
                                    <div class="empty-state">
                                        <i class="pi pi-inbox"></i>
                                        <p>No backup files found</p>
                                    </div>
                                </template>
                                
                                <Column field="filename" header="Filename" :sortable="true">
                                    <template #body="{ data }">
                                        <span class="filename">{{ data.filename }}</span>
                                    </template>
                                </Column>
                                
                                <Column field="size" header="Size" :sortable="true" headerStyle="width: 120px">
                                    <template #body="{ data }">
                                        {{ formatFileSize(data.size) }}
                                    </template>
                                </Column>
                                
                                <Column header="Actions" :exportable="false" headerStyle="width: 150px; text-align: center" bodyStyle="text-align: center">
                                    <template #body="{ data }">
                                        <Button
                                            icon="pi pi-download"
                                            severity="info"
                                            text
                                            rounded
                                            :loading="isDownloading"
                                            @click="handleDownloadBackup(data)"
                                            v-tooltip.top="'Download backup'"
                                            class="mr-2"
                                        />
                                        <Button
                                            icon="pi pi-trash"
                                            severity="danger"
                                            text
                                            rounded
                                            :loading="isDeleting"
                                            @click="openDeleteDialog(data)"
                                            v-tooltip.top="'Delete backup'"
                                        />
                                    </template>
                                </Column>
                            </DataTable>
                        </div>
                    </template>
                </Card>
            </div>
        </template>
    </VerticalLayout>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
        v-model:visible="deleteDialogVisible"
        :name="backupToDelete?.filename"
        title="Delete Backup"
        message="Are you sure you want to delete this backup file?"
        :onConfirm="handleDeleteBackup"
    />
</template>

<style scoped lang="scss">
.backup-restore-container {
    padding: 2rem;
    max-width: 1400px;
    margin: 0 auto;
}

.page-title {
    font-size: 2rem;
    font-weight: 700;
    margin-bottom: 1.5rem;
    color: var(--text-color);
}

.card-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    
    i {
        color: var(--primary-color);
    }
}

.card-description {
    margin-bottom: 1.5rem;
    color: var(--text-color-secondary);
    line-height: 1.6;
}

.backup-actions {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
    flex-wrap: wrap;
}

.backup-table-container {
    margin-top: 1rem;
}

.empty-state {
    text-align: center;
    padding: 2rem;
    color: var(--text-color-secondary);
    
    i {
        font-size: 3rem;
        margin-bottom: 1rem;
        opacity: 0.5;
    }
    
    p {
        margin: 0;
        font-size: 1.1rem;
    }
}

.filename {
    font-family: monospace;
    font-size: 0.9rem;
}

:deep(.p-card) {
    height: 100%;
}

:deep(.p-card-content) {
    padding-top: 0;
}
</style>

