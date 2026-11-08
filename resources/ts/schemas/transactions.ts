import { z } from 'zod'
import i18n from '@/lib/i18n'
import { isDateInFuture } from '@/lib/dates'
import { isTransferType } from '@/lib/transaction-type'

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

/** The fee of a transfer: spent by the source on top of the amount, in an expense category. */
export const transferFeeSchema = z.object({
    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),
    category_id: z.coerce.number().positive().optional().nullable(),
})

/** One line of plain text: the title of the transaction. */
export const descriptionSchema = z
    .string()
    .max(500)
    .refine((value) => !/[\r\n]/.test(value), i18n.t('validation.singleLine'))

export const transactionObject = z.object({
    /** `transfer_out` is a transfer to a linked space: it goes through the transfers API. */
    type: z.enum(['income', 'expense', 'transfer', 'transfer_out'], {
        error: i18n.t('validation.selectTransactionType'),
    }),

    description: descriptionSchema.optional(),

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

    /** A transfer: the first payment is the source and what it sends. */
    to_account_id: z.coerce.number().positive().optional().nullable(),

    /** What the destination receives; the exchange rate is only what the form works it out with. */
    to_amount: z.coerce.number().positive().optional().nullable(),
    exchange_rate: z.coerce.number().positive().optional().nullable(),

    /** transfer_out only: the linked space `to_account_id` belongs to. */
    to_space_id: z.coerce.number().positive().optional().nullable(),

    fee: transferFeeSchema.optional().nullable(),

    /** The note of the line of the default category: kept so a saved note is not lost by editing. */
    default_note: z.string().optional().nullable(),
})

type TransactionData = z.infer<typeof transactionObject>

export type TransactionIssueOptions = {
    /** A recurring rule's template: what it makes later has no date, and a transfer may leave what it receives open. */
    template?: boolean
}

/** What the fields say together that no field can say by itself. */
export function applyTransactionIssues(
    data: Pick<TransactionData, 'payments' | 'change' | 'lines' | 'default_category_id' | 'to_account_id' | 'to_space_id' | 'to_amount'> & { type: string },
    ctx: z.RefinementCtx,
    options?: TransactionIssueOptions,
) {
    if (isTransferType(data.type)) {
        applyTransferIssues(data, ctx, options)
        return
    }

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
}

export const transactionSchema = transactionObject.superRefine((data, ctx) => applyTransactionIssues(data, ctx))

export { isTransferType }

type TransferFields = {
    type: string
    payments: Array<{ account_id: number }>
    to_account_id?: number | null
    to_space_id?: number | null
    to_amount?: number | null
}

/** What a transfer needs that a field cannot say by itself: a destination, and not the source again. */
function applyTransferIssues(data: TransferFields, ctx: z.RefinementCtx, options?: TransactionIssueOptions) {
    const issue = (message: string, path: string[]) =>
        ctx.addIssue({ code: z.ZodIssueCode.custom, message, path })

    if (data.type === 'transfer_out' && !data.to_space_id) {
        issue(i18n.t('validation.selectSpace'), ['to_space_id'])
    }
    if (!data.to_account_id) {
        issue(i18n.t('validation.transferDestination'), ['to_account_id'])
    } else if (data.type === 'transfer' && Number(data.payments[0]?.account_id) === Number(data.to_account_id)) {
        issue(i18n.t('validation.accountUsedTwice'), ['to_account_id'])
    }
    // The destination receives something; the form fills it in when both accounts share a currency. A template
    // may leave it open: each occurrence takes what the rates of its day make of the amount sent.
    if (!options?.template && !(Number(data.to_amount) > 0)) {
        issue(i18n.t('validation.amountRequired'), ['to_amount'])
    }
}

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
