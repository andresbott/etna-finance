import type { QueryClient } from '@tanstack/vue-query'

/**
 * Invalidates and refetches queries for the given query key.
 * Use after mutations so the UI shows fresh data.
 */
export function invalidateAndRefetch(queryClient: QueryClient, queryKey: unknown[]): void {
    queryClient.invalidateQueries({ queryKey })
    queryClient.refetchQueries({ queryKey })
}
