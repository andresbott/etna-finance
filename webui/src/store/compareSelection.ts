import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const MAX_COMPARE = 10

export const useCompareSelection = defineStore('compareSelection', () => {
    const selectedIds = ref<number[]>([])

    const count = computed(() => selectedIds.value.length)
    const canCompare = computed(() => selectedIds.value.length >= 2)

    const isSelected = (id: number) => selectedIds.value.includes(id)

    // Returns false when the action was rejected (cap reached), so callers can toast.
    const toggle = (id: number): boolean => {
        const idx = selectedIds.value.indexOf(id)
        if (idx >= 0) {
            selectedIds.value.splice(idx, 1)
            return true
        }
        if (selectedIds.value.length >= MAX_COMPARE) return false
        selectedIds.value.push(id)
        return true
    }

    const clear = () => {
        selectedIds.value = []
    }

    // Replace the whole selection at once (e.g. when seeding from a shared URL),
    // keeping the same cap as toggle().
    const setSelection = (ids: number[]) => {
        const unique = [...new Set(ids)]
        selectedIds.value = unique.slice(0, MAX_COMPARE)
    }

    return { selectedIds, count, canCompare, isSelected, toggle, clear, setSelection }
})
