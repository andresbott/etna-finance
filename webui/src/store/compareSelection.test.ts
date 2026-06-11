import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCompareSelection, MAX_COMPARE } from './compareSelection'

describe('compareSelection store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
    })

    it('starts empty and cannot compare', () => {
        const store = useCompareSelection()
        expect(store.selectedIds).toEqual([])
        expect(store.count).toBe(0)
        expect(store.canCompare).toBe(false)
    })

    it('toggle adds an id and returns true', () => {
        const store = useCompareSelection()
        expect(store.toggle(5)).toBe(true)
        expect(store.selectedIds).toEqual([5])
        expect(store.isSelected(5)).toBe(true)
    })

    it('toggle removes an already-selected id', () => {
        const store = useCompareSelection()
        store.toggle(5)
        expect(store.toggle(5)).toBe(true)
        expect(store.selectedIds).toEqual([])
        expect(store.isSelected(5)).toBe(false)
    })

    it('canCompare becomes true at 2 selections', () => {
        const store = useCompareSelection()
        store.toggle(1)
        expect(store.canCompare).toBe(false)
        store.toggle(2)
        expect(store.canCompare).toBe(true)
    })

    it('rejects the 11th selection and leaves state unchanged', () => {
        const store = useCompareSelection()
        for (let i = 1; i <= MAX_COMPARE; i++) expect(store.toggle(i)).toBe(true)
        expect(store.count).toBe(MAX_COMPARE)
        expect(store.toggle(99)).toBe(false)
        expect(store.count).toBe(MAX_COMPARE)
        expect(store.isSelected(99)).toBe(false)
    })

    it('can still deselect when at the cap', () => {
        const store = useCompareSelection()
        for (let i = 1; i <= MAX_COMPARE; i++) store.toggle(i)
        expect(store.toggle(1)).toBe(true)
        expect(store.count).toBe(MAX_COMPARE - 1)
    })

    it('clear empties the selection', () => {
        const store = useCompareSelection()
        store.toggle(1)
        store.toggle(2)
        store.clear()
        expect(store.selectedIds).toEqual([])
        expect(store.canCompare).toBe(false)
    })

    it('setSelection replaces the selection, dedupes, and respects the cap', () => {
        const store = useCompareSelection()
        store.toggle(1)
        store.setSelection([5, 6, 6, 7])
        expect(store.selectedIds).toEqual([5, 6, 7])

        store.setSelection(Array.from({ length: MAX_COMPARE + 3 }, (_, i) => i + 1))
        expect(store.count).toBe(MAX_COMPARE)
    })
})
