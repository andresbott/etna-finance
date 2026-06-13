import { ref, computed, watch } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { listExecutions, triggerTask } from '@/lib/api/Tasks'
import type { ExecutionInfo } from '@/lib/api/Tasks'
// Shared key so TanStack Query dedups the executions cache with the Tasks page.
import { EXECUTIONS_QUERY_KEY } from './useTasks'

const POLL_INTERVAL_MS = 500

const ACTIVE_STATUSES = ['waiting', 'running']
const TERMINAL_STATUSES = ['complete', 'failed', 'panicked']

function hasActiveExecutions(executions: ExecutionInfo[] | undefined): boolean {
    return !!executions?.some((e) => ACTIVE_STATUSES.includes(e.status))
}

interface TaskRunnerOptions {
    /** Called once when the triggered execution reaches a terminal state, with that status. */
    onComplete?: (status: string) => void
}

/**
 * Drives a one-off "run this background task" action through the task runner.
 * Triggers the named task, reuses the executions poll for live progress, and
 * invokes onComplete when the triggered execution reaches a terminal state so
 * the caller can refresh data and surface feedback.
 *
 * Generalizes the mechanics first proven in useBackupTask.
 */
export function useTaskRunner(taskName: string, options: TaskRunnerOptions = {}) {
    const queryClient = useQueryClient()
    const trackedId = ref<string | null>(null)
    const lastStatus = ref<string | null>(null)

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

    // Reflect any active execution of this task type — not just one we triggered —
    // so the action stays disabled after a remount while a prior run is still going.
    const isRunning = computed(() =>
        (executionsQuery.data.value ?? []).some(
            (e) => e.task_name === taskName && ACTIVE_STATUSES.includes(e.status)
        )
    )

    const triggerMutation = useMutation({
        mutationFn: () => triggerTask(taskName),
        // Reset before every attempt so a prior run's terminal status never lingers.
        onMutate: () => {
            lastStatus.value = null
        },
        onSuccess: (executionId) => {
            trackedId.value = executionId
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
        },
    })

    // When the tracked execution reaches a terminal state, record it and notify the caller.
    watch(
        () => trackedExecution.value?.status,
        (status) => {
            if (!status) return
            if (TERMINAL_STATUSES.includes(status)) {
                lastStatus.value = status
                options.onComplete?.(status)
            }
        }
    )

    const run = () => triggerMutation.mutateAsync()

    return {
        run,
        isRunning,
        isTriggering: triggerMutation.isPending,
        lastStatus,
        triggerError: triggerMutation.error,
    }
}
