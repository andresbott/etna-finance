import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderComposable, createTestQueryClient } from '@/test/helpers'
import type { ExecutionInfo } from '@/lib/api/Tasks'

vi.mock('../lib/api/Tasks', () => ({
    listExecutions: vi.fn(),
    triggerTask: vi.fn(),
}))

vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

import { listExecutions, triggerTask } from '../lib/api/Tasks'
import { useBackupTask } from './useBackupTask'

const mockListExecutions = listExecutions as ReturnType<typeof vi.fn>
const mockTriggerTask = triggerTask as ReturnType<typeof vi.fn>

function makeExecution(o: Partial<ExecutionInfo> = {}): ExecutionInfo {
    return {
        id: 'exec-1',
        task_name: 'backup',
        status: 'running',
        queued_at: '2026-06-06T10:00:00Z',
        started_at: '2026-06-06T10:00:01Z',
        ended_at: '',
        ...o,
    }
}

describe('useBackupTask', () => {
    let unmount: () => void
    afterEach(() => {
        unmount?.()
        vi.restoreAllMocks()
    })

    it('runBackup triggers the backup task', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([makeExecution({ status: 'running' })])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useBackupTask(), { queryClient: qc })
        unmount = w.unmount

        await w.result.runBackup()

        expect(mockTriggerTask).toHaveBeenCalledWith('backup')
    })

    it('tracks the execution as running', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([makeExecution({ status: 'running' })])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useBackupTask(), { queryClient: qc })
        unmount = w.unmount

        await w.result.runBackup()

        await vi.waitFor(() => expect(w.result.isBackupRunning.value).toBe(true))
    })

    it('records a failed status and stops reporting running when the execution fails', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([
            makeExecution({ status: 'failed', ended_at: '2026-06-06T10:00:05Z' }),
        ])

        const qc = createTestQueryClient()
        const w = renderComposable(() => useBackupTask(), { queryClient: qc })
        unmount = w.unmount

        await w.result.runBackup()

        await vi.waitFor(() => expect(w.result.lastBackupStatus.value).toBe('failed'))
        expect(w.result.isBackupRunning.value).toBe(false)
    })

    it('invalidates the backup files list when the tracked execution completes', async () => {
        mockTriggerTask.mockResolvedValue('exec-1')
        mockListExecutions.mockResolvedValue([
            makeExecution({ status: 'complete', ended_at: '2026-06-06T10:00:05Z' }),
        ])

        const qc = createTestQueryClient()
        const spy = vi.spyOn(qc, 'invalidateQueries')
        const w = renderComposable(() => useBackupTask(), { queryClient: qc })
        unmount = w.unmount

        await w.result.runBackup()

        await vi.waitFor(() => expect(w.result.lastBackupStatus.value).toBe('complete'))
        expect(spy).toHaveBeenCalledWith({ queryKey: ['backupFiles'] })
    })
})
