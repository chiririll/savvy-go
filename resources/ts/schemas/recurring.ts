import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'
import { applyTransactionIssues, transactionObject } from './transactions'

/**
 * A rule is a transaction (the template) and a schedule. The form holds both in one set of values: the
 * template part is the transaction form's own, so the same fields draw it.
 */
export const recurringSchema = transactionObject
    .extend({
        /** A rule makes only these; debts and transfers to other spaces are made elsewhere. */
        type: z.enum(['income', 'expense', 'transfer'], {
            error: i18n.t('validation.selectType'),
        }),

        /** The amount is a guess until the occurrence is confirmed. */
        estimated: z.boolean(),

        frequency: z.enum(['daily', 'weekly', 'monthly', 'yearly'], {
            error: i18n.t('validation.selectFrequency'),
        }),

        interval: z.coerce.number()
            .min(1, i18n.t('validation.intervalMin'))
            .max(365, i18n.t('validation.intervalMax')),

        day_of_week: z.coerce.number().min(0).max(6).nullable().optional(),

        day_of_month: z.coerce.number().min(1).max(31).nullable().optional(),

        start_date: z.string({
            error: i18n.t('validation.selectStartDate'),
        }).min(1, i18n.t('validation.selectStartDate')),

        end_date: z.string().nullable().optional(),

        is_active: z.boolean(),
    })
    .superRefine((data, ctx) => {
        applyTransactionIssues(data, ctx, { template: true })

        // The API refuses a schedule that ends before it starts.
        if (data.end_date && data.start_date && data.end_date < data.start_date) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: i18n.t('validation.endBeforeStart'),
                path: ['end_date'],
            })
        }
    })

export type RecurringFormValues = z.infer<typeof recurringSchema>
