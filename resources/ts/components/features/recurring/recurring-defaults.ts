import { formatDateLocal } from '@/lib/utils'
import type { RecurringFormValues } from '@/schemas/recurring'
import { toFormDefaults } from '@/components/features/transactions/form/use-transaction-form'

/** What an opener can ask for: any of the values of the form. */
export type RecurringFormDefaults = Partial<RecurringFormValues>

/** The values a new rule starts with: a monthly expense that begins today, active. The template has no date. */
export function toRecurringDefaults(defaults?: RecurringFormDefaults): RecurringFormValues {
    const today = new Date()
    return {
        ...toFormDefaults(defaults as Parameters<typeof toFormDefaults>[0]),
        type: defaults?.type ?? 'expense',
        date: null,
        estimated: defaults?.estimated ?? false,
        frequency: defaults?.frequency ?? 'monthly',
        interval: defaults?.interval ?? 1,
        day_of_week: defaults?.day_of_week ?? today.getDay(),
        day_of_month: defaults?.day_of_month ?? today.getDate(),
        start_date: defaults?.start_date ?? formatDateLocal(),
        end_date: defaults?.end_date ?? null,
        is_active: defaults?.is_active ?? true,
    }
}
