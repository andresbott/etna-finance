import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import {
    getInstruments,
    createInstrument,
    updateInstrument,
    deleteInstrument
} from '@/lib/api/Instrument'
import type { CreateInstrumentDTO, UpdateInstrumentDTO } from '@/types/instrument'

const INSTRUMENTS_QUERY_KEY = ['instruments']

export function useInstruments() {
    const queryClient = useQueryClient()

    const invalidateAndRefetch = () => {
        queryClient.invalidateQueries({ queryKey: INSTRUMENTS_QUERY_KEY })
        queryClient.refetchQueries({ queryKey: INSTRUMENTS_QUERY_KEY })
    }

    const instrumentsQuery = useQuery({
        queryKey: INSTRUMENTS_QUERY_KEY,
        queryFn: getInstruments
    })

    const createInstrumentMutation = useMutation({
        mutationFn: (payload: CreateInstrumentDTO) => createInstrument(payload),
        onSuccess: invalidateAndRefetch
    })

    const updateInstrumentMutation = useMutation({
        mutationFn: ({
            id,
            payload
        }: { id: number; payload: UpdateInstrumentDTO }) => updateInstrument(id, payload),
        onSuccess: invalidateAndRefetch
    })

    const deleteInstrumentMutation = useMutation({
        mutationFn: (id: number) => deleteInstrument(id),
        onSuccess: invalidateAndRefetch
    })

    return {
        instruments: instrumentsQuery.data,
        isLoading: instrumentsQuery.isLoading,
        isError: instrumentsQuery.isError,
        error: instrumentsQuery.error,
        refetch: instrumentsQuery.refetch,

        createInstrument: createInstrumentMutation.mutateAsync,
        updateInstrument: updateInstrumentMutation.mutateAsync,
        deleteInstrument: deleteInstrumentMutation.mutateAsync,

        isCreatingInstrument: createInstrumentMutation.isPending,
        isUpdatingInstrument: updateInstrumentMutation.isPending,
        isDeleting: deleteInstrumentMutation.isPending
    }
}
