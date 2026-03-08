import { apiClient } from './client'
import type { ImportProfile, CategoryRule, ParsedRow, PreviewResult, ReapplyRow, ReapplySubmitItem } from '@/types/csvimport'

// Profiles
export const getProfiles = () => apiClient.get<ImportProfile[]>('/import/profiles').then(r => r.data)
export const createProfile = (p: Omit<ImportProfile, 'id'>) => apiClient.post<ImportProfile>('/import/profiles', p).then(r => r.data)
export const updateProfile = (id: number, p: Partial<ImportProfile>) => apiClient.put(`/import/profiles/${id}`, p).then(r => r.data)
export const deleteProfile = (id: number) => apiClient.delete(`/import/profiles/${id}`).then(r => r.data)

// Category Rules
export const getCategoryRules = () => apiClient.get<CategoryRule[]>('/import/category-rules').then(r => r.data)
export const createCategoryRule = (r: Omit<CategoryRule, 'id'>) => apiClient.post<CategoryRule>('/import/category-rules', r).then(res => res.data)
export const updateCategoryRule = (id: number, r: Partial<CategoryRule>) => apiClient.put(`/import/category-rules/${id}`, r).then(res => res.data)
export const deleteCategoryRule = (id: number) => apiClient.delete(`/import/category-rules/${id}`).then(r => r.data)

// Import
export const parseCSV = (accountId: number, file: File) => {
  const form = new FormData()
  form.append('file', file)
  form.append('accountId', String(accountId))
  return apiClient.post<{ rows: ParsedRow[] }>('/import/parse', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then(r => r.data)
}

export const submitImport = (accountId: number, rows: ParsedRow[]) =>
  apiClient.post<{ created: number }>('/import/submit', { accountId, rows }).then(r => r.data)

export const previewCSV = (file: File, config: {
  csvSeparator?: string
  skipRows?: number
  dateColumn?: string
  dateFormat?: string
  descriptionColumn?: string
  amountMode?: string
  amountColumn?: string
  creditColumn?: string
  debitColumn?: string
}) => {
  const form = new FormData()
  form.append('file', file)
  if (config.csvSeparator) form.append('csvSeparator', config.csvSeparator)
  if (config.skipRows !== undefined) form.append('skipRows', String(config.skipRows))
  if (config.dateColumn) form.append('dateColumn', config.dateColumn)
  if (config.dateFormat) form.append('dateFormat', config.dateFormat)
  if (config.descriptionColumn) form.append('descriptionColumn', config.descriptionColumn)
  if (config.amountMode) form.append('amountMode', config.amountMode)
  if (config.amountColumn) form.append('amountColumn', config.amountColumn)
  if (config.creditColumn) form.append('creditColumn', config.creditColumn)
  if (config.debitColumn) form.append('debitColumn', config.debitColumn)
  return apiClient.post<PreviewResult>('/import/preview', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then(r => r.data)
}

// Reapply category rules
export const reapplyPreview = () =>
  apiClient.post<ReapplyRow[]>('/import/reapply-preview').then(r => r.data)

export const reapplySubmit = (items: ReapplySubmitItem[]) =>
  apiClient.post<{ updated: number }>('/import/reapply-submit', items).then(r => r.data)
