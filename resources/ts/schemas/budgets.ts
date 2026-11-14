import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'

export const budgetSchema = z.object({
    name: z.string()
        .min(2, i18n.t('validation.minChars', { count: 2 }))
        .max(100, i18n.t('validation.maxChars', { count: 100 })),

    amount: z.coerce.number()
        .min(0.01, i18n.t('validation.amountGreaterThanZero')),

    currency_id: z.coerce.number().nullable().optional(),

    period: z.enum(['weekly', 'monthly', 'yearly', 'one_time'], {
        message: i18n.t('validation.selectPeriod'),
    }),

    start_date: z.string().nullable().optional(),
    end_date: z.string().nullable().optional(),

    is_global: z.boolean().default(false),

    notify_at_percent: z.coerce.number()
        .min(1).max(100)
        .nullable()
        .optional(),

    is_active: z.boolean().default(true),

    category_ids: z.array(z.number()).default([]),

    tag_ids: z.array(z.number()).default([]),
})

export type BudgetFormData = z.infer<typeof budgetSchema>
