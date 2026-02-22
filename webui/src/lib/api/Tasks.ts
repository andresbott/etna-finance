import { apiClient } from '@/lib/api/client'

const TASKS_PATH = '/tasks'
const EXECUTIONS_PATH = '/tasks/executions'

export interface TaskDef {
    id: string
    name: string
    description: string
}

export interface TaskSchedule {
    id: number
    task_name: string
    cron_expression: string
    enabled: boolean
    created_at: string
    updated_at: string
}

/** Task with optional schedule (combined API response). */
export interface TaskWithSchedule extends TaskDef {
    schedule?: TaskSchedule | null
}

export interface ExecutionInfo {
    id: string
    task_name: string
    status: string
    queued_at: string
    /** Set only when the task actually ran; omitted for e.g. canceled-before-run so duration is empty. */
    started_at?: string
    ended_at: string
}

export interface ListTasksResponse {
    tasks: TaskWithSchedule[]
}

export interface ListExecutionsResponse {
    executions: ExecutionInfo[]
}

export interface TriggerTaskResponse {
    execution_id: string
}

export interface UpsertTaskBody {
    cron_expression: string
    enabled?: boolean
}

export interface PatchTaskBody {
    cron_expression?: string
    enabled?: boolean
}

export async function listTasks(): Promise<TaskWithSchedule[]> {
    const { data } = await apiClient.get<ListTasksResponse>(TASKS_PATH)
    return data.tasks ?? []
}

export async function getTask(name: string): Promise<TaskWithSchedule> {
    const { data } = await apiClient.get<TaskWithSchedule>(
        `${TASKS_PATH}/${encodeURIComponent(name)}`
    )
    return data
}

export async function listExecutions(): Promise<ExecutionInfo[]> {
    const { data } = await apiClient.get<ListExecutionsResponse>(EXECUTIONS_PATH)
    return data.executions ?? []
}

export async function triggerTask(name: string): Promise<string> {
    const { data } = await apiClient.post<TriggerTaskResponse>(
        `${TASKS_PATH}/${encodeURIComponent(name)}/trigger`
    )
    return data.execution_id
}

export async function cancelExecution(executionId: string): Promise<void> {
    await apiClient.post(
        `${TASKS_PATH}/executions/${encodeURIComponent(executionId)}/cancel`
    )
}

/** Fetches plain-text task log for an execution. Returns empty string if no logs. */
export async function getExecutionLog(executionId: string): Promise<string> {
    const { data } = await apiClient.get<string>(
        `${TASKS_PATH}/executions/${encodeURIComponent(executionId)}/logs`,
        { responseType: 'text' }
    )
    return data ?? ''
}

export async function upsertTask(
    name: string,
    body: UpsertTaskBody
): Promise<TaskWithSchedule> {
    const { data } = await apiClient.put<TaskWithSchedule>(
        `${TASKS_PATH}/${encodeURIComponent(name)}`,
        body
    )
    return data
}

export async function patchTask(
    name: string,
    body: PatchTaskBody
): Promise<TaskWithSchedule> {
    const { data } = await apiClient.patch<TaskWithSchedule>(
        `${TASKS_PATH}/${encodeURIComponent(name)}`,
        body
    )
    return data
}

export async function deleteTaskSchedule(name: string): Promise<void> {
    await apiClient.delete(
        `${TASKS_PATH}/${encodeURIComponent(name)}`
    )
}
