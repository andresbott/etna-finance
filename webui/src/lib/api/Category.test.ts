import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
    GetIncomeCategories,
    CreateIncomeCategory,
    UpdateIncomeCategory,
    deleteIncomeCategory,
    GetExpenseCategories,
    CreateExpenseCategory,
    UpdateExpenseCategory,
    DeleteExpenseCategory
} from '@/lib/api/Category'

// Mock the apiClient module
vi.mock('@/lib/api/client', () => ({
    apiClient: {
        get: vi.fn(),
        post: vi.fn(),
        put: vi.fn(),
        delete: vi.fn()
    }
}))

// Import the mocked client so we can set return values
import { apiClient } from '@/lib/api/client'

const mockedClient = vi.mocked(apiClient, true)

beforeEach(() => {
    vi.clearAllMocks()
})

// ──────────────────────────────────────────────
//  INCOME CATEGORIES
// ──────────────────────────────────────────────

describe('GetIncomeCategories', () => {
    it('calls GET /fin/category/income and returns items', async () => {
        const categories = [
            { id: '1', name: 'Salary', description: 'Monthly salary' },
            { id: '2', name: 'Freelance' }
        ]
        mockedClient.get.mockResolvedValue({ data: { items: categories } })

        const result = await GetIncomeCategories()

        expect(mockedClient.get).toHaveBeenCalledWith('/fin/category/income')
        expect(mockedClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(categories)
    })

    it('returns an empty array when the API returns no items', async () => {
        mockedClient.get.mockResolvedValue({ data: { items: [] } })

        const result = await GetIncomeCategories()

        expect(result).toEqual([])
    })

    it('propagates API errors', async () => {
        mockedClient.get.mockRejectedValue(new Error('Network Error'))

        await expect(GetIncomeCategories()).rejects.toThrow('Network Error')
    })
})

describe('CreateIncomeCategory', () => {
    it('calls POST /fin/category/income with the payload and returns the created category', async () => {
        const payload = { name: 'Bonus', description: 'Year-end bonus', icon: 'gift' }
        const created = { id: '3', ...payload }
        mockedClient.post.mockResolvedValue({ data: created })

        const result = await CreateIncomeCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/income', payload)
        expect(mockedClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(created)
    })

    it('sends a minimal payload (name only)', async () => {
        const payload = { name: 'Interest' }
        const created = { id: '4', name: 'Interest' }
        mockedClient.post.mockResolvedValue({ data: created })

        const result = await CreateIncomeCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/income', payload)
        expect(result).toEqual(created)
    })

    it('sends parentId when provided', async () => {
        const payload = { name: 'Sub-salary', parentId: 1 }
        mockedClient.post.mockResolvedValue({ data: { id: '5', ...payload } })

        await CreateIncomeCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/income', payload)
    })

    it('propagates API errors', async () => {
        mockedClient.post.mockRejectedValue(new Error('Bad Request'))

        await expect(CreateIncomeCategory({ name: 'Fail' })).rejects.toThrow('Bad Request')
    })
})

describe('UpdateIncomeCategory', () => {
    it('calls PUT /fin/category/income/:id with the payload and returns updated category', async () => {
        const payload = { name: 'Updated Salary', description: 'Updated desc' }
        const updated = { id: '1', ...payload }
        mockedClient.put.mockResolvedValue({ data: updated })

        const result = await UpdateIncomeCategory({ id: 1, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/income/1', payload)
        expect(mockedClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual(updated)
    })

    it('sends a partial update (single field)', async () => {
        const payload = { icon: 'star' }
        mockedClient.put.mockResolvedValue({ data: { id: '2', name: 'Freelance', icon: 'star' } })

        const result = await UpdateIncomeCategory({ id: 2, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/income/2', payload)
        expect(result.icon).toBe('star')
    })

    it('interpolates the id into the URL correctly', async () => {
        const payload = { name: 'Test' }
        mockedClient.put.mockResolvedValue({ data: { id: '99', name: 'Test' } })

        await UpdateIncomeCategory({ id: 99, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/income/99', payload)
    })

    it('propagates API errors', async () => {
        mockedClient.put.mockRejectedValue(new Error('Not Found'))

        await expect(UpdateIncomeCategory({ id: 999, payload: { name: 'X' } })).rejects.toThrow(
            'Not Found'
        )
    })
})

describe('deleteIncomeCategory', () => {
    it('calls DELETE /fin/category/income/:id and returns void', async () => {
        mockedClient.delete.mockResolvedValue({ data: null })

        const result = await deleteIncomeCategory(1)

        expect(mockedClient.delete).toHaveBeenCalledWith('/fin/category/income/1')
        expect(mockedClient.delete).toHaveBeenCalledTimes(1)
        expect(result).toBeUndefined()
    })

    it('interpolates the id into the URL correctly', async () => {
        mockedClient.delete.mockResolvedValue({ data: null })

        await deleteIncomeCategory(42)

        expect(mockedClient.delete).toHaveBeenCalledWith('/fin/category/income/42')
    })

    it('propagates API errors', async () => {
        mockedClient.delete.mockRejectedValue(new Error('Forbidden'))

        await expect(deleteIncomeCategory(1)).rejects.toThrow('Forbidden')
    })
})

// ──────────────────────────────────────────────
//  EXPENSE CATEGORIES
// ──────────────────────────────────────────────

describe('GetExpenseCategories', () => {
    it('calls GET /fin/category/expense and returns items', async () => {
        const categories = [
            { id: '10', name: 'Food', description: 'Groceries and dining' },
            { id: '11', name: 'Transport' }
        ]
        mockedClient.get.mockResolvedValue({ data: { items: categories } })

        const result = await GetExpenseCategories()

        expect(mockedClient.get).toHaveBeenCalledWith('/fin/category/expense')
        expect(mockedClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(categories)
    })

    it('returns an empty array when the API returns no items', async () => {
        mockedClient.get.mockResolvedValue({ data: { items: [] } })

        const result = await GetExpenseCategories()

        expect(result).toEqual([])
    })

    it('propagates API errors', async () => {
        mockedClient.get.mockRejectedValue(new Error('Server Error'))

        await expect(GetExpenseCategories()).rejects.toThrow('Server Error')
    })
})

describe('CreateExpenseCategory', () => {
    it('calls POST /fin/category/expense with the payload and returns the created category', async () => {
        const payload = { name: 'Rent', description: 'Monthly rent', icon: 'home' }
        const created = { id: '12', ...payload }
        mockedClient.post.mockResolvedValue({ data: created })

        const result = await CreateExpenseCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/expense', payload)
        expect(mockedClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(created)
    })

    it('sends a minimal payload (name only)', async () => {
        const payload = { name: 'Utilities' }
        const created = { id: '13', name: 'Utilities' }
        mockedClient.post.mockResolvedValue({ data: created })

        const result = await CreateExpenseCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/expense', payload)
        expect(result).toEqual(created)
    })

    it('sends parentId when provided', async () => {
        const payload = { name: 'Electricity', parentId: 13 }
        mockedClient.post.mockResolvedValue({ data: { id: '14', ...payload } })

        await CreateExpenseCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/expense', payload)
    })

    it('handles null parentId', async () => {
        const payload = { name: 'Misc', parentId: null }
        mockedClient.post.mockResolvedValue({ data: { id: '15', name: 'Misc', parentId: null } })

        await CreateExpenseCategory(payload)

        expect(mockedClient.post).toHaveBeenCalledWith('/fin/category/expense', payload)
    })

    it('propagates API errors', async () => {
        mockedClient.post.mockRejectedValue(new Error('Validation Error'))

        await expect(CreateExpenseCategory({ name: '' })).rejects.toThrow('Validation Error')
    })
})

describe('UpdateExpenseCategory', () => {
    it('calls PUT /fin/category/expense/:id with the payload and returns updated category', async () => {
        const payload = { name: 'Updated Food', description: 'Updated desc' }
        const updated = { id: '10', ...payload }
        mockedClient.put.mockResolvedValue({ data: updated })

        const result = await UpdateExpenseCategory({ id: 10, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/expense/10', payload)
        expect(mockedClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual(updated)
    })

    it('sends a partial update (single field)', async () => {
        const payload = { icon: 'car' }
        mockedClient.put.mockResolvedValue({
            data: { id: '11', name: 'Transport', icon: 'car' }
        })

        const result = await UpdateExpenseCategory({ id: 11, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/expense/11', payload)
        expect(result.icon).toBe('car')
    })

    it('interpolates the id into the URL correctly', async () => {
        const payload = { name: 'Test' }
        mockedClient.put.mockResolvedValue({ data: { id: '77', name: 'Test' } })

        await UpdateExpenseCategory({ id: 77, payload })

        expect(mockedClient.put).toHaveBeenCalledWith('/fin/category/expense/77', payload)
    })

    it('propagates API errors', async () => {
        mockedClient.put.mockRejectedValue(new Error('Not Found'))

        await expect(
            UpdateExpenseCategory({ id: 999, payload: { name: 'X' } })
        ).rejects.toThrow('Not Found')
    })
})

describe('DeleteExpenseCategory', () => {
    it('calls DELETE /fin/category/expense/:id and returns void', async () => {
        mockedClient.delete.mockResolvedValue({ data: null })

        const result = await DeleteExpenseCategory(10)

        expect(mockedClient.delete).toHaveBeenCalledWith('/fin/category/expense/10')
        expect(mockedClient.delete).toHaveBeenCalledTimes(1)
        expect(result).toBeUndefined()
    })

    it('interpolates the id into the URL correctly', async () => {
        mockedClient.delete.mockResolvedValue({ data: null })

        await DeleteExpenseCategory(55)

        expect(mockedClient.delete).toHaveBeenCalledWith('/fin/category/expense/55')
    })

    it('propagates API errors', async () => {
        mockedClient.delete.mockRejectedValue(new Error('Forbidden'))

        await expect(DeleteExpenseCategory(10)).rejects.toThrow('Forbidden')
    })
})
