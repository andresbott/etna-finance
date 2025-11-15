import { apiClient } from '@/lib/api/client'

export interface BackupFile {
    id: string
    filename: string
    size: number
}

export interface BackupListResponse {
    files: BackupFile[]
}

export const GetBackupFiles = async (): Promise<BackupFile[]> => {
    const { data } = await apiClient.get<BackupListResponse>('/backup')
    return data.files
}

export const DeleteBackupFile = async (id: string): Promise<void> => {
    await apiClient.delete(`/backup/${id}`)
}

export const DownloadBackupFile = async (id: string, filename: string): Promise<void> => {
    const response = await apiClient.get(`/backup/${id}`, {
        responseType: 'blob'
    })
    
    // Create a blob URL and trigger download
    const blob = new Blob([response.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
}

export const CreateBackup = async (): Promise<void> => {
    await apiClient.post('/backup')
}

export const RestoreBackup = async (file: File): Promise<void> => {
    const formData = new FormData()
    formData.append('file', file)
    await apiClient.post('/restore', formData, {
        headers: {
            'Content-Type': 'multipart/form-data'
        }
    })
}

export const RestoreBackupFromExisting = async (id: string): Promise<void> => {
    await apiClient.post(`/restore/${id}`)
}

