import { describe, it, expect } from 'vitest'
import { buildTree, buildTreeForTable } from './convertToTree'

describe('buildTree', () => {
    it('returns empty array for empty input', () => {
        expect(buildTree([])).toEqual([])
    })

    it('returns empty array for null/undefined-like falsy input', () => {
        // The function checks !items, so passing null/undefined (cast) should return []
        expect(buildTree(null as any)).toEqual([])
        expect(buildTree(undefined as any)).toEqual([])
    })

    it('builds a flat list with no parents into multiple roots', () => {
        const items = [
            { id: 1, parentId: null, name: 'A' },
            { id: 2, parentId: null, name: 'B' },
            { id: 3, parentId: null, name: 'C' }
        ]
        const result = buildTree(items)
        expect(result).toHaveLength(3)
        result.forEach((node: any) => {
            expect(node.children).toEqual([])
        })
    })

    it('builds a simple parent-child tree', () => {
        const items = [
            { id: 1, parentId: null, name: 'Root' },
            { id: 2, parentId: 1, name: 'Child' }
        ]
        const result = buildTree(items)
        expect(result).toHaveLength(1)
        expect(result[0].name).toBe('Root')
        expect(result[0].children).toHaveLength(1)
        expect(result[0].children[0].name).toBe('Child')
        expect(result[0].children[0].children).toEqual([])
    })

    it('handles multiple roots with children', () => {
        const items = [
            { id: 1, parentId: null, name: 'Root1' },
            { id: 2, parentId: null, name: 'Root2' },
            { id: 3, parentId: 1, name: 'Child of Root1' },
            { id: 4, parentId: 2, name: 'Child of Root2' }
        ]
        const result = buildTree(items)
        expect(result).toHaveLength(2)
        expect(result[0].children).toHaveLength(1)
        expect(result[0].children[0].name).toBe('Child of Root1')
        expect(result[1].children).toHaveLength(1)
        expect(result[1].children[0].name).toBe('Child of Root2')
    })

    it('builds deeply nested items', () => {
        const items = [
            { id: 1, parentId: null, name: 'Level 0' },
            { id: 2, parentId: 1, name: 'Level 1' },
            { id: 3, parentId: 2, name: 'Level 2' },
            { id: 4, parentId: 3, name: 'Level 3' }
        ]
        const result = buildTree(items)
        expect(result).toHaveLength(1)
        const level0 = result[0]
        expect(level0.name).toBe('Level 0')
        const level1 = level0.children[0]
        expect(level1.name).toBe('Level 1')
        const level2 = level1.children[0]
        expect(level2.name).toBe('Level 2')
        const level3 = level2.children[0]
        expect(level3.name).toBe('Level 3')
        expect(level3.children).toEqual([])
    })

    it('treats items with orphan parentId (parent not in list) as lost nodes', () => {
        // parentId references a non-existent parent — node won't appear as root or child
        const items = [
            { id: 1, parentId: null, name: 'Root' },
            { id: 2, parentId: 999, name: 'Orphan' }
        ]
        const result = buildTree(items)
        // Only the root shows up; the orphan is silently dropped
        expect(result).toHaveLength(1)
        expect(result[0].name).toBe('Root')
    })

    it('does not mutate original items', () => {
        const items = [
            { id: 1, parentId: null, name: 'Root' },
            { id: 2, parentId: 1, name: 'Child' }
        ]
        const copy = JSON.parse(JSON.stringify(items))
        buildTree(items)
        expect(items).toEqual(copy)
    })

    it('handles a single item', () => {
        const items = [{ id: 1, parentId: null, name: 'Only' }]
        const result = buildTree(items)
        expect(result).toHaveLength(1)
        expect(result[0].name).toBe('Only')
        expect(result[0].children).toEqual([])
    })
})

describe('buildTreeForTable', () => {
    it('returns empty array for undefined input', () => {
        expect(buildTreeForTable(undefined)).toEqual([])
    })

    it('returns empty array for empty array input', () => {
        expect(buildTreeForTable([])).toEqual([])
    })

    it('produces PrimeVue TreeTable format with key, data, children', () => {
        const items = [
            { id: 1, parentId: null, name: 'Root' },
            { id: 2, parentId: 1, name: 'Child' }
        ]
        const result = buildTreeForTable(items)
        expect(result).toHaveLength(1)

        const root = result[0]
        expect(root.key).toBe('1')
        expect(root.data).toEqual({ id: 1, parentId: null, name: 'Root' })
        expect(root.children).toHaveLength(1)

        const child = root.children[0]
        expect(child.key).toBe('2')
        expect(child.data).toEqual({ id: 2, parentId: 1, name: 'Child' })
        // Leaf nodes have children: undefined (no array initialized)
        expect(child.children).toBeUndefined()
    })

    it('handles multiple roots', () => {
        const items = [
            { id: 1, parentId: null, name: 'A' },
            { id: 2, parentId: null, name: 'B' }
        ]
        const result = buildTreeForTable(items)
        expect(result).toHaveLength(2)
        expect(result[0].key).toBe('1')
        expect(result[1].key).toBe('2')
        // Leaf nodes remain with children undefined
        expect(result[0].children).toBeUndefined()
        expect(result[1].children).toBeUndefined()
    })

    it('builds deeply nested items in TreeTable format', () => {
        const items = [
            { id: 10, parentId: null, name: 'L0' },
            { id: 20, parentId: 10, name: 'L1' },
            { id: 30, parentId: 20, name: 'L2' },
            { id: 40, parentId: 30, name: 'L3' }
        ]
        const result = buildTreeForTable(items)
        expect(result).toHaveLength(1)

        const l0 = result[0]
        expect(l0.key).toBe('10')
        expect(l0.children).toHaveLength(1)

        const l1 = l0.children[0]
        expect(l1.key).toBe('20')
        expect(l1.children).toHaveLength(1)

        const l2 = l1.children[0]
        expect(l2.key).toBe('30')
        expect(l2.children).toHaveLength(1)

        const l3 = l2.children[0]
        expect(l3.key).toBe('40')
        expect(l3.data.name).toBe('L3')
        expect(l3.children).toBeUndefined()
    })

    it('key is always a string', () => {
        const items = [{ id: 42, parentId: null, name: 'X' }]
        const result = buildTreeForTable(items)
        expect(typeof result[0].key).toBe('string')
        expect(result[0].key).toBe('42')
    })

    it('data contains all original item fields', () => {
        const items = [{ id: 1, parentId: null, name: 'Test', amount: 100, currency: 'EUR' }]
        const result = buildTreeForTable(items)
        expect(result[0].data).toEqual({ id: 1, parentId: null, name: 'Test', amount: 100, currency: 'EUR' })
    })

    it('orphan items with non-existent parentId are dropped', () => {
        const items = [
            { id: 1, parentId: null, name: 'Root' },
            { id: 2, parentId: 999, name: 'Orphan' }
        ]
        const result = buildTreeForTable(items)
        expect(result).toHaveLength(1)
        expect(result[0].data.name).toBe('Root')
    })
})
