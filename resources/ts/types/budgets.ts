import { BaseEntity } from './api'
import { Category } from './categories'
import { Currency } from './currencies'
import { Tag } from './tags'

export type BudgetPeriod = 'weekly' | 'monthly' | 'yearly' | 'one_time'

export interface BudgetProgress {
    spent: number
    remaining: number
    percent: number
    periodStart: string
    periodEnd: string
    isExceeded: boolean
}

export interface Budget extends BaseEntity {
    name: string
    amount: number
    currency: Currency | null
    period: BudgetPeriod
    startDate: string | null
    endDate: string | null
    isGlobal: boolean
    notifyAtPercent: number | null
    isActive: boolean
    categories: Category[]
    tags: Tag[]
    progress: BudgetProgress | null
}
