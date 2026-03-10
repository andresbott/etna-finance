import type { QueryClient } from '@tanstack/vue-query'

/**
 * Invalidates queries for the given query key.
 * invalidateQueries already triggers a refetch for active queries by default.
 */
export function invalidateAndRefetch(queryClient: QueryClient, queryKey: unknown[]): void {
    queryClient.invalidateQueries({ queryKey })
}
