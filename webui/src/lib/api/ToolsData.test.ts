import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import { listCases, getCase, createCase, updateCase, deleteCase, type CaseStudy } from './ToolsData'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockCase: CaseStudy = {
    id: 1,
    toolType: 'portfolio-simulator',
    name: 'Conservative',
    description: 'Low risk',
    expectedAnnualReturn: 4.5,
    params: { initialContribution: 10000, growthRatePct: 7, expenseRatioPct: 0.2, capitalGainTaxPct: 19, taxModel: 'exit' },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
}

describe('listCases', () => {
    it('calls GET /tools/{toolType}/cases and returns items', async () => {
        const items = [mockCase];
        (apiClient.get as Mock).mockResolvedValue({ data: items })

        const result = await listCases('portfolio-simulator')

        expect(apiClient.get).toHaveBeenCalledWith('/tools/portfolio-simulator/cases')
        expect(result).toEqual(items)
    })

    it('returns empty array when data is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: null })

        const result = await listCases('portfolio-simulator')

        expect(result).toEqual([])
    })
})

describe('getCase', () => {
    it('calls GET /tools/{toolType}/cases/{id}', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockCase })

        const result = await getCase('portfolio-simulator', 1)

        expect(apiClient.get).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1')
        expect(result).toEqual(mockCase)
    })
})

describe('createCase', () => {
    it('calls POST /tools/{toolType}/cases with payload', async () => {
        const payload = { name: 'New', description: 'desc', expectedAnnualReturn: 5.0, params: { x: 1 } };
        (apiClient.post as Mock).mockResolvedValue({ data: mockCase })

        const result = await createCase('portfolio-simulator', payload)

        expect(apiClient.post).toHaveBeenCalledWith('/tools/portfolio-simulator/cases', payload)
        expect(result).toEqual(mockCase)
    })
})

describe('updateCase', () => {
    it('calls PUT /tools/{toolType}/cases/{id} with payload', async () => {
        const payload = { name: 'Updated' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockCase, name: 'Updated' } })

        const result = await updateCase('portfolio-simulator', 1, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1', payload)
        expect(result.name).toBe('Updated')
    })
})

describe('deleteCase', () => {
    it('calls DELETE /tools/{toolType}/cases/{id}', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteCase('portfolio-simulator', 1)

        expect(apiClient.delete).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1')
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteCase('portfolio-simulator', 1)

        expect(result).toBeUndefined()
    })
})
