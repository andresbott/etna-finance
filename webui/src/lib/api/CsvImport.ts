import { apiClient } from './client'
import type { ImportProfile, CategoryRuleGroup, CategoryRulePattern, ParsedRow, PreviewResult, ReapplyRow, ReapplySubmitItem } from '@/types/csvimport'

// Profiles
export const getProfiles = () => apiClient.get<ImportProfile[]>('/import/profiles').then(r => r.data)
export const createProfile = (p: Omit<ImportProfile, 'id'>) => apiClient.post<ImportProfile>('/import/profiles', p).then(r => r.data)
export const updateProfile = (id: number, p: Partial<ImportProfile>) => apiClient.put(`/import/profiles/${id}`, p).then(r => r.data)
export const deleteProfile = (id: number) => apiClient.delete(`/import/profiles/${id}`).then(r => r.data)

// Category Rule Groups
export const getCategoryRuleGroups = () => apiClient.get<CategoryRuleGroup[]>('/import/category-rule-groups').then(r => r.data)
export const createCategoryRuleGroup = (g: Omit<CategoryRuleGroup, 'id'>) => apiClient.post<CategoryRuleGroup>('/import/category-rule-groups', g).then(r => r.data)
export const updateCategoryRuleGroup = (id: number, g: Partial<CategoryRuleGroup>) => apiClient.put(`/import/category-rule-groups/${id}`, g).then(r => r.data)
export const deleteCategoryRuleGroup = (id: number) => apiClient.delete(`/import/category-rule-groups/${id}`).then(r => r.data)

// Category Rule Patterns
export const createCategoryRulePattern = (groupId: number, p: Omit<CategoryRulePattern, 'id'>) => apiClient.post<CategoryRulePattern>(`/import/category-rule-groups/${groupId}/patterns`, p).then(r => r.data)
export const updateCategoryRulePattern = (groupId: number, patternId: number, p: Partial<CategoryRulePattern>) => apiClient.put(`/import/category-rule-groups/${groupId}/patterns/${patternId}`, p).then(r => r.data)
export const deleteCategoryRulePattern = (groupId: number, patternId: number) => apiClient.delete(`/import/category-rule-groups/${groupId}/patterns/${patternId}`).then(r => r.data)

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
