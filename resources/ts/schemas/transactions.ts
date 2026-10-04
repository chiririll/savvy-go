import { z } from 'zod'
import i18n from '@/lib/i18n'
import { isDateInFuture } from '@/lib/dates'
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

export const transactionSchema = z.object({
    type: z.enum(['income', 'expense', 'transfer'], {
        error: i18n.t('validation.selectTransactionType'),
    }),

    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),

    to_account_id: z.coerce.number().positive().optional().nullable(),

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
}).superRefine((data, ctx) => {
    // Transfer requires to_account_id
    if (data.type === 'transfer' && !data.to_account_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.transferDestination'),
            path: ['to_account_id'],
        })
    }

    // Transfer should not have category
    if (data.type === 'transfer' && data.category_id) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.transferNoCategory'),
            path: ['category_id'],
        })
    }

    // Income/Expense should have category
    if (data.type !== 'transfer' && !data.category_id) {
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
export type TransactionItemFormValues = z.infer<typeof transactionItemSchema>
export type TransactionFormData = TransactionFormValues
export type TransactionItemFormData = TransactionItemFormValues
