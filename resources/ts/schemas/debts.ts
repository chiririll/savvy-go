import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'
import { isDateInFuture } from '@/lib/format/dates'

const nameField = z.string()
    .min(1, i18n.t('validation.nameRequired'))
    .max(255, i18n.t('validation.maxChars', { count: 255 }))

const debtTypeField = z.enum(['i_owe', 'owed_to_me'], {
    error: i18n.t('validation.selectDebtType'),
})

const amountField = z.coerce.number()
    .positive(i18n.t('validation.amountGreaterThanZero'))

const optionalNotes = {
    due_date: z.string().optional(),
    counterparty: z.string().max(255).optional(),
    description: z.string().max(1000).optional(),
}

export const debtSchema = z.object({
    name: nameField,
    debt_type: debtTypeField,
    currency_id: z.coerce.number({
        error: i18n.t('validation.selectCurrency'),
    }).positive(i18n.t('validation.selectCurrency')),
    amount: amountField,
    ...optionalNotes,
})

export const createDebtSchema = z.object({
    origin: z.enum(['new', 'existing']),
    name: nameField,
    debt_type: debtTypeField,
    amount: amountField,
    account_id: z.coerce.number().optional(),
    date: z.string().optional(),
    currency_id: z.coerce.number().optional(),
    ...optionalNotes,
}).superRefine((data, ctx) => {
    if (data.origin === 'new') {
        if (!data.account_id || data.account_id <= 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['account_id'],
                message: i18n.t('validation.selectAccount'),
            })
        }
        if (!data.date) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['date'],
                message: i18n.t('validation.dateRequired'),
            })
        }
        return
    }

    if (!data.currency_id || data.currency_id <= 0) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['currency_id'],
            message: i18n.t('validation.selectCurrency'),
        })
    }
})

export type DebtFormData = z.infer<typeof createDebtSchema>
export type DebtEditFormData = z.infer<typeof debtSchema>

export function getDebtSchema(mode: 'create' | 'edit') {
    return mode === 'create' ? createDebtSchema : debtSchema
}

export const debtPaymentSchema = z.object({
    account_id: z.coerce.number({
        error: i18n.t('validation.selectAccount'),
    }).positive(i18n.t('validation.selectAccount')),

    amount: z.coerce.number()
        .positive(i18n.t('validation.amountGreaterThanZero')),

    date: z.string().min(1, i18n.t('validation.dateRequired')),

    description: z.string().max(1000).optional(),
}).superRefine((data, ctx) => {
    if (isDateInFuture(data.date)) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.dateCannotBeFuture'),
            path: ['date'],
        })
    }
})

export type DebtPaymentFormData = z.infer<typeof debtPaymentSchema>
