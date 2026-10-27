import { z } from 'zod'
import i18n from '@/lib/i18n'
import { isDateInFuture } from '@/lib/dates'
import { isTransferType } from '@/lib/transaction-type'
import {
    ITEM_QTY_DECIMALS,
    quantityDecimalPlaces,
    sumTransactionItems,
} from '@/lib/transaction-items'

export const transactionItemSchema = z.object({
    name: z.string().min(1, i18n.t('validation.nameRequired')).max(255),
    quantity: z.coerce.number()
        .positive(i18n.t('validation.quantityPositive'))
        .refine(
            (value) => quantityDecimalPlaces(value) <= ITEM_QTY_DECIMALS,
            i18n.t('validation.quantityPrecision'),
        ),
    price_per_unit: z.coerce.number().min(0, i18n.t('validation.cannotBeNegative')),
})

export { isTransferType }

/** A row of a payment: an amount and the account it was paid from, or the change went to. */
export const paymentRowSchema = z.object({
    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),
    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),
})

interface PaymentFields {
    date?: string | null
    extra_payments?: unknown[]
    changes?: unknown[]
}

/** What a payment in several parts needs that one field cannot say; each row checks itself. */
function applyPaymentIssues(data: PaymentFields, ctx: z.RefinementCtx) {
    if (!data.extra_payments?.length && !data.changes?.length) {
        return
    }
    // A payment is settled: it moves balances at once.
    if (!data.date) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: i18n.t('validation.dateRequired'), path: ['date'] })
    }
}

export const transactionSchema = z.object({
    type: z.enum(['income', 'expense', 'transfer', 'transfer_out'], {
        error: i18n.t('validation.selectTransactionType'),
    }),

    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),

    to_account_id: z.coerce.number().positive().optional().nullable(),

    /** transfer_out only: the linked space to_account_id belongs to. */
    to_space_id: z.coerce.number().positive().optional().nullable(),

    category_id: z.coerce.number().positive().optional().nullable(),

    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),

    to_amount: z.coerce.number().positive().optional().nullable(),

    is_estimated: z.boolean().optional(),

    exchange_rate: z.coerce.number().positive().optional().nullable(),

    description: z.string().max(500).optional(),

    date: z.preprocess(
        (val) => (val === '' || val === undefined ? null : val),
        z.string().nullable()
    ),

    items: z.array(transactionItemSchema).optional(),

    tag_ids: z.array(z.number()).optional(),

    /**
     * An expense paid in several parts is saved as legs. extra_payments are the
     * accounts that paid besides account_id, which pays amount; changes are the
     * accounts the change went to.
     */
    extra_payments: z.array(paymentRowSchema).optional(),
    changes: z.array(paymentRowSchema).optional(),
}).superRefine((data, ctx) => {
    applyPaymentIssues(data, ctx)

    const isTransfer = isTransferType(data.type)

    // Transfers require to_account_id
    if (isTransfer && !data.to_account_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.transferDestination'),
            path: ['to_account_id'],
        })
    }

    if (data.type === 'transfer_out' && !data.to_space_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.selectSpace'),
            path: ['to_space_id'],
        })
    }

    // Transfers should not have category
    if (isTransfer && data.category_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.transferNoCategory'),
            path: ['category_id'],
        })
    }

    // Income/Expense should have category
    if (!isTransfer && !data.category_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.selectCategory'),
            path: ['category_id'],
        })
    }
})

export type TransactionSchemaOptions = {
    rejectFutureDate?: boolean
    currencyDecimals?: number
}

function applyItemsTotalIssue(
    data: z.infer<typeof transactionSchema>,
    ctx: z.RefinementCtx,
    decimals = 2,
) {
    const items = data.items ?? []
    if (items.length === 0) {
        return
    }

    const itemsTotal = sumTransactionItems(items, decimals)
    const tolerance = 1 / (10 ** (Math.max(decimals, 0) + 3))
    if (itemsTotal > 0 && Math.abs(itemsTotal - data.amount) > tolerance) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.itemsTotalMismatch', {
                itemsTotal: itemsTotal.toFixed(decimals),
                amount: data.amount.toFixed(decimals),
            }),
            path: ['items'],
        })
    }
}

export function getTransactionSchema(options?: TransactionSchemaOptions) {
    return transactionSchema.superRefine((data, ctx) => {
        applyItemsTotalIssue(data, ctx, options?.currencyDecimals ?? 2)

        const settled = options?.rejectFutureDate || Boolean(data.extra_payments?.length || data.changes?.length)
        if (settled && data.date && isDateInFuture(data.date)) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: i18n.t('validation.dateCannotBeFuture'),
                path: ['date'],
            })
        }
    })
}

export type TransactionFormValues = z.infer<typeof transactionSchema>
export type TransactionItemFormValues = z.infer<typeof transactionItemSchema>
export type TransactionFormData = TransactionFormValues
export type TransactionItemFormData = TransactionItemFormValues
