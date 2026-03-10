import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { createEntry, updateEntry, deleteEntry } from '@/lib/api/Entry'
import type { CreateEntryDTO, UpdateEntryDTO } from '@/types/entry'

export function useEntryMutations() {
    const queryClient = useQueryClient()

    const createEntryMutation = useMutation({
        mutationFn: (payload: CreateEntryDTO) => createEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    const updateEntryMutation = useMutation({
        mutationFn: (payload: UpdateEntryDTO) => updateEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    const deleteEntryMutation = useMutation({
        mutationFn: (id: string) => deleteEntry(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    return {
        createEntry: createEntryMutation.mutateAsync,
        updateEntry: updateEntryMutation.mutateAsync,
        deleteEntry: deleteEntryMutation.mutateAsync,
        isCreating: createEntryMutation.isPending,
        isUpdating: updateEntryMutation.isPending,
        isDeleting: deleteEntryMutation.isPending
    }
}
