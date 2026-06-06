import { ref, computed, watch } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { listExecutions, triggerTask } from '@/lib/api/Tasks'
import type { ExecutionInfo } from '@/lib/api/Tasks'
// Shared keys so TanStack Query dedups the executions cache and the backup file list.
import { EXECUTIONS_QUERY_KEY } from './useTasks'
import { BACKUP_FILES_QUERY_KEY } from './useBackups'

/** Task id of the registered backup task (matches backend BackupTaskName). */
export const BACKUP_TASK_NAME = 'backup'

const POLL_INTERVAL_MS = 500

const ACTIVE_STATUSES = ['waiting', 'running']
const SUCCESS_STATUSES = ['complete']
const FAILURE_STATUSES = ['failed', 'panicked']

function hasActiveExecutions(executions: ExecutionInfo[] | undefined): boolean {
    return !!executions?.some((e) => ACTIVE_STATUSES.includes(e.status))
}

/**
 * Drives the Backup page's "Create Backup" action through the task runner.
 * Triggers the backup task, reuses the executions poll for live progress, and
 * refreshes the backup file list when the triggered execution completes.
 */
export function useBackupTask() {
    const queryClient = useQueryClient()
    const trackedId = ref<string | null>(null)
    const lastBackupStatus = ref<string | null>(null)

    const executionsQuery = useQuery({
        queryKey: EXECUTIONS_QUERY_KEY,
        queryFn: listExecutions,
        refetchInterval: (query) =>
            hasActiveExecutions(query.state.data) ? POLL_INTERVAL_MS : false,
        refetchIntervalInBackground: false,
    })

    const trackedExecution = computed<ExecutionInfo | undefined>(() => {
        const id = trackedId.value
        if (!id) return undefined
        return (executionsQuery.data.value ?? []).find((e) => e.id === id)
    })

    const isBackupRunning = computed(() => {
        const e = trackedExecution.value
        return !!e && ACTIVE_STATUSES.includes(e.status)
    })

    const triggerMutation = useMutation({
        mutationFn: () => triggerTask(BACKUP_TASK_NAME),
        // Reset before every attempt so a prior run's terminal status never lingers.
        onMutate: () => {
            lastBackupStatus.value = null
        },
        onSuccess: (executionId) => {
            trackedId.value = executionId
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
        },
    })

    // When the tracked execution reaches a terminal state, record it and (on
    // success) refresh the file list so the new backup shows up.
    watch(
        () => trackedExecution.value?.status,
        (status) => {
            if (!status) return
            if (SUCCESS_STATUSES.includes(status)) {
                lastBackupStatus.value = status
                queryClient.invalidateQueries({ queryKey: BACKUP_FILES_QUERY_KEY })
            } else if (FAILURE_STATUSES.includes(status)) {
                lastBackupStatus.value = status
            }
        }
    )

    const runBackup = () => triggerMutation.mutateAsync()

    return {
        runBackup,
        isBackupRunning,
        isTriggering: triggerMutation.isPending,
        lastBackupStatus,
        triggerError: triggerMutation.error,
    }
}
