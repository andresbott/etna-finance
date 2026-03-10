import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    uploadAttachment,
    getAttachmentUrl,
    deleteAttachment,
    type AttachmentMeta,
} from './Attachment'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockMeta: AttachmentMeta = {
    id: 1,
    originalName: 'receipt.pdf',
    mimeType: 'application/pdf',
    fileSize: 12345,
}

describe('uploadAttachment', () => {
    it('calls POST /fin/entries/:txId/attachment with FormData and returns metadata', async () => {
        (apiClient.post as Mock).mockResolvedValue({ data: mockMeta })

        const file = new File(['content'], 'receipt.pdf', { type: 'application/pdf' })
        const result = await uploadAttachment(42, file)

        expect(apiClient.post).toHaveBeenCalledTimes(1)
        const [url, formData, config] = (apiClient.post as Mock).mock.calls[0]
        expect(url).toBe('/fin/entries/42/attachment')
        expect(formData).toBeInstanceOf(FormData)
        expect(formData.get('file')).toBe(file)
        expect(config).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } })
        expect(result).toEqual(mockMeta)
    })

    it('propagates errors from apiClient', async () => {
        const error = new Error('upload failed');
        (apiClient.post as Mock).mockRejectedValue(error)

        const file = new File(['x'], 'bad.txt')
        await expect(uploadAttachment(1, file)).rejects.toThrow('upload failed')
    })
})

describe('getAttachmentUrl', () => {
    it('returns the full URL for the given transaction id', () => {
        const url = getAttachmentUrl(99)

        expect(url).toContain('/fin/entries/99/attachment')
    })

    it('includes the API base URL prefix', () => {
        const url = getAttachmentUrl(1)
        expect(url).toContain('/fin/entries/1/attachment')
    })
})

describe('deleteAttachment', () => {
    it('calls DELETE /fin/entries/:txId/attachment', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteAttachment(42)

        expect(apiClient.delete).toHaveBeenCalledWith('/fin/entries/42/attachment')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteAttachment(42)

        expect(result).toBeUndefined()
    })

    it('propagates errors from apiClient', async () => {
        const error = new Error('delete failed');
        (apiClient.delete as Mock).mockRejectedValue(error)

        await expect(deleteAttachment(1)).rejects.toThrow('delete failed')
    })
})
