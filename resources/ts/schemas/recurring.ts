import { z } from 'zod'
import i18n from '@/lib/i18n'

export const recurringSchema = z.object({
    type: z.enum(['income', 'expense', 'transfer'], {
        message: i18n.t('validation.selectType'),
    }),

    account_id: z.coerce.number({
        message: i18n.t('validation.selectAccount'),
    }).min(1, i18n.t('validation.selectAccount')),

    to_account_id: z.coerce.number().nullable().optional(),

    category_id: z.coerce.number().nullable().optional(),

    amount: z.coerce.number()
        .min(0.01, i18n.t('validation.amountGreaterThanZero')),

    to_amount: z.coerce.number().nullable().optional(),

    is_estimated: z.boolean().default(false),

    description: z.string().max(255).nullable().optional(),

    frequency: z.enum(['daily', 'weekly', 'monthly', 'yearly'], {
        message: i18n.t('validation.selectFrequency'),
    }),

    interval: z.coerce.number()
        .min(1, i18n.t('validation.intervalMin'))
        .max(365, i18n.t('validation.intervalMax'))
        .default(1),

    day_of_week: z.coerce.number().min(0).max(6).nullable().optional(),

    day_of_month: z.coerce.number().min(1).max(31).nullable().optional(),

    start_date: z.string({
        message: i18n.t('validation.selectStartDate'),
    }),

    end_date: z.string().nullable().optional(),

    is_active: z.boolean().default(true),

    tag_ids: z.array(z.number()).default([]),
}).superRefine((data, ctx) => {
    // Require category for income/expense
    if ((data.type === 'income' || data.type === 'expense') && !data.category_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.selectCategory'),
            path: ['category_id'],
        })
    }

    // Require to_account_id for transfer
    if (data.type === 'transfer' && !data.to_account_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.selectDestination'),
            path: ['to_account_id'],
        })
    }

    // Don't allow same account for transfer
    if (data.type === 'transfer' && data.account_id === data.to_account_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.destinationDifferent'),
            path: ['to_account_id'],
        })
    }
})

export type RecurringFormData = z.infer<typeof recurringSchema>
