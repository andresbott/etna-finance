import { describe, it, expect } from 'vitest'
import { findNodeById, buildCategoryPath, type CategoryNode } from './categoryUtils'

// Sample tree data used across tests
const sampleTree: CategoryNode[] = [
    {
        key: '1',
        label: 'Food',
        data: { id: 1, name: 'Food', path: 'Food', description: 'Food expenses' },
        children: [
            {
                key: '2',
                label: 'Groceries',
                data: { id: 2, parentId: 1, name: 'Groceries', path: 'Food > Groceries' },
                children: [
                    {
                        key: '5',
                        label: 'Vegetables',
                        data: { id: 5, parentId: 2, name: 'Vegetables', path: 'Food > Groceries > Vegetables' },
                    },
                ],
            },
            {
                key: '3',
                label: 'Restaurants',
                data: { id: 3, parentId: 1, name: 'Restaurants', path: 'Food > Restaurants' },
            },
        ],
    },
    {
        key: '4',
        label: 'Transport',
        data: { id: 4, name: 'Transport', path: 'Transport' },
    },
]

describe('findNodeById', () => {
    it('finds a root-level node by numeric id', () => {
        const result = findNodeById(sampleTree, 4)
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Transport')
    })

    it('finds a root-level node by string key', () => {
        const result = findNodeById(sampleTree, '4')
        expect(result).not.toBeNull()
        expect(result!.data?.id).toBe(4)
    })

    it('finds a nested child node', () => {
        const result = findNodeById(sampleTree, 2)
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Groceries')
    })

    it('finds a deeply nested node', () => {
        const result = findNodeById(sampleTree, 5)
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Vegetables')
    })

    it('returns null when id does not exist', () => {
        const result = findNodeById(sampleTree, 999)
        expect(result).toBeNull()
    })

    it('returns null for an empty tree', () => {
        const result = findNodeById([], 1)
        expect(result).toBeNull()
    })

    it('matches by key string even when data.id differs', () => {
        const tree: CategoryNode[] = [
            { key: '10', label: 'Special', data: { id: 99, name: 'Special', path: '' } },
        ]
        // Searching with string '10' should match via key
        const result = findNodeById(tree, '10')
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Special')
    })

    it('matches by data.id even when key is different format', () => {
        const tree: CategoryNode[] = [
            { key: 'cat-7', label: 'Custom', data: { id: 7, name: 'Custom', path: '' } },
        ]
        const result = findNodeById(tree, 7)
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Custom')
    })

    it('handles nodes without children property', () => {
        const tree: CategoryNode[] = [
            { key: '1', label: 'Leaf' },
        ]
        const result = findNodeById(tree, '1')
        expect(result).not.toBeNull()
        expect(result!.label).toBe('Leaf')
    })

    it('handles nodes without data property', () => {
        const tree: CategoryNode[] = [
            { key: '1', label: 'NoData' },
        ]
        const result = findNodeById(tree, '1')
        expect(result).not.toBeNull()
        // Searching by numeric id should not match a node without data
        const result2 = findNodeById(tree, 1)
        // key '1' === String(1) so it should still match
        expect(result2).not.toBeNull()
    })
})

describe('buildCategoryPath', () => {
    it('returns the data.path when available', () => {
        const path = buildCategoryPath(sampleTree, 2)
        expect(path).toBe('Food > Groceries')
    })

    it('returns the data.path for a deeply nested node', () => {
        const path = buildCategoryPath(sampleTree, 5)
        expect(path).toBe('Food > Groceries > Vegetables')
    })

    it('returns the data.path for a root node', () => {
        const path = buildCategoryPath(sampleTree, 1)
        expect(path).toBe('Food')
    })

    it('returns "Unknown" when id does not exist', () => {
        const path = buildCategoryPath(sampleTree, 999)
        expect(path).toBe('Unknown')
    })

    it('returns "Unknown" for an empty tree', () => {
        const path = buildCategoryPath([], 1)
        expect(path).toBe('Unknown')
    })

    it('builds path by traversal when data.path is empty', () => {
        const tree: CategoryNode[] = [
            {
                key: '10',
                label: 'Root',
                data: { id: 10, name: 'Root', path: '' },
                children: [
                    {
                        key: '20',
                        label: 'Child',
                        data: { id: 20, parentId: 10, name: 'Child', path: '' },
                    },
                ],
            },
        ]
        const path = buildCategoryPath(tree, 20)
        expect(path).toBe('Root > Child')
    })

    it('builds path by traversal for three-level deep tree with empty paths', () => {
        const tree: CategoryNode[] = [
            {
                key: '10',
                label: 'Root',
                data: { id: 10, name: 'Root', path: '' },
                children: [
                    {
                        key: '20',
                        label: 'Mid',
                        data: { id: 20, parentId: 10, name: 'Mid', path: '' },
                        children: [
                            {
                                key: '30',
                                label: 'Leaf',
                                data: { id: 30, parentId: 20, name: 'Leaf', path: '' },
                            },
                        ],
                    },
                ],
            },
        ]
        const path = buildCategoryPath(tree, 30)
        expect(path).toBe('Root > Mid > Leaf')
    })

    it('uses label as fallback when data.name is missing during traversal', () => {
        const tree: CategoryNode[] = [
            {
                key: '10',
                label: 'RootLabel',
                children: [
                    {
                        key: '20',
                        label: 'ChildLabel',
                        data: { id: 20, parentId: 10, name: 'ChildName', path: '' },
                    },
                ],
            },
        ]
        // Parent node (key=10) has no data.name, so label 'RootLabel' should be used
        const path = buildCategoryPath(tree, 20)
        expect(path).toBe('RootLabel > ChildName')
    })

    it('handles single node with no parent (no data.path)', () => {
        const tree: CategoryNode[] = [
            {
                key: '10',
                label: 'Solo',
                data: { id: 10, name: 'Solo', path: '' },
            },
        ]
        const path = buildCategoryPath(tree, 10)
        // Empty path, no parentId, so just the node name
        expect(path).toBe('Solo')
    })

    it('works with string id parameter', () => {
        const path = buildCategoryPath(sampleTree, '3')
        expect(path).toBe('Food > Restaurants')
    })

    it('does not loop infinitely on circular parentId references', () => {
        const tree: CategoryNode[] = [
            {
                key: '1',
                label: 'A',
                data: { id: 1, parentId: 2, name: 'A', path: '' },
            },
            {
                key: '2',
                label: 'B',
                data: { id: 2, parentId: 1, name: 'B', path: '' },
            },
        ]
        // Should not hang — visited set prevents infinite loop
        const path = buildCategoryPath(tree, 1)
        // The exact output depends on traversal order, just verify it terminates and returns a string
        expect(typeof path).toBe('string')
        expect(path.length).toBeGreaterThan(0)
    })
})
