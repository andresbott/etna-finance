import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { ref } from 'vue'
import { renderComposable, createTestQueryClient } from '@/test/helpers'
import type { TaskWithSchedule, ExecutionInfo } from '@/lib/api/Tasks'

vi.mock('../lib/api/Tasks', () => ({
    listTasks: vi.fn(),
    listExecutions: vi.fn(),
    triggerTask: vi.fn(),
    cancelExecution: vi.fn(),
    getExecutionLog: vi.fn(),
    upsertTask: vi.fn(),
    patchTask: vi.fn(),
    deleteTaskSchedule: vi.fn(),
}))

vi.mock('primevue/usetoast', () => ({
    useToast: () => ({ add: vi.fn() }),
}))

import { listTasks, listExecutions } from '../lib/api/Tasks'
import { useTasks, useTaskExecutions, EXECUTION_STATUS, TASK_STATUS } from './useTasks'
import type { Task, TaskExecution } from './useTasks'

const mockListTasks = listTasks as ReturnType<typeof vi.fn>
const mockListExecutions = listExecutions as ReturnType<typeof vi.fn>

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeTask(overrides: Partial<TaskWithSchedule> = {}): TaskWithSchedule {
    return {
        id: 'sync-accounts',
        name: 'Sync Accounts',
        description: 'Synchronize external accounts',
        schedule: null,
        ...overrides,
    }
}

