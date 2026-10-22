// Sample backend data for stories. A builder takes overrides, so a story states only what it is about.
import type { Account, Category, Currency, CurrencyCatalogItem, Tag } from '@/types'

export const currency = (overrides: Partial<Currency> = {}): Currency => ({
    id: 1,
    code: 'USD',
    name: 'US Dollar',
    symbol: '$',
    decimals: 2,
    isBase: true,
    rate: 1,
    ...overrides,
})

export const catalogItem = (overrides: Partial<CurrencyCatalogItem> = {}): CurrencyCatalogItem => ({
    code: 'CHF',
    name: 'Swiss Franc',
    symbol: 'CHF',
    decimals: 2,
    rate: 0.9,
    ...overrides,
})

// Currencies that differ in decimals, which is where money code goes wrong.
export const currencies: Currency[] = [
    currency(),
    currency({ id: 2, code: 'EUR', name: 'Euro', symbol: '€', isBase: false, rate: 0.92 }),
    currency({ id: 3, code: 'JPY', name: 'Japanese Yen', symbol: '¥', decimals: 0, isBase: false, rate: 150 }),
    currency({ id: 4, code: 'KWD', name: 'Kuwaiti Dinar', symbol: 'KD', decimals: 3, isBase: false, rate: 0.31 }),
]

export const catalog: CurrencyCatalogItem[] = [
    catalogItem(),
    catalogItem({ code: 'GBP', name: 'Pound Sterling', symbol: '£', rate: 0.79 }),
    catalogItem({ code: 'RUB', name: 'Russian Ruble', symbol: '₽', rate: 92 }),
]

export const account = (overrides: Partial<Account> = {}): Account => ({
    id: 1,
    name: 'Cash',
    type: 'cash',
    initialBalance: 0,
    currentBalance: 120.5,
    isActive: true,
    sortOrder: 0,
    currency: currencies[0],
    ...overrides,
})

// One account per currency precision, so a balance shows 0, 2 and 3 decimals.
export const accounts: Account[] = [
    account(),
    account({ id: 2, name: 'Euro card', type: 'bank', currentBalance: 980.1, currency: currencies[1] }),
    account({ id: 3, name: 'Yen wallet', currentBalance: 15000, currency: currencies[2] }),
    account({ id: 4, name: 'Kuwait savings', type: 'bank', currentBalance: 12.345, currency: currencies[3] }),
]

export const category = (overrides: Partial<Category> = {}): Category => ({
    id: 1,
    name: 'Groceries',
    type: 'expense',
    icon: '🛒',
    color: '#16a34a',
    isDefault: false,
    transactionsCount: 12,
    totalAmount: null,
    ...overrides,
})

export const categories: Category[] = [
    category(),
    category({ id: 2, name: 'Salary', type: 'income', icon: '💼', color: '#2563eb', transactionsCount: 3 }),
    category({ id: 3, name: 'Transport', icon: '🚌', color: '#d97706', transactionsCount: 30 }),
]

export const tags: Tag[] = [
    { id: 1, name: 'holiday', transactionsCount: 4 },
    { id: 2, name: 'work', transactionsCount: 9 },
    { id: 3, name: 'gifts', transactionsCount: 1 },
]
