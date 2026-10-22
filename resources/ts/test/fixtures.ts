// Sample backend data for stories. A builder takes overrides, so a story states only what it is about.
import type {
    Account,
    Budget,
    Category,
    Currency,
    CurrencyCatalogItem,
    Debt,
    RecurringTransaction,
    Settings,
    Tag,
    Transaction,
} from '@/types'

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

// Currencies that differ in decimals, which is where money code goes wrong. A rate is what one unit is worth in the
// base currency: 1 EUR is 1.25 USD, 1 KWD is 4 USD, and 1 JPY is 0.005 USD (so 1 USD buys 200 JPY).
export const currencies: Currency[] = [
    currency(),
    currency({ id: 2, code: 'EUR', name: 'Euro', symbol: '€', isBase: false, rate: 1.25 }),
    currency({ id: 3, code: 'JPY', name: 'Japanese Yen', symbol: '¥', decimals: 0, isBase: false, rate: 0.005 }),
    currency({ id: 4, code: 'KWD', name: 'Kuwaiti Dinar', symbol: 'KD', decimals: 3, isBase: false, rate: 4 }),
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

export const settings: Settings = {
    auto_update_currencies: false,
    sso_allow_signup: false,
    password_login_enabled: true,
    sso_require_verified_email: false,
    require_email_verification: false,
    space_quota_mb: 0,
    space_backups_max: 0,
    max_spaces_per_user: null,
    space_invites_can_register: false,
    app_url: 'https://savvy.example.com',
    config_writable: true,
    onboarding_server_done: true,
}

export const budget = (overrides: Partial<Budget> = {}): Budget => ({
    id: 1,
    name: 'Groceries',
    amount: 300,
    currency: currencies[0],
    period: 'monthly',
    startDate: null,
    endDate: null,
    isGlobal: false,
    notifyAtPercent: 80,
    isActive: true,
    categories: [categories[0]],
    tags: [],
    progress: {
        spent: 120,
        remaining: 180,
        percent: 40,
        periodStart: '2026-10-01',
        periodEnd: '2026-10-31',
        isExceeded: false,
    },
    ...overrides,
})

export const debt = (overrides: Partial<Debt> = {}): Debt => ({
    ...account({ id: 20, name: 'Loan from Alex', type: 'debt', currentBalance: 400 }),
    type: 'debt',
    debtType: 'i_owe',
    targetAmount: 500,
    paymentProgress: 20,
    dueDate: '2099-12-31',
    counterparty: 'Alex',
    description: null,
    isPaidOff: false,
    ...overrides,
})

export const recurring = (overrides: Partial<RecurringTransaction> = {}): RecurringTransaction => ({
    id: 1,
    type: 'expense',
    amount: 1100,
    toAmount: null,
    isEstimated: false,
    description: 'Rent',
    frequency: 'monthly',
    interval: 1,
    dayOfWeek: null,
    dayOfMonth: 1,
    startDate: '2026-01-01',
    endDate: null,
    nextRunDate: '2099-01-01',
    lastRunDate: null,
    isActive: true,
    account: accounts[0],
    toAccount: null,
    category: categories[0],
    tags: [],
    ...overrides,
})

export const transaction = (overrides: Partial<Transaction> = {}): Transaction => ({
    id: 1,
    type: 'expense',
    amount: 42.5,
    toAmount: null,
    isEstimated: false,
    description: 'Weekly shop',
    date: '2020-05-04',
    status: 'confirmed',
    recurringTransactionId: null,
    actions: { edit: true, duplicate: true, delete: true, confirm: false, skip: false },
    account: accounts[0],
    toAccount: null,
    category: categories[0],
    items: [],
    tags: [],
    ...overrides,
})

/** Waiting for its day: it can be confirmed or skipped. */
export const pendingTransaction = (overrides: Partial<Transaction> = {}): Transaction =>
    transaction({
        status: 'pending',
        date: '2099-06-01',
        actions: { edit: true, duplicate: false, delete: true, confirm: true, skip: true },
        ...overrides,
    })
