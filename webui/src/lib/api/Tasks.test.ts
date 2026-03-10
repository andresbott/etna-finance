import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    listTasks,
    getTask,
    listExecutions,
    triggerTask,
    cancelExecution,
    getExecutionLog,
    upsertTask,
    patchTask,
    deleteTaskSchedule,
    type TaskWithSchedule,
    type ExecutionInfo,
    type UpsertTaskBody,
    type PatchTaskBody,
} from './Tasks'

vi.mock('./client', () => ({
    apiClient: {
        get: vi.fn(),
        post: vi.fn(),
        put: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
    },
}))

beforeEach(() => vi.clearAllMocks())

const mockTask: TaskWithSchedule = {
    id: 'task-1',
    name: 'daily-sync',
    description: 'Syncs data daily',
    schedule: {
        id: 100,
        task_name: 'daily-sync',
        cron_expression: '0 0 * * *',
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
    },
}

const mockTaskNoSchedule: TaskWithSchedule = {
    id: 'task-2',
    name: 'manual-job',
    description: 'Runs manually',
    schedule: null,
}

const mockExecution: ExecutionInfo = {
    id: 'exec-abc',
    task_name: 'daily-sync',
    status: 'completed',
    queued_at: '2025-06-01T10:00:00Z',
    started_at: '2025-06-01T10:00:01Z',
    ended_at: '2025-06-01T10:05:00Z',
}

// ---------------------------------------------------------------------------
// listTasks
// ---------------------------------------------------------------------------

describe('listTasks', () => {
    it('calls GET /tasks and returns tasks array', async () => {
        const tasks = [mockTask, mockTaskNoSchedule];
        (apiClient.get as Mock).mockResolvedValue({ data: { tasks } })

        const result = await listTasks()

        expect(apiClient.get).toHaveBeenCalledWith('/tasks')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(tasks)
    })

    it('returns empty array when tasks is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await listTasks()

        expect(result).toEqual([])
    })

    it('returns empty array when tasks is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { tasks: null } })

        const result = await listTasks()

        expect(result).toEqual([])
    })
})

// ---------------------------------------------------------------------------
// getTask
// ---------------------------------------------------------------------------

describe('getTask', () => {
    it('calls GET /tasks/:name and returns the task', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockTask })

        const result = await getTask('daily-sync')

        expect(apiClient.get).toHaveBeenCalledWith('/tasks/daily-sync')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockTask)
    })

    it('URL-encodes the task name', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockTask })

        await getTask('my task/special&chars')

        expect(apiClient.get).toHaveBeenCalledWith(
            '/tasks/my%20task%2Fspecial%26chars'
        )
    })
})

// ---------------------------------------------------------------------------
// listExecutions
// ---------------------------------------------------------------------------

describe('listExecutions', () => {
    it('calls GET /tasks/executions and returns executions array', async () => {
        const executions = [mockExecution];
        (apiClient.get as Mock).mockResolvedValue({ data: { executions } })

        const result = await listExecutions()

        expect(apiClient.get).toHaveBeenCalledWith('/tasks/executions')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(executions)
    })

    it('returns empty array when executions is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await listExecutions()

        expect(result).toEqual([])
    })

    it('returns empty array when executions is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { executions: null } })

        const result = await listExecutions()

        expect(result).toEqual([])
    })
})

// ---------------------------------------------------------------------------
// triggerTask
// ---------------------------------------------------------------------------

describe('triggerTask', () => {
    it('calls POST /tasks/:name/trigger and returns the execution_id', async () => {
        (apiClient.post as Mock).mockResolvedValue({
            data: { execution_id: 'exec-new-123' },
        })

        const result = await triggerTask('daily-sync')

        expect(apiClient.post).toHaveBeenCalledWith('/tasks/daily-sync/trigger')
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toBe('exec-new-123')
    })

    it('URL-encodes the task name', async () => {
        (apiClient.post as Mock).mockResolvedValue({
            data: { execution_id: 'exec-x' },
        })

        await triggerTask('spaces in name')

        expect(apiClient.post).toHaveBeenCalledWith(
            '/tasks/spaces%20in%20name/trigger'
        )
    })
})

// ---------------------------------------------------------------------------
// cancelExecution
// ---------------------------------------------------------------------------

describe('cancelExecution', () => {
    it('calls POST /tasks/executions/:id/cancel', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await cancelExecution('exec-abc')

        expect(apiClient.post).toHaveBeenCalledWith(
            '/tasks/executions/exec-abc/cancel'
        )
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await cancelExecution('exec-abc')

        expect(result).toBeUndefined()
    })

    it('URL-encodes the execution id', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await cancelExecution('id/with special')

        expect(apiClient.post).toHaveBeenCalledWith(
            '/tasks/executions/id%2Fwith%20special/cancel'
        )
    })
})

