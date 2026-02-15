import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import {
    getSecurities,
    createSecurity,
    updateSecurity,
    deleteSecurity
} from '@/lib/api/Security'
import type { CreateSecurityDTO, UpdateSecurityDTO } from '@/types/security'

const SECURITIES_QUERY_KEY = ['securities']

export function useSecurities() {
    const queryClient = useQueryClient()

    const securitiesQuery = useQuery({
        queryKey: SECURITIES_QUERY_KEY,
        queryFn: getSecurities
    })

    const createSecurityMutation = useMutation({
        mutationFn: (payload: CreateSecurityDTO) => createSecurity(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: SECURITIES_QUERY_KEY })
        }
    })

    const updateSecurityMutation = useMutation({
        mutationFn: ({ id, payload }: { id: number; payload: UpdateSecurityDTO }) =>
            updateSecurity(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: SECURITIES_QUERY_KEY })
        }
    })

    const deleteSecurityMutation = useMutation({
        mutationFn: (id: number) => deleteSecurity(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: SECURITIES_QUERY_KEY })
        }
    })

    return {
        securities: securitiesQuery.data,
        isLoading: securitiesQuery.isLoading,
        isError: securitiesQuery.isError,
        error: securitiesQuery.error,
        refetch: securitiesQuery.refetch,

        createSecurity: createSecurityMutation.mutateAsync,
        updateSecurity: updateSecurityMutation.mutateAsync,
        deleteSecurity: deleteSecurityMutation.mutateAsync,

        isCreating: createSecurityMutation.isPending,
        isUpdating: updateSecurityMutation.isPending,
        isDeleting: deleteSecurityMutation.isPending
    }
}
