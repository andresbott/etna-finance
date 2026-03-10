import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    getProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
    getCategoryRules,
    createCategoryRule,
    updateCategoryRule,
    deleteCategoryRule,
    parseCSV,
    submitImport,
    previewCSV,
    reapplyPreview,
    reapplySubmit,
} from './CsvImport'
import type {
    ImportProfile,
    CategoryRule,
    ParsedRow,
    PreviewResult,
    ReapplyRow,
    ReapplySubmitItem,
} from '@/types/csvimport'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockProfile: ImportProfile = {
    id: 1,
    name: 'Bank CSV',
    csvSeparator: ',',
    skipRows: 1,
    dateColumn: 'Date',
    dateFormat: 'YYYY-MM-DD',
    descriptionColumn: 'Description',
    amountColumn: 'Amount',
    amountMode: 'single',
    creditColumn: '',
    debitColumn: '',
}

const mockCategoryRule: CategoryRule = {
    id: 1,
    pattern: 'grocery',
    isRegex: false,
    categoryId: 5,
    position: 0,
}

const mockParsedRow: ParsedRow = {
    rowNumber: 1,
    date: '2026-01-15',
    description: 'Supermarket purchase',
    amount: 42.5,
    type: 'expense',
    categoryId: 5,
    isDuplicate: false,
}

const mockPreviewResult: PreviewResult = {
    headers: ['Date', 'Description', 'Amount'],
    rows: [mockParsedRow],
    totalRows: 1,
    detectedSeparator: ',',
    detectedSkipRows: 1,
    detectedDateFormat: 'YYYY-MM-DD',
    detectedColumns: {
        dateColumn: 'Date',
        descriptionColumn: 'Description',
        amountColumn: 'Amount',
        amountMode: 'single',
    },
}

// --- Profiles ---

describe('getProfiles', () => {
    it('calls GET /import/profiles and returns data', async () => {
        const profiles = [mockProfile];
        (apiClient.get as Mock).mockResolvedValue({ data: profiles })

        const result = await getProfiles()

        expect(apiClient.get).toHaveBeenCalledWith('/import/profiles')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(profiles)
    })

    it('propagates network errors', async () => {
        (apiClient.get as Mock).mockRejectedValue(new Error('Network Error'))

        await expect(getProfiles()).rejects.toThrow('Network Error')
    })
})

describe('createProfile', () => {
    it('calls POST /import/profiles with payload and returns created profile', async () => {
        const { id, ...payload } = mockProfile;
        (apiClient.post as Mock).mockResolvedValue({ data: mockProfile })

        const result = await createProfile(payload)

        expect(apiClient.post).toHaveBeenCalledWith('/import/profiles', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockProfile)
    })
})

describe('updateProfile', () => {
    it('calls PUT /import/profiles/:id with partial payload', async () => {
        const payload: Partial<ImportProfile> = { name: 'Updated CSV' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockProfile, ...payload } })

        const result = await updateProfile(1, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/import/profiles/1', payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ ...mockProfile, ...payload })
    })
})

describe('deleteProfile', () => {
    it('calls DELETE /import/profiles/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({ data: {} })

        await deleteProfile(1)

        expect(apiClient.delete).toHaveBeenCalledWith('/import/profiles/1')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })
})

// --- Category Rules ---

describe('getCategoryRules', () => {
    it('calls GET /import/category-rules and returns data', async () => {
        const rules = [mockCategoryRule];
        (apiClient.get as Mock).mockResolvedValue({ data: rules })

        const result = await getCategoryRules()

        expect(apiClient.get).toHaveBeenCalledWith('/import/category-rules')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(rules)
    })
})

describe('createCategoryRule', () => {
    it('calls POST /import/category-rules with payload and returns created rule', async () => {
        const { id, ...payload } = mockCategoryRule;
        (apiClient.post as Mock).mockResolvedValue({ data: mockCategoryRule })

        const result = await createCategoryRule(payload)

        expect(apiClient.post).toHaveBeenCalledWith('/import/category-rules', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockCategoryRule)
    })
})

describe('updateCategoryRule', () => {
    it('calls PUT /import/category-rules/:id with partial payload', async () => {
        const payload: Partial<CategoryRule> = { pattern: 'supermarket' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockCategoryRule, ...payload } })

        const result = await updateCategoryRule(1, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/import/category-rules/1', payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ ...mockCategoryRule, ...payload })
    })
})

describe('deleteCategoryRule', () => {
    it('calls DELETE /import/category-rules/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({ data: {} })

        await deleteCategoryRule(1)

        expect(apiClient.delete).toHaveBeenCalledWith('/import/category-rules/1')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })
})

// --- parseCSV ---

describe('parseCSV', () => {
    it('sends FormData with file and accountId to POST /import/parse', async () => {
        const file = new File(['col1,col2\na,b'], 'test.csv', { type: 'text/csv' });
        (apiClient.post as Mock).mockResolvedValue({ data: { rows: [mockParsedRow] } })

        const result = await parseCSV(42, file)

        expect(apiClient.post).toHaveBeenCalledTimes(1)
        const [url, formData, config] = (apiClient.post as Mock).mock.calls[0]
        expect(url).toBe('/import/parse')
        expect(formData).toBeInstanceOf(FormData)
        expect(formData.get('file')).toBe(file)
        expect(formData.get('accountId')).toBe('42')
        expect(config).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } })
        expect(result).toEqual({ rows: [mockParsedRow] })
    })
})

