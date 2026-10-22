// Sample CSV import data for the stories of this folder.
import type { CsvParseResult, ImportPreviewResult, ImportResult, PreviewTransaction } from '@/types/import'

export const parsed = (overrides: Partial<CsvParseResult> = {}): CsvParseResult => ({
    importId: 'imp1',
    headers: ['Date', 'Amount', 'Payee', 'Category', 'Currency'],
    previewRows: [
        ['2030-01-05', '-42.50', 'Weekly shop', 'Groceries', 'EUR'],
        ['2030-01-06', '-15000', 'Ramen', 'Eating out', 'JPY'],
        ['2030-01-07', '2500.00', 'Salary', 'Salary', 'EUR'],
        ['2030-01-08', '-12.345', 'Fees', 'Bank', 'KWD'],
        ['2030-01-09', '-3.20', 'Coffee', 'Eating out', 'EUR'],
        ['2030-01-10', '-60.00', 'Fuel', 'Transport', 'EUR'],
    ],
    totalRows: 120,
    detectedFormats: { dateFormat: 'ISO', amountFormat: 'US', hasHeader: true, delimiter: ',' },
    suggestedMapping: { date: 0, amount: 1, description: 2, category: 3, currency: 4 },
    ...overrides,
})

export const previewRow = (overrides: Partial<PreviewTransaction> = {}): PreviewTransaction => ({
    row: 2,
    date: '2030-01-05',
    type: 'expense',
    amount: 42.5,
    description: 'Weekly shop',
    category: 'Groceries',
    tags: [],
    status: 'new',
    duplicateOf: null,
    warnings: [],
    error: null,
    ...overrides,
})

export const preview = (overrides: Partial<ImportPreviewResult['summary']> = {}): ImportPreviewResult => ({
    previewTransactions: [
        previewRow(),
        previewRow({ row: 3, amount: 15000, description: 'Ramen', category: 'Eating out', warnings: ['New currency JPY'] }),
        previewRow({ row: 4, type: 'income', amount: 2500, description: 'Salary', category: 'Salary' }),
        previewRow({ row: 5, status: 'duplicate', duplicateOf: 77, description: 'Coffee', amount: 3.2 }),
        previewRow({ row: 6, status: 'error', date: '', amount: 0, description: null, error: 'Bad date "31/02"' }),
    ],
    summary: {
        willCreate: 3,
        willSkip: 1,
        hasErrors: 1,
        totalRows: 5,
        sampled: 5,
        currenciesToCreate: ['JPY'],
        tagsToCreate: [],
        categoriesToCreate: ['Eating out'],
        categories: [
            { name: 'Groceries', type: 'expense', count: 40, matchId: 1 },
            { name: 'Eating out', type: 'expense', count: 1200, matchId: null },
            { name: 'Salary', type: 'income', count: 12, matchId: 2 },
        ],
        ...overrides,
    },
})

export const result = (overrides: Partial<ImportResult> = {}): ImportResult => ({
    created: 117,
    skippedDuplicates: 2,
    errors: [{ row: 6, message: 'Bad date "31/02"' }],
    createdCurrencies: ['JPY'],
    createdTags: ['imported'],
    createdCategories: ['Eating out'],
    ...overrides,
})
