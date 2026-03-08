import { ref, computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useToast } from 'primevue/usetoast'
import {
    listTasks as listTasksApi,
    listExecutions,
    triggerTask as triggerTaskApi,
    cancelExecution as cancelExecutionApi,
    getExecutionLog as getExecutionLogApi,
    upsertTask as upsertTaskApi,
    patchTask as patchTaskApi,
    deleteTaskSchedule as deleteTaskScheduleApi
} from '@/lib/api/Tasks'
import { getApiErrorMessage } from '@/utils/apiError'
import type {
    TaskWithSchedule,
    ExecutionInfo,
    UpsertTaskBody,
    PatchTaskBody
} from '@/lib/api/Tasks'

export const TASKS_QUERY_KEY = ['tasks'] as const
export const EXECUTIONS_QUERY_KEY = ['tasks', 'executions'] as const

/**
 * Polling: we poll the executions list only while any execution is waiting or running.
 * - refetchInterval: 500ms when hasActiveExecutions(data), false otherwise (no polling when all terminal).
 * - refetchIntervalInBackground: false so we don't poll when the tab is hidden.
 * Trigger/cancel mutations still invalidate the query so the first update after user action is immediate.
 */
const EXECUTIONS_POLL_INTERVAL_MS = 500

export const TASK_STATUS = {
    idle: 'idle',
    running: 'running',
    success: 'success',
    error: 'error'
} as const

export const EXECUTION_STATUS = {
    waiting: 'waiting',
    running: 'running',
    complete: 'complete',
    failed: 'failed',
    panicked: 'panicked',
    canceled: 'canceled',
    cancel_error: 'cancel_error'
} as const

/** Task with schedule and derived last execution for list view */
export interface Task extends TaskWithSchedule {
    lastExecution: string | null
    lastExecutionStatus: string | null
}

/** Execution for UI (API uses snake_case; we keep id as string) */
export interface TaskExecution {
    id: string
    /** When the task was queued (for "Queued at" column) */
    queuedAt: string
    /** When the task actually started running; null if it never ran (duration is then empty) */
    executionStartedAt: string | null
    finishedAt: string | null
    status: string
    task_name: string
}

function executionToUi(ex: ExecutionInfo): TaskExecution {
    return {
        id: ex.id,
        queuedAt: ex.queued_at,
        executionStartedAt: ex.started_at ?? null,
        finishedAt: ex.ended_at || null,
        status: ex.status,
        task_name: ex.task_name
    }
}

/** True if any execution is waiting or running (non-terminal). */
function hasActiveExecutions(executions: ExecutionInfo[] | undefined): boolean {
    if (!executions?.length) return false
    return executions.some(
        (e) => e.status === EXECUTION_STATUS.waiting || e.status === EXECUTION_STATUS.running
    )
}

function deriveTasksWithLastExecution(
    tasks: TaskWithSchedule[],
    executions: ExecutionInfo[]
): Task[] {
    const byTask = new Map<string, ExecutionInfo[]>()
    for (const e of executions) {
        const list = byTask.get(e.task_name) ?? []
        list.push(e)
        byTask.set(e.task_name, list)
    }
    return tasks.map((t) => {
        const list = (byTask.get(t.id) ?? []).sort(
            (a, b) =>
                new Date(b.queued_at).getTime() - new Date(a.queued_at).getTime()
        )
        const last = list[0]
        return {
            ...t,
            lastExecution: last ? (last.started_at || last.queued_at) : null,
            lastExecutionStatus: last?.status ?? null
        }
    })
}