// --- submitImport ---

describe('submitImport', () => {
    it('calls POST /import/submit with accountId and rows', async () => {
        const rows = [mockParsedRow];
        (apiClient.post as Mock).mockResolvedValue({ data: { created: 1 } })

        const result = await submitImport(42, rows)

        expect(apiClient.post).toHaveBeenCalledWith('/import/submit', { accountId: 42, rows })
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ created: 1 })
    })
})

// --- previewCSV ---

describe('previewCSV', () => {
    it('sends FormData with file and all config fields to POST /import/preview', async () => {
        const file = new File(['data'], 'test.csv', { type: 'text/csv' });
        (apiClient.post as Mock).mockResolvedValue({ data: mockPreviewResult })

        const config = {
            csvSeparator: ',',
            skipRows: 1,
            dateColumn: 'Date',
            dateFormat: 'YYYY-MM-DD',
            descriptionColumn: 'Description',
            amountMode: 'single',
            amountColumn: 'Amount',
            creditColumn: 'Credit',
            debitColumn: 'Debit',
        }

        const result = await previewCSV(file, config)

        expect(apiClient.post).toHaveBeenCalledTimes(1)
        const [url, formData, reqConfig] = (apiClient.post as Mock).mock.calls[0]
        expect(url).toBe('/import/preview')
        expect(formData).toBeInstanceOf(FormData)
        expect(formData.get('file')).toBe(file)
        expect(formData.get('csvSeparator')).toBe(',')
        expect(formData.get('skipRows')).toBe('1')
        expect(formData.get('dateColumn')).toBe('Date')
        expect(formData.get('dateFormat')).toBe('YYYY-MM-DD')
        expect(formData.get('descriptionColumn')).toBe('Description')
        expect(formData.get('amountMode')).toBe('single')
        expect(formData.get('amountColumn')).toBe('Amount')
        expect(formData.get('creditColumn')).toBe('Credit')
        expect(formData.get('debitColumn')).toBe('Debit')
        expect(reqConfig).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } })
        expect(result).toEqual(mockPreviewResult)
    })

    it('omits config fields that are undefined', async () => {
        const file = new File(['data'], 'test.csv', { type: 'text/csv' });
        (apiClient.post as Mock).mockResolvedValue({ data: mockPreviewResult })

        await previewCSV(file, {})

        const [, formData] = (apiClient.post as Mock).mock.calls[0]
        expect(formData.get('file')).toBe(file)
        expect(formData.get('csvSeparator')).toBeNull()
        expect(formData.get('skipRows')).toBeNull()
        expect(formData.get('dateColumn')).toBeNull()
        expect(formData.get('dateFormat')).toBeNull()
        expect(formData.get('descriptionColumn')).toBeNull()
        expect(formData.get('amountMode')).toBeNull()
        expect(formData.get('amountColumn')).toBeNull()
        expect(formData.get('creditColumn')).toBeNull()
        expect(formData.get('debitColumn')).toBeNull()
    })

    it('includes skipRows when set to 0', async () => {
        const file = new File(['data'], 'test.csv', { type: 'text/csv' });
        (apiClient.post as Mock).mockResolvedValue({ data: mockPreviewResult })

        await previewCSV(file, { skipRows: 0 })

        const [, formData] = (apiClient.post as Mock).mock.calls[0]
        expect(formData.get('skipRows')).toBe('0')
    })

    it('omits empty string config fields', async () => {
        const file = new File(['data'], 'test.csv', { type: 'text/csv' });
        (apiClient.post as Mock).mockResolvedValue({ data: mockPreviewResult })

        await previewCSV(file, {
            csvSeparator: '',
            dateColumn: '',
            amountColumn: '',
        })

        const [, formData] = (apiClient.post as Mock).mock.calls[0]
        // Empty strings are falsy, so these should not be appended
        expect(formData.get('csvSeparator')).toBeNull()
        expect(formData.get('dateColumn')).toBeNull()
        expect(formData.get('amountColumn')).toBeNull()
    })
})

// --- Reapply ---

describe('reapplyPreview', () => {
    it('calls POST /import/reapply-preview and returns rows', async () => {
        const mockReapplyRows: ReapplyRow[] = [{
            transactionId: 1,
            transactionType: 'expense',
            description: 'Supermarket',
            date: '2026-01-15',
            amount: 42.5,
            accountId: 10,
            accountName: 'Checking',
            currentCategoryId: 3,
            currentCategoryName: 'Food',
            newCategoryId: 5,
            newCategoryName: 'Groceries',
            changed: true,
        }];
        (apiClient.post as Mock).mockResolvedValue({ data: mockReapplyRows })

        const result = await reapplyPreview()

        expect(apiClient.post).toHaveBeenCalledWith('/import/reapply-preview')
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockReapplyRows)
    })
})

describe('reapplySubmit', () => {
    it('calls POST /import/reapply-submit with items and returns count', async () => {
        const items: ReapplySubmitItem[] = [{
            transactionId: 1,
            transactionType: 'expense',
            newCategoryId: 5,
        }];
        (apiClient.post as Mock).mockResolvedValue({ data: { updated: 1 } })

        const result = await reapplySubmit(items)

        expect(apiClient.post).toHaveBeenCalledWith('/import/reapply-submit', items)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ updated: 1 })
    })
})
