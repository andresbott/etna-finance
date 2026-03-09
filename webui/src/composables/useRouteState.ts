import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

function formatDate(d: Date): string {
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
}

function parseDate(s: string): Date | null {
    const parts = s.match(/^(\d{4})-(\d{2})-(\d{2})$/)
    if (!parts) return null
    const d = new Date(Number(parts[1]), Number(parts[2]) - 1, Number(parts[3]))
    return isNaN(d.getTime()) ? null : d
}

/**
 * Syncs startDate, endDate, page, and limit with URL query parameters.
 * On init, reads from query params if present; otherwise uses provided defaults.
 * On change, updates the URL without a full navigation.
 */
export function useRouteState(defaults: {
    startDate: Date
    endDate: Date
    page?: number
    limit?: number
}) {
    const route = useRoute()
    const router = useRouter()

    const initFrom = route.query.from ? parseDate(route.query.from as string) : null
    const initTo = route.query.to ? parseDate(route.query.to as string) : null
    const initPage = route.query.page ? Number(route.query.page) : null
    const initLimit = route.query.limit ? Number(route.query.limit) : null

    const startDate = ref(initFrom ?? defaults.startDate)
    const endDate = ref(initTo ?? defaults.endDate)
    const page = ref(initPage && initPage > 0 ? initPage : (defaults.page ?? 1))
    const limit = ref(initLimit && initLimit > 0 ? initLimit : (defaults.limit ?? 25))

    let updating = false

    function syncToRoute() {
        if (updating) return
        updating = true
        router.replace({
            query: {
                ...route.query,
                from: formatDate(startDate.value),
                to: formatDate(endDate.value),
                page: String(page.value),
                limit: String(limit.value)
            }
        }).finally(() => { updating = false })
    }

    watch([startDate, endDate, page, limit], syncToRoute, { immediate: true })

    // When navigating back/forward, pick up query changes
    watch(
        () => route.query,
        (q) => {
            if (updating) return
            const qFrom = q.from ? parseDate(q.from as string) : null
            const qTo = q.to ? parseDate(q.to as string) : null
            const qPage = q.page ? Number(q.page) : null
            const qLimit = q.limit ? Number(q.limit) : null

            if (qFrom && formatDate(qFrom) !== formatDate(startDate.value)) startDate.value = qFrom
            if (qTo && formatDate(qTo) !== formatDate(endDate.value)) endDate.value = qTo
            if (qPage && qPage > 0 && qPage !== page.value) page.value = qPage
            if (qLimit && qLimit > 0 && qLimit !== limit.value) limit.value = qLimit
        }
    )

    return { startDate, endDate, page, limit }
}