function makeExecution(overrides: Partial<ExecutionInfo> = {}): ExecutionInfo {
    return {
        id: 'exec-1',
        task_name: 'sync-accounts',
        status: 'complete',
        queued_at: '2025-06-01T10:00:00Z',
        started_at: '2025-06-01T10:00:01Z',
        ended_at: '2025-06-01T10:00:05Z',
        ...overrides,
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('useTasks', () => {
    let unmount: () => void

    afterEach(() => {
        unmount?.()
        vi.restoreAllMocks()
    })

    function setup(tasks: TaskWithSchedule[] = [], executions: ExecutionInfo[] = []) {
        mockListTasks.mockResolvedValue(tasks)
        mockListExecutions.mockResolvedValue(executions)

        const qc = createTestQueryClient()
        const wrapper = renderComposable(() => useTasks(), { queryClient: qc })
        unmount = wrapper.unmount
        return wrapper.result
    }

    // -- derive tasks with last execution info --

    describe('derives tasks with last execution info', () => {
        it('returns empty array when API returns no tasks', async () => {
            const result = setup([], [])
            // Wait for queries to settle
            await vi.waitFor(() => {
                expect(result.tasks.value).toEqual([])
            })
        })

        it('attaches null lastExecution when task has no executions', async () => {
            const task = makeTask({ id: 'my-task' })
            const result = setup([task], [])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            const derived = result.tasks.value[0]
            expect(derived.lastExecution).toBeNull()
            expect(derived.lastExecutionStatus).toBeNull()
        })

        it('picks the most recent execution by queued_at', async () => {
            const task = makeTask({ id: 'my-task' })
            const older = makeExecution({
                id: 'exec-old',
                task_name: 'my-task',
                status: 'complete',
                queued_at: '2025-01-01T00:00:00Z',
                started_at: '2025-01-01T00:00:01Z',
            })
            const newer = makeExecution({
                id: 'exec-new',
                task_name: 'my-task',
                status: 'failed',
                queued_at: '2025-06-01T00:00:00Z',
                started_at: '2025-06-01T00:00:02Z',
            })
            const result = setup([task], [older, newer])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            const derived = result.tasks.value[0]
            expect(derived.lastExecution).toBe('2025-06-01T00:00:02Z')
            expect(derived.lastExecutionStatus).toBe('failed')
        })

        it('uses queued_at as lastExecution when started_at is absent', async () => {
            const task = makeTask({ id: 'my-task' })
            const exec = makeExecution({
                task_name: 'my-task',
                queued_at: '2025-03-01T12:00:00Z',
                started_at: undefined,
                status: 'canceled',
            })
            const result = setup([task], [exec])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            expect(result.tasks.value[0].lastExecution).toBe('2025-03-01T12:00:00Z')
        })

        it('preserves original task fields in derived task', async () => {
            const schedule = {
                id: 1,
                task_name: 'my-task',
                cron_expression: '0 * * * *',
                enabled: true,
                created_at: '2025-01-01T00:00:00Z',
                updated_at: '2025-01-01T00:00:00Z',
            }
            const task = makeTask({ id: 'my-task', name: 'My Task', description: 'desc', schedule })
            const result = setup([task], [])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            const derived = result.tasks.value[0]
            expect(derived.id).toBe('my-task')
            expect(derived.name).toBe('My Task')
            expect(derived.description).toBe('desc')
            expect(derived.schedule).toEqual(schedule)
        })

        it('maps executions to correct tasks when multiple tasks exist', async () => {
            const taskA = makeTask({ id: 'task-a', name: 'Task A' })
            const taskB = makeTask({ id: 'task-b', name: 'Task B' })
            const execA = makeExecution({
                id: 'exec-a',
                task_name: 'task-a',
                status: 'complete',
                queued_at: '2025-06-01T00:00:00Z',
            })
            const execB = makeExecution({
                id: 'exec-b',
                task_name: 'task-b',
                status: 'running',
                queued_at: '2025-06-02T00:00:00Z',
            })
            const result = setup([taskA, taskB], [execA, execB])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(2)
            })

            const derivedA = result.tasks.value.find((t) => t.id === 'task-a')!
            const derivedB = result.tasks.value.find((t) => t.id === 'task-b')!
            expect(derivedA.lastExecutionStatus).toBe('complete')
            expect(derivedB.lastExecutionStatus).toBe('running')
        })
    })

    // -- executions computed --

    describe('executions computed', () => {
        it('maps API executions to UI shape', async () => {
            const exec = makeExecution({
                id: 'e1',
                task_name: 'task-x',
                status: 'complete',
                queued_at: '2025-06-01T10:00:00Z',
                started_at: '2025-06-01T10:00:01Z',
                ended_at: '2025-06-01T10:00:05Z',
            })
            const result = setup([], [exec])

            await vi.waitFor(() => {
                expect(result.executions.value).toHaveLength(1)
            })

            const uiExec = result.executions.value[0]
            expect(uiExec.id).toBe('e1')
            expect(uiExec.queuedAt).toBe('2025-06-01T10:00:00Z')
            expect(uiExec.executionStartedAt).toBe('2025-06-01T10:00:01Z')
            expect(uiExec.finishedAt).toBe('2025-06-01T10:00:05Z')
            expect(uiExec.status).toBe('complete')
            expect(uiExec.task_name).toBe('task-x')
        })

        it('sets executionStartedAt to null when started_at is missing', async () => {
            const exec = makeExecution({ started_at: undefined })
            const result = setup([], [exec])

            await vi.waitFor(() => {
                expect(result.executions.value).toHaveLength(1)
            })

            expect(result.executions.value[0].executionStartedAt).toBeNull()
        })

        it('returns empty array when no executions', async () => {
            const result = setup([], [])

            await vi.waitFor(() => {
                expect(result.executions.value).toEqual([])
            })
        })
    })

    // -- getStatusSeverity --

    describe('getStatusSeverity', () => {
        it('returns "success" for complete', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.complete)).toBe('success')
        })

        it('returns "success" for task status success', () => {
            const result = setup()
            expect(result.getStatusSeverity(TASK_STATUS.success)).toBe('success')
        })

        it('returns "danger" for failed', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.failed)).toBe('danger')
        })

        it('returns "danger" for panicked', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.panicked)).toBe('danger')
        })

        it('returns "danger" for cancel_error', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.cancel_error)).toBe('danger')
        })

        it('returns "danger" for task status error', () => {
            const result = setup()
            expect(result.getStatusSeverity(TASK_STATUS.error)).toBe('danger')
        })

        it('returns "warn" for waiting', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.waiting)).toBe('warn')
        })

        it('returns "info" for execution running', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.running)).toBe('info')
        })

        it('returns "info" for task status running', () => {
            const result = setup()
            expect(result.getStatusSeverity(TASK_STATUS.running)).toBe('info')
        })

        it('returns "contrast" for canceled', () => {
            const result = setup()
            expect(result.getStatusSeverity(EXECUTION_STATUS.canceled)).toBe('contrast')
        })

        it('returns "secondary" for unknown status', () => {
            const result = setup()
            expect(result.getStatusSeverity('unknown-status')).toBe('secondary')
        })

        it('returns "secondary" for idle task status', () => {
            const result = setup()
            expect(result.getStatusSeverity(TASK_STATUS.idle)).toBe('secondary')
        })
    })

    // -- getStatusLabel --

    describe('getStatusLabel', () => {
        it('returns "queued" for waiting status', () => {
            const result = setup()
            expect(result.getStatusLabel(EXECUTION_STATUS.waiting)).toBe('queued')
        })

        it('returns the status as-is for non-waiting statuses', () => {
            const result = setup()
            expect(result.getStatusLabel(EXECUTION_STATUS.running)).toBe('running')
            expect(result.getStatusLabel(EXECUTION_STATUS.complete)).toBe('complete')
            expect(result.getStatusLabel(EXECUTION_STATUS.failed)).toBe('failed')
            expect(result.getStatusLabel(EXECUTION_STATUS.panicked)).toBe('panicked')
            expect(result.getStatusLabel(EXECUTION_STATUS.canceled)).toBe('canceled')
            expect(result.getStatusLabel(EXECUTION_STATUS.cancel_error)).toBe('cancel_error')
        })

        it('returns unknown strings as-is', () => {
            const result = setup()
            expect(result.getStatusLabel('some-custom-status')).toBe('some-custom-status')
        })
    })

    // -- getTaskById --

    describe('getTaskById', () => {
        it('returns undefined when no tasks loaded', async () => {
            const result = setup([], [])

            await vi.waitFor(() => {
                expect(result.getTaskById('nonexistent')).toBeUndefined()
            })
        })

        it('finds a task by id', async () => {
            const task = makeTask({ id: 'find-me', name: 'Find Me' })
            const result = setup([task], [])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            const found = result.getTaskById('find-me')
            expect(found).toBeDefined()
            expect(found!.name).toBe('Find Me')
        })

        it('returns undefined for non-matching id', async () => {
            const task = makeTask({ id: 'exists' })
            const result = setup([task], [])

            await vi.waitFor(() => {
                expect(result.tasks.value).toHaveLength(1)
            })

            expect(result.getTaskById('nope')).toBeUndefined()
        })
    })

    // -- empty state --

    describe('empty state', () => {
        it('tasks starts empty before queries resolve', () => {
            mockListTasks.mockReturnValue(new Promise(() => {})) // never resolves
            mockListExecutions.mockReturnValue(new Promise(() => {}))

            const qc = createTestQueryClient()
            const wrapper = renderComposable(() => useTasks(), { queryClient: qc })
            unmount = wrapper.unmount

            expect(wrapper.result.tasks.value).toEqual([])
            expect(wrapper.result.executions.value).toEqual([])
        })

        it('triggeringTaskId starts as null', () => {
            mockListTasks.mockReturnValue(new Promise(() => {}))
            mockListExecutions.mockReturnValue(new Promise(() => {}))

            const qc = createTestQueryClient()
            const wrapper = renderComposable(() => useTasks(), { queryClient: qc })
            unmount = wrapper.unmount

            expect(wrapper.result.triggeringTaskId.value).toBeNull()
        })
    })
})

