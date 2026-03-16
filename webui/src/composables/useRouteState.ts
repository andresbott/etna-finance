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

function parseIds(s: string): number[] {
    if (!s) return []
    return s.split(',').map(Number).filter((n) => !isNaN(n) && n >= 0)
}

function parseStrings(s: string): string[] {
    if (!s) return []
    return s.split(',').filter(Boolean)
}

/**
 * Syncs date range, pagination, and filter state with URL query parameters.
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

    // Filter params
    const categoryIds = ref<number[]>(parseIds(route.query.categoryIds as string))
    const types = ref<string[]>(parseStrings(route.query.types as string))
    const hasAttachment = ref(route.query.hasAttachment === 'true')
    const search = ref((route.query.search as string) || '')

    let updating = false
    let needsResync = false

    function syncToRoute() {
        if (updating) {
            needsResync = true
            return
        }
        updating = true
        needsResync = false

        const query: Record<string, string> = {
            ...route.query as Record<string, string>,
            from: formatDate(startDate.value),
            to: formatDate(endDate.value),
            page: String(page.value),
            limit: String(limit.value)
        }

        if (categoryIds.value.length > 0) {
            query.categoryIds = categoryIds.value.join(',')
        } else {
            delete query.categoryIds
        }
        if (types.value.length > 0) {
            query.types = types.value.join(',')
        } else {
            delete query.types
        }
        if (hasAttachment.value) {
            query.hasAttachment = 'true'
        } else {
            delete query.hasAttachment
        }
        if (search.value) {
            query.search = search.value
        } else {
            delete query.search
        }

        router.replace({ query }).catch(() => {}).finally(() => {
            updating = false
            if (needsResync) {
                syncToRoute()
            }
        })
    }

    watch([startDate, endDate, page, limit, categoryIds, types, hasAttachment, search], syncToRoute, { immediate: true })

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

            const qCategoryIds = parseIds(q.categoryIds as string)
            if (JSON.stringify(qCategoryIds) !== JSON.stringify(categoryIds.value)) categoryIds.value = qCategoryIds

            const qTypes = parseStrings(q.types as string)
            if (JSON.stringify(qTypes) !== JSON.stringify(types.value)) types.value = qTypes

            const qHasAttachment = q.hasAttachment === 'true'
            if (qHasAttachment !== hasAttachment.value) hasAttachment.value = qHasAttachment

            const qSearch = (q.search as string) || ''
            if (qSearch !== search.value) search.value = qSearch
        }
    )

    return { startDate, endDate, page, limit, categoryIds, types, hasAttachment, search }
}
