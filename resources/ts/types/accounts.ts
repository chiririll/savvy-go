import { BaseEntity } from './api'
import { Currency } from './currencies'

// All account types including debt
export const ACCOUNT_TYPES = ['bank', 'crypto', 'cash', 'debt'] as const
export type AccountType = (typeof ACCOUNT_TYPES)[number]

// Regular account types (excluding debt) - for account creation
export type RegularAccountType = 'bank' | 'crypto' | 'cash'

export interface Account extends BaseEntity {
    name: string
    type: AccountType
    initialBalance: number
    currentBalance: number
    isActive: boolean
    sortOrder: number
    currency: Currency | null
}
