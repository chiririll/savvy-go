import { BaseEntity } from './api'

import type { LineInput, PaymentInput, TransactionInput, TransactionType } from './transactions'

export const RECURRING_FREQUENCIES = ['daily', 'weekly', 'monthly', 'yearly'] as const
export type RecurringFrequency = (typeof RECURRING_FREQUENCIES)[number]
export type RecurringTransactionType = Extract<TransactionType, 'income' | 'expense' | 'transfer'>

/** When a rule is due; the keys are the API's own (snake_case), as the rule keeps them. */
export interface RecurringSchedule {
    frequency: RecurringFrequency
    /** Every n days, weeks, months or years. */
    interval: number
    /** 0 Sunday to 6; weekly only. */
    day_of_week?: number | null
    /** 1 to 31; monthly only. */
    day_of_month?: number | null
    /** The first due date. */
    start: string
    /** The last day a due date may fall on. */
    end?: string | null
}

/**
 * The transaction a rule makes, as the API took it: payments and lines, with ids only. One payment may have no
 * amount (`null`): each occurrence then takes whatever balances the others at that day's rates. A template
 * saved by another client may hold amounts as strings; `lib/recurring-template` reads them as numbers.
 */
export type RecurringTemplate = Omit<TransactionInput, 'type' | 'date' | 'status' | 'payments' | 'lines'> & {
    type: RecurringTransactionType
    payments: PaymentInput[]
    lines: LineInput[]
}

export interface RecurringTransaction extends BaseEntity {
    template: RecurringTemplate
    schedule: RecurringSchedule
    isActive: boolean
    /** Why the rule stopped (an account was deleted); saving it again clears it. */
    error: string | null
    /** The due date of its open occurrence; null when the rule is stopped or its schedule has ended. */
    nextRunDate: string | null
    lastRunDate: string | null
}

/** A rule as sent to create or update it. */
export interface RecurringInput {
    template: RecurringTemplate
    schedule: RecurringSchedule
    is_active: boolean
}
