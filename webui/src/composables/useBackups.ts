import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { GetBackupFiles, DeleteBackupFile, CreateBackup, RestoreBackup, DownloadBackupFile } from '@/lib/api/Backup'

export function useBackups() {
    const queryClient = useQueryClient()
    const QUERY_KEY = ['backupFiles']

    // Query to get list of backup files
    const backupFilesQuery = useQuery({
        queryKey: QUERY_KEY,
        queryFn: GetBackupFiles
    })

    // Mutation to create a new backup
    const createBackupMutation = useMutation({
        mutationFn: CreateBackup,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation to delete a backup file
    const deleteBackupMutation = useMutation({
        mutationFn: (id: string) => DeleteBackupFile(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation to download a backup file
    const downloadBackupMutation = useMutation({
        mutationFn: ({ id, filename }: { id: string; filename: string }) => 
            DownloadBackupFile(id, filename)
    })

    // Mutation to restore from a backup file
    const restoreBackupMutation = useMutation({
        mutationFn: (file: File) => RestoreBackup(file),
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
        createBackup: createBackupMutation.mutateAsync,
        deleteBackup: deleteBackupMutation.mutateAsync,
        downloadBackup: downloadBackupMutation.mutateAsync,
        restoreBackup: restoreBackupMutation.mutateAsync,

        // Mutation states
        isCreating: createBackupMutation.isPending,
        isDeleting: deleteBackupMutation.isPending,
        isDownloading: downloadBackupMutation.isPending,
        isRestoring: restoreBackupMutation.isPending
    }
}

