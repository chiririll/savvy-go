import { BaseEntity } from './api'

export const CATEGORY_TYPES = ['income', 'expense'] as const
export type CategoryType = (typeof CATEGORY_TYPES)[number]

export interface Category extends BaseEntity {
    name: string
    type: CategoryType
    icon: string | null
    color: string | null
    isDefault: boolean
    transactionsCount: number
    totalAmount: number | null
}

export interface CategorySummaryResponse {
    data: Category[]
    total: number
    currency: string | null
}
