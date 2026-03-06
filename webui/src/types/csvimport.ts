export interface ImportProfile {
  id: number
  name: string
  csvSeparator: string
  skipRows: number
  dateColumn: string
  dateFormat: string
  descriptionColumn: string
  amountColumn: string
}

export interface CategoryRule {
  id: number
  pattern: string
  isRegex: boolean
  categoryId: number
  position: number
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
