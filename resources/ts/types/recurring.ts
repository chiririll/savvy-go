import { BaseEntity } from './api'
import { Account } from './accounts'
import { Category } from './categories'
import { Tag } from './tags'

import type { TransactionType } from './transactions'

export const RECURRING_FREQUENCIES = ['daily', 'weekly', 'monthly', 'yearly'] as const
export type RecurringFrequency = (typeof RECURRING_FREQUENCIES)[number]
export type RecurringTransactionType = Extract<TransactionType, 'income' | 'expense' | 'transfer'>

export interface RecurringTransaction extends BaseEntity {
    type: RecurringTransactionType
    amount: number
    toAmount: number | null
    isEstimated: boolean
    description: string | null
    frequency: RecurringFrequency
    interval: number
    dayOfWeek: number | null
    dayOfMonth: number | null
    startDate: string
    endDate: string | null
    nextRunDate: string
    lastRunDate: string | null
    isActive: boolean
    account: Account
    toAccount: Account | null
    category: Category | null
    tags: Tag[]
}
