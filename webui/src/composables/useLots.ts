import { computed } from 'vue'
import type { Ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { getLots } from '@/lib/api/Portfolio'

export function useLots(accountId: Ref<number | null>, instrumentId: Ref<number | null>) {
    const { data, isLoading } = useQuery({
        queryKey: computed(() => ['portfolio-lots', accountId.value, instrumentId.value]),
        queryFn: () => getLots(accountId.value!, instrumentId.value!),
        enabled: computed(() => accountId.value != null && instrumentId.value != null),
        staleTime: 30_000
    })

    const lots = computed(() => data.value ?? [])
    return { lots, isLoading }
}
