import { BaseEntity } from './api'
import { Currency } from './currencies'

/** Accounts the app keeps itself (`ledger.isServiceType`, `accounts.is_service`): never paid from, never edited as an account. */
export const SERVICE_ACCOUNT_TYPES = ['debt', 'space'] as const
export type ServiceAccountType = (typeof SERVICE_ACCOUNT_TYPES)[number]

/** Accounts of the user: what a payment can be made from. A credit card is one, with terms of its own. */
export const USER_ACCOUNT_TYPES = ['bank', 'cash', 'crypto', 'credit'] as const
export type UserAccountType = (typeof USER_ACCOUNT_TYPES)[number]

export const ACCOUNT_TYPES = [...USER_ACCOUNT_TYPES, ...SERVICE_ACCOUNT_TYPES] as const
export type AccountType = (typeof ACCOUNT_TYPES)[number]

export const isServiceAccountType = (type: AccountType): type is ServiceAccountType =>
    (SERVICE_ACCOUNT_TYPES as readonly string[]).includes(type)

/** The types the account form offers. A card has terms of its own and goes to its own endpoint. */
export const FORM_ACCOUNT_TYPES = ['bank', 'cash', 'crypto', 'credit'] as const satisfies readonly UserAccountType[]
export type FormAccountType = (typeof FORM_ACCOUNT_TYPES)[number]

/** Levels of `CardStatus`, `ComputeCardState` on the backend. */
export const CARD_STATUS_LEVELS = ['ok', 'warning', 'danger'] as const
export type CardStatusLevel = (typeof CARD_STATUS_LEVELS)[number]

export const CARD_STATUS_REASONS = ['overdue', 'over_limit', 'payment_due'] as const
export type CardStatusReason = (typeof CARD_STATUS_REASONS)[number]

/** The statement day and grace period the backend accepts. */
export const CARD_STATEMENT_DAYS = { min: 1, max: 31 } as const
export const CARD_GRACE_DAYS = { min: 0, max: 60 } as const

export interface CardStatus {
    level: CardStatusLevel
    reason?: CardStatusReason
    /** What the status is about; zero when the level is ok. */
    amount: number
    /** No date for `over_limit`. */
    date?: string
}

export interface CardStatement {
    date: string
    balance: number
    dueDate: string
    paid: number
    amountDue: number
}

/** What a credit card adds to its account: the terms and what they say about today. */
export interface CreditCardInfo {
    limit: number
    cycle: string
    statementDay: number | null
    graceDays: number
    /** Owed now, positive. */
    used: number
    available: number
    periodStart: string
    periodEnd: string
    spent: number
    statement: CardStatement
    status: CardStatus
}

export interface Account extends BaseEntity {
    name: string
    type: AccountType
    initialBalance: number
    currentBalance: number
    isActive: boolean
    sortOrder: number
    currency: Currency | null
    /** Set for an account of type credit. */
    creditCard?: CreditCardInfo
}