export function useTasks() {
    const queryClient = useQueryClient()
    const toast = useToast()
    const triggeringTaskId = ref<string | null>(null)

    const tasksQuery = useQuery({
        queryKey: TASKS_QUERY_KEY,
        queryFn: listTasksApi
    })

    const executionsQuery = useQuery({
        queryKey: EXECUTIONS_QUERY_KEY,
        queryFn: listExecutions,
        refetchInterval: (query) =>
            hasActiveExecutions(query.state.data) ? EXECUTIONS_POLL_INTERVAL_MS : false,
        refetchIntervalInBackground: false
    })

    const tasks = computed<Task[]>(() => {
        const list = tasksQuery.data.value ?? []
        const execs = executionsQuery.data.value ?? []
        return deriveTasksWithLastExecution(list, execs)
    })

    const executions = computed<TaskExecution[]>(() => {
        const list = executionsQuery.data.value ?? []
        return list.map(executionToUi)
    })

    const triggerMutation = useMutation({
        mutationFn: (name: string) => triggerTaskApi(name),
        onMutate: (name) => {
            triggeringTaskId.value = name
        },
        onError: (error) => {
            toast.add({
                severity: 'error',
                summary: 'Task trigger failed',
                detail: getApiErrorMessage(error),
                life: 5000
            })
        },
        onSettled: () => {
            triggeringTaskId.value = null
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
        }
    })

    const cancelMutation = useMutation({
        mutationFn: (executionId: string) => cancelExecutionApi(executionId),
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
        }
    })

    const triggerTask = (task: Task) => {
        triggerMutation.mutate(task.id)
    }

    const cancelTaskExecution = (executionId: string) => {
        cancelMutation.mutate(executionId)
    }

    const upsertTaskMutation = useMutation({
        mutationFn: ({ name, body }: { name: string; body: UpsertTaskBody }) =>
            upsertTaskApi(name, body),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: TASKS_QUERY_KEY })
        }
    })

    const patchTaskMutation = useMutation({
        mutationFn: ({ name, body }: { name: string; body: PatchTaskBody }) =>
            patchTaskApi(name, body),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: TASKS_QUERY_KEY })
        }
    })

    const deleteTaskScheduleMutation = useMutation({
        mutationFn: (name: string) => deleteTaskScheduleApi(name),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: TASKS_QUERY_KEY })
        }
    })

    const upsertTask = (name: string, body: UpsertTaskBody) =>
        upsertTaskMutation.mutateAsync({ name, body })
    const patchTask = (name: string, body: PatchTaskBody) =>
        patchTaskMutation.mutateAsync({ name, body })
    const deleteTaskSchedule = (name: string) =>
        deleteTaskScheduleMutation.mutateAsync(name)

    const getTaskById = (id: string): Task | undefined => {
        return tasks.value.find((t) => t.id === id)
    }

    const getExecutionLog = (executionId: string): Promise<string> => {
        return getExecutionLogApi(executionId)
    }

    const getStatusSeverity = (status: string): string => {
        if (
            status === EXECUTION_STATUS.complete ||
            status === TASK_STATUS.success
        )
            return 'success'
        if (
            status === EXECUTION_STATUS.failed ||
            status === EXECUTION_STATUS.panicked ||
            status === EXECUTION_STATUS.cancel_error ||
            status === TASK_STATUS.error
        )
            return 'danger'
        if (status === EXECUTION_STATUS.waiting) return 'warn'
        if (
            status === EXECUTION_STATUS.running ||
            status === TASK_STATUS.running
        )
            return 'info'
        if (status === EXECUTION_STATUS.canceled) return 'contrast'
        return 'secondary'
    }

    const getStatusLabel = (status: string): string => {
        if (status === EXECUTION_STATUS.waiting) return 'queued'
        return status
    }

    return {
        tasks,
        executions,
        triggeringTaskId,
        tasksQuery,
        executionsQuery,
        getTaskById,
        triggerTask,
        cancelTaskExecution,
        cancelMutation,
        upsertTask,
        patchTask,
        deleteTaskSchedule,
        upsertTaskMutation,
        patchTaskMutation,
        deleteTaskScheduleMutation,
        getStatusSeverity,
        getStatusLabel,
        getExecutionLog,
        refetchTasks: () => queryClient.invalidateQueries({ queryKey: TASKS_QUERY_KEY }),
        refetchExecutions: () =>
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
    }
}

/** Filter executions by task id (task_name). Use in the same component that calls useTasks(). */
export function useTaskExecutions(
    taskId: MaybeRefOrGetter<string | undefined>,
    executions: MaybeRefOrGetter<TaskExecution[]>
) {
    return computed(() => {
        const id = typeof taskId === 'function' ? taskId() : unref(taskId)
        const list = typeof executions === 'function' ? executions() : unref(executions)
        if (!id || !list) return []
        const filtered = list.filter((e) => e.task_name === id)
        return [...filtered].sort(
            (a, b) =>
                new Date(b.queuedAt).getTime() - new Date(a.queuedAt).getTime()
        )
    })
}
