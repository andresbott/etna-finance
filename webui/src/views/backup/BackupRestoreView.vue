<script setup>
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Message from 'primevue/message'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import FileInput from '@/components/common/FileInput.vue'
import { useBackups } from '@/composables/useBackups'

const successMessage = ref('')
const errorMessage = ref('')
const deleteDialogVisible = ref(false)
const backupToDelete = ref(null)
const restoreDialogVisible = ref(false)
const backupToRestore = ref(null)
const uploadRestoreDialogVisible = ref(false)
const selectedFile = ref(null)

// Use the composable
const {
    backupFiles,
    isLoading,
    createBackup,
    deleteBackup,
    downloadBackup,
    restoreBackup,
    restoreBackupFromExisting,
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

const openUploadRestoreDialog = () => {
    selectedFile.value = null
    uploadRestoreDialogVisible.value = true
}

const handleUploadRestore = async () => {
    successMessage.value = ''
    errorMessage.value = ''

    try {
        await restoreBackup(selectedFile.value)
        uploadRestoreDialogVisible.value = false
        successMessage.value = 'Data restored successfully from uploaded backup!'
    } catch (error) {
        errorMessage.value = 'Failed to restore data: ' + error.message
    } finally {
        selectedFile.value = null
    }
}

const openDeleteDialog = (backup) => {
    backupToDelete.value = backup
    deleteDialogVisible.value = true
}

const handleDeleteBackup = async () => {
    try {
        await deleteBackup(backupToDelete.value.id)
        deleteDialogVisible.value = false
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

const openRestoreDialog = (backup) => {
    backupToRestore.value = backup
    restoreDialogVisible.value = true
}

const handleRestoreBackup = async () => {
    successMessage.value = ''
    errorMessage.value = ''
    
    try {
        await restoreBackupFromExisting(backupToRestore.value.id)
        restoreDialogVisible.value = false
        successMessage.value = `Data restored successfully from "${backupToRestore.value.filename}"!`
    } catch (error) {
        errorMessage.value = 'Failed to restore backup: ' + error.message
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
    <div>
        <div class="mb-4">
            <h1 class="text-2xl font-bold mb-2 text-color">Backup & Restore</h1>
            <p class="text-color-secondary m-0 mb-3 text-base">
                Create and manage backups of your application data
            </p>
            <div class="flex justify-content-end gap-2">
                <Button
                    label="Create Backup"
                    icon="ti ti-download"
                    @click="handleBackup"
                    :loading="isCreating"
                    :disabled="isRestoring"
                />
                <Button
                    label="Upload & Restore"
                    icon="ti ti-upload"
                    severity="secondary"
                    @click="openUploadRestoreDialog"
                    :disabled="isCreating || isRestoring"
                />
            </div>
        </div>

        <Message v-if="successMessage" severity="success" :closable="true" @close="successMessage = ''" class="mb-3">
            {{ successMessage }}
        </Message>

        <Message v-if="errorMessage" severity="error" :closable="true" @close="errorMessage = ''" class="mb-3">
            {{ errorMessage }}
        </Message>

        <Card>
            <template #content>
                <!-- Backup Files Table -->
                <div>
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
                                        <i class="ti ti-inbox"></i>
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
                                
                                <Column header="Actions" :exportable="false" headerStyle="width: 200px; text-align: center" bodyStyle="text-align: center">
                                    <template #body="{ data }">
                                        <Button
                                            icon="ti ti-download"
                                            severity="info"
                                            text
                                            rounded
                                            :loading="isDownloading"
                                            @click="handleDownloadBackup(data)"
                                            v-tooltip.top="'Download backup'"
                                            class="mr-2"
                                        />
                                        <Button
                                            icon="ti ti-rotate"
                                            severity="success"
                                            text
                                            rounded
                                            :loading="isRestoring"
                                            @click="openRestoreDialog(data)"
                                            v-tooltip.top="'Restore backup'"
                                            class="mr-2"
                                        />
                                        <Button
                                            icon="ti ti-trash"
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

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
        v-model:visible="deleteDialogVisible"
        :name="backupToDelete?.filename"
        title="Delete Backup"
        message="Are you sure you want to delete this backup file?"
        @confirm="handleDeleteBackup"
    />

    <!-- Restore from existing backup Confirmation Dialog -->
    <ConfirmDialog
        v-model:visible="restoreDialogVisible"
        :name="backupToRestore?.filename"
        title="Restore Backup"
        message="Are you sure you want to restore this backup? This will replace all current data."
        @confirm="handleRestoreBackup"
    />

    <!-- Upload & Restore Dialog -->
    <Dialog
        v-model:visible="uploadRestoreDialogVisible"
        modal
        :closable="true"
        :draggable="false"
        header="Restore from file"
        class="entry-dialog"
    >
        <Message severity="warn" :closable="false" class="mb-3" :pt="{ transition: { name: 'none' } }">
            <strong>Warning:</strong> Restoring a backup will permanently wipe all current data and replace it with the contents of the uploaded file. This action cannot be undone.
        </Message>

        <div class="mb-4">
            <label class="block mb-2">Select a backup file:</label>
            <FileInput v-model="selectedFile" accept=".json,.zip" label="Choose backup file" />
        </div>

        <div class="flex justify-content-end gap-3">
            <Button
                type="button"
                label="Restore"
                icon="pi pi-upload"
                severity="danger"
                :loading="isRestoring"
                :disabled="!selectedFile"
                @click="handleUploadRestore"
            />
            <Button
                type="button"
                label="Cancel"
                icon="pi pi-times"
                severity="secondary"
                :disabled="isRestoring"
                @click="uploadRestoreDialogVisible = false"
            />
        </div>
    </Dialog>
</template>

<style scoped lang="scss">
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

:deep(.p-card-content) {
    padding-top: 0;
}
</style>

