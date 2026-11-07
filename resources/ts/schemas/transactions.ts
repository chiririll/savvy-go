import { z } from 'zod'
import i18n from '@/lib/i18n'
import { isDateInFuture } from '@/lib/dates'

/** A payment of the form: the account and a positive amount; the type of the transaction gives it its sign. */
export const paymentRowSchema = z.object({
    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),
    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),
})

/** A line the user wrote: what part of the payment was for which category. */
export const lineRowSchema = z.object({
    category_id: z.coerce.number().positive().optional().nullable(),
    currency_id: z.coerce.number().positive().optional().nullable(),
    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),
    /** Markdown. */
    note: z.string().optional().nullable(),
})

export const transactionSchema = z.object({
    // Transfers come back with their own form (F4d).
    type: z.enum(['income', 'expense'], {
        error: i18n.t('validation.selectTransactionType'),
    }),

    /** One line of plain text: the title of the transaction. */
    description: z.string().max(500).refine((value) => !/[\r\n]/.test(value), i18n.t('validation.singleLine')).optional(),

    date: z.preprocess(
        (val) => (val === '' || val === undefined ? null : val),
        z.string().nullable()
    ),

    tag_ids: z.array(z.number()).optional(),

    /** The accounts that paid, or received for an income. An account appears once in the transaction. */
    payments: z.array(paymentRowSchema).min(1, i18n.t('validation.selectAccount')),

    /** Money that came back to other accounts. */
    change: z.array(paymentRowSchema).optional(),

    /** Lines the user wrote. What they leave uncovered is saved in the default category. */
    lines: z.array(lineRowSchema).optional(),

    default_category_id: z.coerce.number().positive().optional().nullable(),

    /** The note of the line of the default category: kept so a saved note is not lost by editing. */
    default_note: z.string().optional().nullable(),
}).superRefine((data, ctx) => {
    // The same account in two rows: the API refuses it. The second one gets the message.
    const seen = new Set<number>()
    const rows = [
        ...data.payments.map((row, index) => ({ row, path: ['payments', index, 'account_id'] })),
        ...(data.change ?? []).map((row, index) => ({ row, path: ['change', index, 'account_id'] })),
    ]
    for (const { row, path } of rows) {
        const id = Number(row.account_id)
        if (!(id > 0)) {
            continue
        }
        if (seen.has(id)) {
            ctx.addIssue({ code: z.ZodIssueCode.custom, message: i18n.t('validation.accountUsedTwice'), path })
        }
        seen.add(id)
    }

    // Without lines the default category is the category of the transaction.
    if (!data.lines?.length && !data.default_category_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.selectCategory'),
            path: ['default_category_id'],
        })
    }
})

export type TransactionSchemaOptions = {
    /** The transaction already moved balances, so it cannot be dated in the future. */
    rejectFutureDate?: boolean
}

export function getTransactionSchema(options?: TransactionSchemaOptions) {
    return transactionSchema.superRefine((data, ctx) => {
        if (options?.rejectFutureDate && data.date && isDateInFuture(data.date)) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: i18n.t('validation.dateCannotBeFuture'),
                path: ['date'],
            })
        }
    })
}

export type TransactionFormValues = z.infer<typeof transactionSchema>
export type TransactionFormData = TransactionFormValues
