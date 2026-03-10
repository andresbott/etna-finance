import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    GetBackupFiles,
    DeleteBackupFile,
    DownloadBackupFile,
    CreateBackup,
    RestoreBackup,
    RestoreBackupFromExisting,
    type BackupFile,
} from './Backup'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => {
    vi.clearAllMocks()
    // Reset DOM/window mocks
    vi.restoreAllMocks()
})

const mockBackupFile: BackupFile = {
    id: 'backup-001',
    filename: 'backup-2026-03-10.db',
    size: 1024,
}

const mockBackupFile2: BackupFile = {
    id: 'backup-002',
    filename: 'backup-2026-03-09.db',
    size: 2048,
}

describe('GetBackupFiles', () => {
    it('calls GET /backup and returns files array', async () => {
        const files = [mockBackupFile, mockBackupFile2];
        (apiClient.get as Mock).mockResolvedValue({ data: { files } })

        const result = await GetBackupFiles()

        expect(apiClient.get).toHaveBeenCalledWith('/backup')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(files)
    })

    it('returns empty array when no backup files exist', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { files: [] } })

        const result = await GetBackupFiles()

        expect(result).toEqual([])
    })

    it('propagates API errors', async () => {
        const error = new Error('Network error');
        (apiClient.get as Mock).mockRejectedValue(error)

        await expect(GetBackupFiles()).rejects.toThrow('Network error')
    })
})

describe('DeleteBackupFile', () => {
    it('calls DELETE /backup/:id with the correct id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await DeleteBackupFile('backup-001')

        expect(apiClient.delete).toHaveBeenCalledWith('/backup/backup-001')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await DeleteBackupFile('backup-001')

        expect(result).toBeUndefined()
    })

    it('constructs URL correctly with special characters in id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await DeleteBackupFile('abc-123-def')

        expect(apiClient.delete).toHaveBeenCalledWith('/backup/abc-123-def')
    })

    it('propagates API errors', async () => {
        (apiClient.delete as Mock).mockRejectedValue(new Error('Not found'));

        await expect(DeleteBackupFile('nonexistent')).rejects.toThrow('Not found')
    })
})

describe('DownloadBackupFile', () => {
    let mockCreateObjectURL: Mock
    let mockRevokeObjectURL: Mock
    let mockCreateElement: Mock
    let mockAppendChild: Mock
    let mockRemoveChild: Mock
    let mockClick: Mock
    let mockLink: { href: string; download: string; click: Mock }

    beforeEach(() => {
        mockClick = vi.fn()
        mockLink = { href: '', download: '', click: mockClick }

        mockCreateObjectURL = vi.fn().mockReturnValue('blob:http://localhost/fake-url')
        mockRevokeObjectURL = vi.fn()
        mockCreateElement = vi.spyOn(document, 'createElement').mockReturnValue(mockLink as unknown as HTMLElement)
        mockAppendChild = vi.spyOn(document.body, 'appendChild').mockImplementation((node) => node)
        mockRemoveChild = vi.spyOn(document.body, 'removeChild').mockImplementation((node) => node)

        window.URL.createObjectURL = mockCreateObjectURL
        window.URL.revokeObjectURL = mockRevokeObjectURL
    })

    it('calls GET /backup/:id with responseType blob', async () => {
        const blobData = new Uint8Array([1, 2, 3]);
        (apiClient.get as Mock).mockResolvedValue({ data: blobData })

        await DownloadBackupFile('backup-001', 'backup-2026-03-10.db')

        expect(apiClient.get).toHaveBeenCalledWith('/backup/backup-001', {
            responseType: 'blob',
        })
        expect(apiClient.get).toHaveBeenCalledTimes(1)
    })

    it('creates a blob URL and triggers a download link click', async () => {
        const blobData = new Uint8Array([1, 2, 3]);
        (apiClient.get as Mock).mockResolvedValue({ data: blobData })

        await DownloadBackupFile('backup-001', 'my-backup.db')

        expect(mockCreateObjectURL).toHaveBeenCalledTimes(1)
        expect(mockLink.href).toBe('blob:http://localhost/fake-url')
        expect(mockLink.download).toBe('my-backup.db')
        expect(mockAppendChild).toHaveBeenCalledWith(mockLink)
        expect(mockClick).toHaveBeenCalledTimes(1)
        expect(mockRemoveChild).toHaveBeenCalledWith(mockLink)
        expect(mockRevokeObjectURL).toHaveBeenCalledWith('blob:http://localhost/fake-url')
    })

    it('creates an anchor element for the download', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: new Uint8Array([]) })

        await DownloadBackupFile('backup-001', 'test.db')

        expect(mockCreateElement).toHaveBeenCalledWith('a')
    })

    it('cleans up the blob URL after download', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: new Uint8Array([]) })

        await DownloadBackupFile('backup-001', 'test.db')

        expect(mockRevokeObjectURL).toHaveBeenCalledTimes(1)
    })

    it('constructs URL correctly with the given id', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: new Uint8Array([]) })

        await DownloadBackupFile('xyz-789', 'file.db')

        expect(apiClient.get).toHaveBeenCalledWith('/backup/xyz-789', {
            responseType: 'blob',
        })
    })
})

