import { BaseEntity } from './api'
import { Account } from './accounts'
import { Category } from './categories'
import { Tag } from './tags'

export const ALL_TRANSACTION_TYPES = ['income', 'expense', 'transfer', 'debt_payment', 'debt_collection', 'debt_lend', 'debt_borrow', 'transfer_out', 'transfer_in'] as const
export type TransactionType = (typeof ALL_TRANSACTION_TYPES)[number]
export const TRANSACTION_STATUSES = ['pending', 'confirmed', 'skipped'] as const
export type TransactionStatus = (typeof TRANSACTION_STATUSES)[number]

export interface TransactionItem {
    id?: number
    name: string
    quantity: number
    pricePerUnit: number
    totalPrice: number
}

export interface TransactionActions {
    edit: boolean
    duplicate: boolean
    delete: boolean
    confirm: boolean
    skip: boolean
}

export interface Transaction extends BaseEntity {
    type: TransactionType
    amount: number
    toAmount: number | null
    isEstimated: boolean
    description: string | null
    date: string | null
    status: TransactionStatus
    recurringTransactionId: number | null
    actions: TransactionActions
    account: Account
    toAccount: Account | null
    category: Category | null
    items: TransactionItem[]
    tags: Tag[]
}

export interface TransactionFilters {
    type?: TransactionType
    account_id?: number
    category_id?: number
    category_ids?: number[]
    tag_ids?: number[]
    start_date?: string
    end_date?: string
    sort_by?: 'date' | 'amount' | 'created_at'
    sort_direction?: 'asc' | 'desc'
    per_page?: number
    page?: number
    status?: TransactionStatus
}

export interface TransactionSummary {
    income: number
    expense: number
    balance: number
    transactionsCount: number
    currency: string | null
}
