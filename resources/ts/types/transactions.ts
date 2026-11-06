import { BaseEntity } from './api'
import { Account } from './accounts'
import { Category } from './categories'
import { Tag } from './tags'

export const ALL_TRANSACTION_TYPES = [
    'income',
    'expense',
    'transfer',
    'debt_payment',
    'debt_collection',
    'debt_lend',
    'debt_borrow',
    'transfer_out',
    'transfer_in',
] as const
export type TransactionType = (typeof ALL_TRANSACTION_TYPES)[number]
export const TRANSACTION_STATUSES = ['pending', 'confirmed', 'skipped'] as const
export type TransactionStatus = (typeof TRANSACTION_STATUSES)[number]

/** Why a transaction cannot be changed freely; mirrors `ledger.Reason`. */
export const RESTRICTION_REASONS = ['debt', 'space_transfer', 'transfer_review', 'recurring_pending'] as const
export type RestrictionReason = (typeof RESTRICTION_REASONS)[number]

export interface TransactionRestriction {
    reason: RestrictionReason
    /** The reason in words, in English; the app shows its own translation by `reason`. */
    message: string
}

export interface TransactionActions {
    edit: boolean
    /** Only the description, tags and the categories and notes of lines can change. */
    editDetails: boolean
    duplicate: boolean
    delete: boolean
    confirm: boolean
    skip: boolean
}

/** Money that moved on one account: signed, in the account's currency. Change is a payment of the opposite sign. */
export interface Payment {
    accountId: number
    account: Account | null
    amount: number
}

/** An amount for a category, positive, in the currency `currencyId` names (one of the payment accounts'). */
export interface TransactionLine {
    seq: number
    categoryId: number | null
    category: Category | null
    currencyId: number
    amount: number
    /** Markdown. */
    note: string | null
}

export interface Transaction extends BaseEntity {
    type: TransactionType
    description: string | null
    date: string | null
    status: TransactionStatus
    recurringTransactionId: number | null
    /** What the payments net to; null when they span several currencies. */
    amount: number | null
    /** Payments less lines in the base currency; set only on the answer to a save, when they disagree. */
    difference: number | null
    actions: TransactionActions
    restrictions: TransactionRestriction[]
    payments: Payment[]
    lines: TransactionLine[]
    tags: Tag[]
}

/** A payment as sent to the API. An empty amount is allowed only in a recurring template. */
export interface PaymentInput {
    account_id: number
    amount: number | null
    estimated?: boolean
}

/** A line as sent to the API. A missing `currency_id` means the currency all the payment accounts share. */
export interface LineInput {
    category_id: number | null
    currency_id?: number
    amount: number
    note?: string | null
}

/** A transaction as sent to create or update it; left out, payments, lines and tags stay as they are on update. */
export interface TransactionInput {
    type: TransactionType
    description?: string | null
    date?: string | null
    status?: TransactionStatus
    payments?: PaymentInput[]
    lines?: LineInput[]
    tag_ids?: number[]
}

export interface TransactionFilters {
    type?: TransactionType
    account_id?: number
    category_id?: number
    account_ids?: number[]
    /** Currency codes; a transfer matches if either account is in one. */
    currency?: string[]
    /** Free text: every word must match the description or a number. */
    q?: string
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
