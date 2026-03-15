import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, reactive, nextTick, effectScope } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { useRouteState } from './useRouteState'

const mockQuery = reactive<Record<string, string>>({})
const mockReplace = vi.fn()

vi.mock('vue-router', () => ({
    useRoute: () => ({ query: mockQuery, params: {} }),
    useRouter: () => ({ replace: mockReplace }),
}))

function setupRouteState(
    query: Record<string, string>,
    defaults: { startDate: Date; endDate: Date; page?: number; limit?: number }
) {
    // Reset query to provided values
    Object.keys(mockQuery).forEach((k) => delete mockQuery[k])
    Object.assign(mockQuery, query)
    mockReplace.mockReset()
    mockReplace.mockResolvedValue(undefined)

    let result!: ReturnType<typeof useRouteState>
    const scope = effectScope()
    scope.run(() => {
        result = useRouteState(defaults)
    })
    return { result, scope }
}

const defaults = {
    startDate: new Date(2026, 0, 15),
    endDate: new Date(2026, 2, 15),
}

describe('useRouteState', () => {
    beforeEach(() => {
        Object.keys(mockQuery).forEach((k) => delete mockQuery[k])
        mockReplace.mockReset()
        mockReplace.mockResolvedValue(undefined)
    })

    describe('initialization', () => {
        it('uses defaults when no query params present', () => {
            const { result, scope } = setupRouteState({}, defaults)

            expect(result.startDate.value).toEqual(defaults.startDate)
            expect(result.endDate.value).toEqual(defaults.endDate)
            expect(result.page.value).toBe(1)
            expect(result.limit.value).toBe(25)
            scope.stop()
        })

        it('reads dates from query params', () => {
            const { result, scope } = setupRouteState(
                { from: '2025-06-01', to: '2025-12-31' },
                defaults
            )

            expect(result.startDate.value).toEqual(new Date(2025, 5, 1))
            expect(result.endDate.value).toEqual(new Date(2025, 11, 31))
            scope.stop()
        })

        it('reads page from query params', () => {
            const { result, scope } = setupRouteState(
                { from: '2026-01-01', to: '2026-03-15', page: '3' },
                defaults
            )

            expect(result.page.value).toBe(3)
            scope.stop()
        })

        it('reads limit from query params', () => {
            const { result, scope } = setupRouteState(
                { from: '2026-01-01', to: '2026-03-15', limit: '50' },
                defaults
            )

            expect(result.limit.value).toBe(50)
            scope.stop()
        })

        it('uses default limit when not in query', () => {
            const { result, scope } = setupRouteState({}, { ...defaults, limit: 100 })

            expect(result.limit.value).toBe(100)
            scope.stop()
        })

        it('ignores invalid page values', () => {
            const { result, scope } = setupRouteState(
                { page: '-1' },
                defaults
            )

            expect(result.page.value).toBe(1)
            scope.stop()
        })

        it('ignores invalid limit values', () => {
            const { result, scope } = setupRouteState(
                { limit: '0' },
                defaults
            )

            expect(result.limit.value).toBe(25)
            scope.stop()
        })

        it('ignores malformed date strings', () => {
            const { result, scope } = setupRouteState(
                { from: 'not-a-date', to: 'abc' },
                defaults
            )

            expect(result.startDate.value).toEqual(defaults.startDate)
            expect(result.endDate.value).toEqual(defaults.endDate)
            scope.stop()
        })
    })

    describe('syncs ref changes to URL', () => {
        it('calls router.replace when startDate changes', async () => {
            const { result, scope } = setupRouteState({}, defaults)
            mockReplace.mockClear()

            result.startDate.value = new Date(2026, 0, 1)
            await nextTick()

            expect(mockReplace).toHaveBeenCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({ from: '2026-01-01' }),
                })
            )
            scope.stop()
        })

        it('calls router.replace when endDate changes', async () => {
            const { result, scope } = setupRouteState({}, defaults)
            mockReplace.mockClear()

            result.endDate.value = new Date(2026, 11, 31)
            await nextTick()

            expect(mockReplace).toHaveBeenCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({ to: '2026-12-31' }),
                })
            )
            scope.stop()
        })

        it('calls router.replace when page changes', async () => {
            const { result, scope } = setupRouteState({}, defaults)
            mockReplace.mockClear()

            result.page.value = 5
            await nextTick()

            expect(mockReplace).toHaveBeenCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({ page: '5' }),
                })
            )
            scope.stop()
        })

        it('calls router.replace when limit changes', async () => {
            const { result, scope } = setupRouteState({}, defaults)
            mockReplace.mockClear()

            result.limit.value = 50
            await nextTick()

            expect(mockReplace).toHaveBeenCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({ limit: '50' }),
                })
            )
            scope.stop()
        })
    })

    describe('resync after concurrent changes', () => {
        it('re-syncs when changes occur during a pending router.replace', async () => {
            const { result, scope } = setupRouteState({}, defaults)

            // Let the immediate watcher's replace resolve
            await flushPromises()
            const callsAfterInit = mockReplace.mock.calls.length

            // Now make the next replace hang
            let resolveHanging!: () => void
            mockReplace.mockImplementationOnce(
                () => new Promise<void>((r) => { resolveHanging = r })
            )

            // Trigger a change — this call will hang
            result.page.value = 3
            await nextTick()
            expect(mockReplace).toHaveBeenCalledTimes(callsAfterInit + 1)

            // Change again while the replace is still pending
            result.page.value = 5
            await nextTick()

            // The second change was skipped (updating flag), no extra call yet
            expect(mockReplace).toHaveBeenCalledTimes(callsAfterInit + 1)

            // Resolve the hanging replace — needsResync triggers a re-sync
            resolveHanging()
            await flushPromises()

            expect(mockReplace).toHaveBeenCalledTimes(callsAfterInit + 2)
            expect(mockReplace).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({ page: '5' }),
                })
            )
            scope.stop()
        })

        it('captures latest values when multiple changes happen during pending replace', async () => {
            const { result, scope } = setupRouteState({}, defaults)
            await flushPromises()

            // Make the next replace hang
            let resolveHanging!: () => void
            mockReplace.mockImplementationOnce(
                () => new Promise<void>((r) => { resolveHanging = r })
            )

            // Trigger a change to start the hanging replace
            result.page.value = 99
            await nextTick()

            // Multiple changes while replace is pending
            result.startDate.value = new Date(2026, 0, 1)
            result.endDate.value = new Date(2026, 11, 31)
            result.page.value = 2
            result.limit.value = 100
            await nextTick()

            resolveHanging()
            await flushPromises()

            expect(mockReplace).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    query: expect.objectContaining({
                        from: '2026-01-01',
                        to: '2026-12-31',
                        page: '2',
                        limit: '100',
                    }),
                })
            )
            scope.stop()
        })
    })
})
