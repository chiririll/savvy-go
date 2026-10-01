export type DateFormat = 'ISO' | 'DD.MM.YYYY' | 'MM/DD/YYYY' | 'DD/MM/YYYY'
export type AmountFormat = 'US' | 'EU'

export interface DetectedFormats {
    dateFormat: DateFormat
    amountFormat: AmountFormat
    hasHeader: boolean
    delimiter: string
}

export interface SuggestedMapping {
    date?: number
    amount?: number
    description?: number
    category?: number
    tags?: number
    currency?: number
    type?: number
}

export interface CsvParseResult {
    importId: string
    headers: string[]
    previewRows: string[][]
    totalRows: number
    detectedFormats: DetectedFormats
    suggestedMapping: SuggestedMapping
}

export interface ColumnMapping {
    date: number
    amount: number
    description: number | null
    type: number | null
    category: number | null
    tags: number | null
    currency: number | null
}

/** What to do with a category name from the file: an existing category id, create a new one, or none. */
export type ImportCategoryChoice = number | 'create' | 'skip'

export type ImportCategoryMap = Record<string, ImportCategoryChoice>

export interface ImportOptions {
    dateFormat: DateFormat
    amountFormat: AmountFormat
    defaultAccountId: number
    defaultType: 'income' | 'expense'
    skipFirstRow: boolean
    createMissingCurrencies: boolean
    createMissingTags: boolean
    createMissingCategories: boolean
    categoryMap?: ImportCategoryMap
}

export interface ImportCategoryCandidate {
    name: string
    type: 'income' | 'expense'
    count: number
    /** Existing category with the same name and type, if any. */
    matchId: number | null
}

export interface PreviewTransaction {
    row: number
    date: string
    type: string
    amount: number
    description: string | null
    category: string | null
    tags: string[]
    status: 'new' | 'duplicate' | 'error'
    duplicateOf: number | null
    warnings: string[]
    error: string | null
}

export interface ImportPreviewSummary {
    willCreate: number
    willSkip: number
    hasErrors: number
    totalRows: number | null
    sampled: number
    currenciesToCreate: string[]
    tagsToCreate: string[]
    categoriesToCreate: string[]
    categories: ImportCategoryCandidate[]
}

export interface ImportPreviewResult {
    previewTransactions: PreviewTransaction[]
    summary: ImportPreviewSummary
}

export interface ImportError {
    row: number
    message: string
}

export interface ImportResult {
    created: number
    skippedDuplicates: number
    errors: ImportError[]
    createdCurrencies: string[]
    createdTags: string[]
    createdCategories: string[]
}

export type ImportStep = 'upload' | 'mapping' | 'preview' | 'result'

export type ImportStatus =
    | 'pending'
    | 'parsing'
    | 'parsed'
    | 'importing'
    | 'completed'
    | 'failed'

export interface ImportState {
    importId: string
    status: ImportStatus
    totalRows: number | null
    processedRows: number
    created: number
    skipped: number
    errors: number
    message: string | null
    parse: CsvParseResult | null
    result: ImportResult | null
}
