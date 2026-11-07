import { api } from './client'
import { createCrudApi } from './crud'
import { RecurringInput, RecurringTransaction } from '@/types'

export const recurringApi = {
    ...createCrudApi<RecurringTransaction, RecurringInput>('/recurring'),

    getUpcoming: () =>
        api.get<RecurringTransaction[]>('/recurring-upcoming'),
}
