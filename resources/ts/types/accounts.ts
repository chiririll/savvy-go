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

/**
 * The types the account form offers. A card is a user account too, but its form needs a limit and a statement
 * day and goes to its own endpoint; it joins this list with the credit-card screens (F7a).
 */
export const FORM_ACCOUNT_TYPES = ['bank', 'cash', 'crypto'] as const satisfies readonly UserAccountType[]
export type FormAccountType = (typeof FORM_ACCOUNT_TYPES)[number]

export interface Account extends BaseEntity {
    name: string
    type: AccountType
    initialBalance: number
    currentBalance: number
    isActive: boolean
    sortOrder: number
    currency: Currency | null
}
