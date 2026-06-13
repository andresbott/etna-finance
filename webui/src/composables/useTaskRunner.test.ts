import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderComposable, createTestQueryClient } from '@/test/helpers'
import type { ExecutionInfo } from '@/lib/api/Tasks'

vi.mock('../lib/api/Tasks', () => ({
    listExecutions: vi.fn(),
    triggerTask: vi.fn(),
}))

import { listExecutions, triggerTask } from '../lib/api/Tasks'
import { useTaskRunner } from './useTaskRunner'

const mockListExecutions = listExecutions as ReturnType<typeof vi.fn>
const mockTriggerTask = triggerTask as ReturnType<typeof vi.fn>

function makeExecution(o: Partial<ExecutionInfo> = {}): ExecutionInfo {
    return {
        id: 'exec-1',
        task_name: 'fx-import',
        status: 'running',
        queued_at: '2026-06-06T10:00:00Z',
        started_at: '2026-06-06T10:00:01Z',
        ended_at: '',
        ...o,
    }
}

describe('useTaskRunner', () => {
    let unmount: () => void
    afterEach(() => {
        unmount?.()
        vi.restoreAllMocks()
    })

    it('run triggers the named task', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([makeExecution({ status: 'running' })])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import'), { queryClient: qc })
        unmount = w.unmount

        await w.result.run()

        expect(mockTriggerTask).toHaveBeenCalledWith('fx-import')
    })

    it('tracks the execution as running', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([makeExecution({ status: 'running' })])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import'), { queryClient: qc })
        unmount = w.unmount

        await w.result.run()

        await vi.waitFor(() => expect(w.result.isRunning.value).toBe(true))
    })

    it('reports running when an active execution of the task type exists without triggering (e.g. after remount)', async () => {
        mockListExecutions.mockResolvedValue([
            makeExecution({ id: 'started-elsewhere', task_name: 'fx-import', status: 'running' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import'), { queryClient: qc })
        unmount = w.unmount

        await vi.waitFor(() => expect(w.result.isRunning.value).toBe(true))
    })

    it('does not report running for an active execution of a different task type', async () => {
        mockListExecutions.mockResolvedValue([
            makeExecution({ id: 'other-task', task_name: 'backup', status: 'running' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import'), { queryClient: qc })
        unmount = w.unmount

        await vi.waitFor(() => expect(mockListExecutions).toHaveBeenCalled())
        expect(w.result.isRunning.value).toBe(false)
    })

    it('records a failed status and stops reporting running when the execution fails', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([
            makeExecution({ status: 'failed', ended_at: '2026-06-06T10:00:05Z' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import'), { queryClient: qc })
        unmount = w.unmount

        await w.result.run()

        await vi.waitFor(() => expect(w.result.lastStatus.value).toBe('failed'))
        expect(w.result.isRunning.value).toBe(false)
    })

    it('calls onComplete with the terminal status when the tracked execution succeeds', async () => {
        const onComplete = vi.fn()
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([
            makeExecution({ status: 'complete', ended_at: '2026-06-06T10:00:05Z' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import', { onComplete }), {
            queryClient: qc,
        })
        unmount = w.unmount

        await w.result.run()

        await vi.waitFor(() => expect(onComplete).toHaveBeenCalledWith('complete'))
    })

    it('calls onComplete with the terminal status when the tracked execution fails', async () => {
        const onComplete = vi.fn()
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([
            makeExecution({ status: 'failed', ended_at: '2026-06-06T10:00:05Z' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useTaskRunner('fx-import', { onComplete }), {
            queryClient: qc,
        })
        unmount = w.unmount

        await w.result.run()

        await vi.waitFor(() => expect(onComplete).toHaveBeenCalledWith('failed'))
    })
})
