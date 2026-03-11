export interface ImportProfile {
  id: number
  name: string
  csvSeparator: string
  skipRows: number
  dateColumn: string
  dateFormat: string
  descriptionColumn: string
  amountColumn: string
  amountMode: 'single' | 'split'
  creditColumn: string
  debitColumn: string
}

export interface CategoryRuleGroup {
  id: number
  name: string
  categoryId: number
  priority: number
  patterns: CategoryRulePattern[]
}

export interface CategoryRulePattern {
  id: number
  pattern: string
  isRegex: boolean
}

export interface DetectedColumns {
  dateColumn?: string
  descriptionColumn?: string
  amountColumn?: string
  amountMode?: 'single' | 'split'
  creditColumn?: string
  debitColumn?: string
}

export interface PreviewResult {
  headers: string[]
  rows: ParsedRow[]
  totalRows: number
  detectedSeparator?: string
  detectedSkipRows?: number
  detectedDateFormat?: string
  detectedColumns?: DetectedColumns
}

export interface ParsedRow {
  rowNumber: number
  date: string
  description: string
  amount: number
  type: 'income' | 'expense'
  categoryId: number
  isDuplicate: boolean
  error?: string
}

export interface ReapplyRow {
  transactionId: number
  transactionType: 'income' | 'expense'
  description: string
  date: string
  amount: number
  accountId: number
  accountName: string
  currentCategoryId: number
  currentCategoryName: string
  newCategoryId: number
  newCategoryName: string
  changed: boolean
}

export interface ReapplySubmitItem {
  transactionId: number
  transactionType: 'income' | 'expense'
  newCategoryId: number
}
