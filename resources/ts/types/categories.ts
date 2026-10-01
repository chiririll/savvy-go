import { BaseEntity } from './api'

export type CategoryType = 'income' | 'expense'

export interface Category extends BaseEntity {
    name: string
    type: CategoryType
    icon: string
    color: string
    isDefault: boolean
    transactionsCount?: number
    totalAmount?: number
}

export interface CategorySummaryResponse {
    data: Category[]
    total: number
    currency: string | null
}