// ---------------------------------------------------------------------------
// useTaskExecutions
// ---------------------------------------------------------------------------

describe('useTaskExecutions', () => {
    let unmount: () => void

    afterEach(() => {
        unmount?.()
    })

    function makeUiExecution(overrides: Partial<TaskExecution> = {}): TaskExecution {
        return {
            id: 'exec-1',
            queuedAt: '2025-06-01T10:00:00Z',
            executionStartedAt: '2025-06-01T10:00:01Z',
            finishedAt: '2025-06-01T10:00:05Z',
            status: 'complete',
            task_name: 'my-task',
            ...overrides,
        }
    }

    it('filters executions by task id', () => {
        const executions = ref<TaskExecution[]>([
            makeUiExecution({ id: 'e1', task_name: 'task-a' }),
            makeUiExecution({ id: 'e2', task_name: 'task-b' }),
            makeUiExecution({ id: 'e3', task_name: 'task-a' }),
        ])
        const taskId = ref<string | undefined>('task-a')

        const wrapper = renderComposable(() => useTaskExecutions(taskId, executions))
        unmount = wrapper.unmount

        expect(wrapper.result.value).toHaveLength(2)
        expect(wrapper.result.value.every((e) => e.task_name === 'task-a')).toBe(true)
    })

    it('returns empty array when taskId is undefined', () => {
        const executions = ref<TaskExecution[]>([
            makeUiExecution({ id: 'e1', task_name: 'task-a' }),
        ])
        const taskId = ref<string | undefined>(undefined)

        const wrapper = renderComposable(() => useTaskExecutions(taskId, executions))
        unmount = wrapper.unmount

        expect(wrapper.result.value).toEqual([])
    })

    it('returns empty array when executions list is empty', () => {
        const executions = ref<TaskExecution[]>([])
        const taskId = ref<string | undefined>('task-a')

        const wrapper = renderComposable(() => useTaskExecutions(taskId, executions))
        unmount = wrapper.unmount

        expect(wrapper.result.value).toEqual([])
    })

    it('sorts filtered executions by queuedAt descending', () => {
        const executions = ref<TaskExecution[]>([
            makeUiExecution({ id: 'e-old', task_name: 'task-a', queuedAt: '2025-01-01T00:00:00Z' }),
            makeUiExecution({ id: 'e-new', task_name: 'task-a', queuedAt: '2025-06-01T00:00:00Z' }),
            makeUiExecution({ id: 'e-mid', task_name: 'task-a', queuedAt: '2025-03-01T00:00:00Z' }),
        ])
        const taskId = ref<string | undefined>('task-a')

        const wrapper = renderComposable(() => useTaskExecutions(taskId, executions))
        unmount = wrapper.unmount

        const ids = wrapper.result.value.map((e) => e.id)
        expect(ids).toEqual(['e-new', 'e-mid', 'e-old'])
    })

    it('accepts a getter function for taskId', () => {
        const executions = ref<TaskExecution[]>([
            makeUiExecution({ id: 'e1', task_name: 'task-a' }),
            makeUiExecution({ id: 'e2', task_name: 'task-b' }),
        ])

        const wrapper = renderComposable(() => useTaskExecutions(() => 'task-b', executions))
        unmount = wrapper.unmount

        expect(wrapper.result.value).toHaveLength(1)
        expect(wrapper.result.value[0].id).toBe('e2')
    })
})
