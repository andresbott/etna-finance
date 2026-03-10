import { describe, it, expect, vi } from 'vitest'
import type { QueryClient } from '@tanstack/vue-query'
import { invalidateAndRefetch } from './queryUtils'

describe('invalidateAndRefetch', () => {
    function createMockQueryClient() {
        return {
            invalidateQueries: vi.fn(),
        } as unknown as QueryClient
    }

    it('calls invalidateQueries with the provided queryKey', () => {
        const qc = createMockQueryClient()
        const key = ['transactions']

        invalidateAndRefetch(qc, key)

        expect(qc.invalidateQueries).toHaveBeenCalledOnce()
        expect(qc.invalidateQueries).toHaveBeenCalledWith({ queryKey: key })
    })

    it('does not call refetchQueries (invalidateQueries already triggers refetch for active queries)', () => {
        const qc = createMockQueryClient() as QueryClient & { refetchQueries: ReturnType<typeof vi.fn> }
        ;(qc as any).refetchQueries = vi.fn()
        const key = ['transactions']

        invalidateAndRefetch(qc, key)

        expect((qc as any).refetchQueries).not.toHaveBeenCalled()
    })

    it('passes compound query keys correctly', () => {
        const qc = createMockQueryClient()
        const key = ['accounts', 42, 'details']

        invalidateAndRefetch(qc, key)

        expect(qc.invalidateQueries).toHaveBeenCalledWith({ queryKey: key })
    })

    it('calls invalidateQueries exactly once', () => {
        const qc = createMockQueryClient()

        invalidateAndRefetch(qc, ['any'])

        expect(qc.invalidateQueries).toHaveBeenCalledOnce()
    })
})
