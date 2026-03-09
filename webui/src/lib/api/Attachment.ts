import { apiClient } from '@/lib/api/client'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0

export interface AttachmentMeta {
    id: number
    originalName: string
    mimeType: string
    fileSize: number
}

export const uploadAttachment = async (txId: number, file: File): Promise<AttachmentMeta> => {
    const formData = new FormData()
    formData.append('file', file)
    const { data } = await apiClient.post<AttachmentMeta>(`/fin/entries/${txId}/attachment`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
    })
    return data
}

export const getAttachmentUrl = (txId: number): string => {
    return `${API_BASE_URL}/fin/entries/${txId}/attachment`
}

export const deleteAttachment = async (txId: number): Promise<void> => {
    await apiClient.delete(`/fin/entries/${txId}/attachment`)
}
