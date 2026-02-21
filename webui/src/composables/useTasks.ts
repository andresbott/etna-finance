import { ref, computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import {
    listTasks as listTasksApi,
    listExecutions,
    triggerTask as triggerTaskApi,
    upsertTask as upsertTaskApi,
    patchTask as patchTaskApi,
    deleteTaskSchedule as deleteTaskScheduleApi
} from '@/lib/api/Tasks'
import type {
    TaskWithSchedule,
    ExecutionInfo,
    UpsertTaskBody,
    PatchTaskBody
} from '@/lib/api/Tasks'

export const TASKS_QUERY_KEY = ['tasks'] as const
export const EXECUTIONS_QUERY_KEY = ['tasks', 'executions'] as const

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
    startedAt: string
    finishedAt: string | null
    status: string
    task_name: string
}

function executionToUi(ex: ExecutionInfo): TaskExecution {
    return {
        id: ex.id,
        startedAt: ex.started_at || ex.queued_at,
        finishedAt: ex.ended_at || null,
        status: ex.status,
        task_name: ex.task_name
    }
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
    const triggeringTaskId = ref<string | null>(null)

    const tasksQuery = useQuery({
        queryKey: TASKS_QUERY_KEY,
        queryFn: listTasksApi
    })

    const executionsQuery = useQuery({
        queryKey: EXECUTIONS_QUERY_KEY,
        queryFn: listExecutions
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
        onSettled: () => {
            triggeringTaskId.value = null
            queryClient.invalidateQueries({ queryKey: EXECUTIONS_QUERY_KEY })
        }
    })

    const triggerTask = (task: Task) => {
        triggerMutation.mutate(task.id)
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
        if (
            status === EXECUTION_STATUS.running ||
            status === EXECUTION_STATUS.waiting ||
            status === TASK_STATUS.running
        )
            return 'info'
        if (status === EXECUTION_STATUS.canceled) return 'secondary'
        return 'secondary'
    }

    return {
        tasks,
        executions,
        triggeringTaskId,
        tasksQuery,
        executionsQuery,
        getTaskById,
        triggerTask,
        upsertTask,
        patchTask,
        deleteTaskSchedule,
        upsertTaskMutation,
        patchTaskMutation,
        deleteTaskScheduleMutation,
        getStatusSeverity,
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
                new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime()
        )
    })
}
