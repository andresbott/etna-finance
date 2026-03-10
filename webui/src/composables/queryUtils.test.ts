import { describe, it, expect, vi } from 'vitest'
import type { QueryClient } from '@tanstack/vue-query'
import { invalidateAndRefetch } from './queryUtils'

describe('invalidateAndRefetch', () => {
    function createMockQueryClient() {
        return {
            invalidateQueries: vi.fn(),
            refetchQueries: vi.fn(),
        } as unknown as QueryClient
    }

    it('calls invalidateQueries with the provided queryKey', () => {
        const qc = createMockQueryClient()
        const key = ['transactions']

        invalidateAndRefetch(qc, key)

        expect(qc.invalidateQueries).toHaveBeenCalledOnce()
        expect(qc.invalidateQueries).toHaveBeenCalledWith({ queryKey: key })
    })

    it('calls refetchQueries with the provided queryKey', () => {
        const qc = createMockQueryClient()
        const key = ['transactions']

        invalidateAndRefetch(qc, key)

        expect(qc.refetchQueries).toHaveBeenCalledOnce()
        expect(qc.refetchQueries).toHaveBeenCalledWith({ queryKey: key })
    })

    it('passes compound query keys correctly', () => {
        const qc = createMockQueryClient()
        const key = ['accounts', 42, 'details']

        invalidateAndRefetch(qc, key)

        expect(qc.invalidateQueries).toHaveBeenCalledWith({ queryKey: key })
        expect(qc.refetchQueries).toHaveBeenCalledWith({ queryKey: key })
    })

    it('calls both invalidate and refetch (not just one)', () => {
        const qc = createMockQueryClient()

        invalidateAndRefetch(qc, ['any'])

        expect(qc.invalidateQueries).toHaveBeenCalledOnce()
        expect(qc.refetchQueries).toHaveBeenCalledOnce()
    })
})
