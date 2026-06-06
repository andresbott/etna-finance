import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { getBackupFiles, deleteBackupFile, restoreBackup, restoreBackupFromExisting, downloadBackupFile } from '@/lib/api/Backup'

/** Shared so other modules (e.g. useBackupTask) can invalidate the backup file list. */
export const BACKUP_FILES_QUERY_KEY = ['backupFiles'] as const

export function useBackups() {
    const queryClient = useQueryClient()
    const QUERY_KEY = BACKUP_FILES_QUERY_KEY

    // Query to get list of backup files
    const backupFilesQuery = useQuery({
        queryKey: QUERY_KEY,
        queryFn: getBackupFiles
    })

    // Mutation to delete a backup file
    const deleteBackupMutation = useMutation({
        mutationFn: (id: string) => deleteBackupFile(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation to download a backup file
    const downloadBackupMutation = useMutation({
        mutationFn: ({ id, filename }: { id: string; filename: string }) => 
            downloadBackupFile(id, filename)
    })

    // Mutation to restore from a backup file
    const restoreBackupMutation = useMutation({
        mutationFn: (file: File) => restoreBackup(file),
        onSuccess: () => {
            // Optionally invalidate all queries since data is being restored
            queryClient.invalidateQueries()
        }
    })

    // Mutation to restore from an existing backup by ID
    const restoreBackupFromExistingMutation = useMutation({
        mutationFn: (id: string) => restoreBackupFromExisting(id),
        onSuccess: () => {
            // Optionally invalidate all queries since data is being restored
            queryClient.invalidateQueries()
        }
    })

    return {
        // Query data
        backupFiles: backupFilesQuery.data,
        isLoading: backupFilesQuery.isLoading,
        isError: backupFilesQuery.isError,
        error: backupFilesQuery.error,
        refetch: backupFilesQuery.refetch,

        // Mutations
        deleteBackup: deleteBackupMutation.mutateAsync,
        downloadBackup: downloadBackupMutation.mutateAsync,
        restoreBackup: restoreBackupMutation.mutateAsync,
        restoreBackupFromExisting: restoreBackupFromExistingMutation.mutateAsync,

        // Mutation states
        isDeleting: deleteBackupMutation.isPending,
        isDownloading: downloadBackupMutation.isPending,
        isRestoring: restoreBackupMutation.isPending || restoreBackupFromExistingMutation.isPending
    }
}

