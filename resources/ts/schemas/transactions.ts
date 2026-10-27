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

export const PAYMENT_MODES = ['single', 'several', 'change'] as const
export type PaymentMode = (typeof PAYMENT_MODES)[number]

export const extraPaymentSchema = z.object({
    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),
    amount: z.coerce.number({
        error: i18n.t('validation.amountRequired'),
    }).positive(i18n.t('validation.amountPositive')),
})

interface PaymentFields {
    date?: string | null
    amount: number
    payment_mode?: PaymentMode
    extra_payments?: unknown[]
    handed_over?: number | null
    change_account_id?: number | null
    change_amount?: number | null
}

/** What a split payment needs that one field cannot say; the accounts' currencies are checked on submit. */
function applyPaymentIssues(data: PaymentFields, ctx: z.RefinementCtx) {
    const mode = data.payment_mode ?? 'single'
    if (mode === 'single') {
        return
    }
    const issue = (path: string, message: string) =>
        ctx.addIssue({ code: z.ZodIssueCode.custom, message, path: [path] })

    // A payment is settled: it moves balances at once.
    if (!data.date) {
        issue('date', i18n.t('validation.dateRequired'))
    }
    if (mode === 'several' && !data.extra_payments?.length) {
        issue('extra_payments', i18n.t('validation.paymentNeedsTwoAccounts'))
    }
    if (mode === 'change') {
        if (!data.handed_over) {
            issue('handed_over', i18n.t('validation.amountRequired'))
        } else if (data.handed_over <= data.amount) {
            issue('handed_over', i18n.t('validation.handedOverTooLow'))
        }
        if (!data.change_account_id) {
            issue('change_account_id', i18n.t('validation.selectAccount'))
        }
        if (!data.change_amount) {
            issue('change_amount', i18n.t('validation.amountRequired'))
        }
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
     * How an expense was paid: from one account (the default), from several, or
     * with change given back in another currency. The fields below are only
     * read in the last two modes, which save a split payment.
     */
    payment_mode: z.enum(PAYMENT_MODES).optional(),

    /** "several": the accounts that paid besides account_id, which pays amount. */
    extra_payments: z.array(extraPaymentSchema).optional(),

    /** "change": what was handed over (in account_id's currency), and the change that came back. */
    handed_over: z.coerce.number().positive().optional().nullable(),
    change_account_id: z.coerce.number().positive().optional().nullable(),
    change_amount: z.coerce.number().positive().optional().nullable(),
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

        const settled = options?.rejectFutureDate || (data.payment_mode ?? 'single') !== 'single'
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