describe('CreateBackup', () => {
    it('calls POST /backup', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await CreateBackup()

        expect(apiClient.post).toHaveBeenCalledWith('/backup')
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await CreateBackup()

        expect(result).toBeUndefined()
    })

    it('propagates API errors', async () => {
        (apiClient.post as Mock).mockRejectedValue(new Error('Server error'));

        await expect(CreateBackup()).rejects.toThrow('Server error')
    })
})

describe('RestoreBackup', () => {
    it('calls POST /restore with FormData containing the file', async () => {
        const file = new File(['db-content'], 'backup.db', { type: 'application/octet-stream' });
        (apiClient.post as Mock).mockResolvedValue({})

        await RestoreBackup(file)

        expect(apiClient.post).toHaveBeenCalledTimes(1)

        const [url, formData, config] = (apiClient.post as Mock).mock.calls[0]
        expect(url).toBe('/restore')
        expect(formData).toBeInstanceOf(FormData)
        expect(formData.get('file')).toBeInstanceOf(File)
        expect((formData.get('file') as File).name).toBe('backup.db')
    })

    it('sends multipart/form-data Content-Type header', async () => {
        const file = new File(['content'], 'test.db');
        (apiClient.post as Mock).mockResolvedValue({})

        await RestoreBackup(file)

        const [, , config] = (apiClient.post as Mock).mock.calls[0]
        expect(config).toEqual({
            headers: {
                'Content-Type': 'multipart/form-data',
            },
        })
    })

    it('returns void', async () => {
        const file = new File(['content'], 'test.db');
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await RestoreBackup(file)

        expect(result).toBeUndefined()
    })

    it('appends file under the key "file"', async () => {
        const file = new File(['data'], 'my-backup.db');
        (apiClient.post as Mock).mockResolvedValue({})

        await RestoreBackup(file)

        const [, formData] = (apiClient.post as Mock).mock.calls[0]
        expect(formData.has('file')).toBe(true)
    })

    it('propagates API errors', async () => {
        const file = new File(['data'], 'backup.db');
        (apiClient.post as Mock).mockRejectedValue(new Error('Upload failed'));

        await expect(RestoreBackup(file)).rejects.toThrow('Upload failed')
    })
})

describe('RestoreBackupFromExisting', () => {
    it('calls POST /restore/:id with the correct id', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await RestoreBackupFromExisting('backup-001')

        expect(apiClient.post).toHaveBeenCalledWith('/restore/backup-001')
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await RestoreBackupFromExisting('backup-001')

        expect(result).toBeUndefined()
    })

    it('constructs URL correctly with the given id', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await RestoreBackupFromExisting('abc-456')

        expect(apiClient.post).toHaveBeenCalledWith('/restore/abc-456')
    })

    it('propagates API errors', async () => {
        (apiClient.post as Mock).mockRejectedValue(new Error('Restore failed'));

        await expect(RestoreBackupFromExisting('bad-id')).rejects.toThrow('Restore failed')
    })
})