// ---------------------------------------------------------------------------
// getExecutionLog
// ---------------------------------------------------------------------------

describe('getExecutionLog', () => {
    it('calls GET /tasks/executions/:id/logs with responseType text', async () => {
        const logText = 'line1\nline2\nline3';
        (apiClient.get as Mock).mockResolvedValue({ data: logText })

        const result = await getExecutionLog('exec-abc')

        expect(apiClient.get).toHaveBeenCalledWith(
            '/tasks/executions/exec-abc/logs',
            { responseType: 'text' }
        )
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toBe(logText)
    })

    it('returns empty string when data is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: undefined })

        const result = await getExecutionLog('exec-abc')

        expect(result).toBe('')
    })

    it('returns empty string when data is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: null })

        const result = await getExecutionLog('exec-abc')

        expect(result).toBe('')
    })

    it('URL-encodes the execution id', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: '' })

        await getExecutionLog('id/special')

        expect(apiClient.get).toHaveBeenCalledWith(
            '/tasks/executions/id%2Fspecial/logs',
            { responseType: 'text' }
        )
    })
})

// ---------------------------------------------------------------------------
// upsertTask
// ---------------------------------------------------------------------------

describe('upsertTask', () => {
    it('calls PUT /tasks/:name with body and returns the task', async () => {
        const body: UpsertTaskBody = { cron_expression: '0 6 * * *', enabled: true };
        (apiClient.put as Mock).mockResolvedValue({ data: mockTask })

        const result = await upsertTask('daily-sync', body)

        expect(apiClient.put).toHaveBeenCalledWith('/tasks/daily-sync', body)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockTask)
    })

    it('URL-encodes the task name', async () => {
        const body: UpsertTaskBody = { cron_expression: '0 0 * * *' };
        (apiClient.put as Mock).mockResolvedValue({ data: mockTask })

        await upsertTask('my task/name', body)

        expect(apiClient.put).toHaveBeenCalledWith(
            '/tasks/my%20task%2Fname',
            body
        )
    })

    it('sends body without enabled when omitted', async () => {
        const body: UpsertTaskBody = { cron_expression: '*/5 * * * *' };
        (apiClient.put as Mock).mockResolvedValue({ data: mockTask })

        await upsertTask('daily-sync', body)

        expect(apiClient.put).toHaveBeenCalledWith('/tasks/daily-sync', {
            cron_expression: '*/5 * * * *',
        })
    })
})

// ---------------------------------------------------------------------------
// patchTask
// ---------------------------------------------------------------------------

describe('patchTask', () => {
    it('calls PATCH /tasks/:name with body and returns the task', async () => {
        const body: PatchTaskBody = { enabled: false };
        (apiClient.patch as Mock).mockResolvedValue({ data: mockTask })

        const result = await patchTask('daily-sync', body)

        expect(apiClient.patch).toHaveBeenCalledWith('/tasks/daily-sync', body)
        expect(apiClient.patch).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockTask)
    })

    it('URL-encodes the task name', async () => {
        const body: PatchTaskBody = { enabled: true };
        (apiClient.patch as Mock).mockResolvedValue({ data: mockTask })

        await patchTask('name with spaces', body)

        expect(apiClient.patch).toHaveBeenCalledWith(
            '/tasks/name%20with%20spaces',
            body
        )
    })

    it('can send only cron_expression', async () => {
        const body: PatchTaskBody = { cron_expression: '0 12 * * *' };
        (apiClient.patch as Mock).mockResolvedValue({ data: mockTask })

        await patchTask('daily-sync', body)

        expect(apiClient.patch).toHaveBeenCalledWith('/tasks/daily-sync', {
            cron_expression: '0 12 * * *',
        })
    })
})

// ---------------------------------------------------------------------------
// deleteTaskSchedule
// ---------------------------------------------------------------------------

describe('deleteTaskSchedule', () => {
    it('calls DELETE /tasks/:name', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteTaskSchedule('daily-sync')

        expect(apiClient.delete).toHaveBeenCalledWith('/tasks/daily-sync')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteTaskSchedule('daily-sync')

        expect(result).toBeUndefined()
    })

    it('URL-encodes the task name', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteTaskSchedule('task/with spaces')

        expect(apiClient.delete).toHaveBeenCalledWith(
            '/tasks/task%2Fwith%20spaces'
        )
    })
})
