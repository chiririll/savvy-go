import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'
import { CARD_GRACE_DAYS, CARD_STATEMENT_DAYS, FORM_ACCOUNT_TYPES } from '@/types/accounts'

export const accountSchema = z.object({
    name: z.string()
        .min(1, i18n.t('validation.nameRequired'))
        .max(255, i18n.t('validation.maxChars', { count: 255 })),

    type: z.enum(FORM_ACCOUNT_TYPES, {
        error: i18n.t('validation.selectAccountType'),
    }),

    currency: z.string().min(1, i18n.t('validation.selectCurrency')),

    initial_balance: z.coerce.number()
        .min(0, i18n.t('validation.balanceNegative'))
        .optional()
        .default(0),

    is_active: z.boolean().optional().default(true),

    // Terms of a credit card; only read when the type is credit.
    limit: z.coerce.number().optional(),
    statement_day: z.coerce.number().int().optional(),
    grace_days: z.coerce.number().int().optional(),
}).superRefine((value, ctx) => {
    if (value.type !== 'credit') {
        return
    }
    const fail = (path: string, message: string) => ctx.addIssue({ code: 'custom', path: [path], message })
    if (!value.limit || value.limit <= 0) {
        fail('limit', i18n.t('validation.cardLimit'))
    }
    const day = value.statement_day
    if (!day || day < CARD_STATEMENT_DAYS.min || day > CARD_STATEMENT_DAYS.max) {
        fail('statement_day', i18n.t('validation.cardStatementDay', CARD_STATEMENT_DAYS))
    }
    const grace = value.grace_days ?? 0
    if (grace < CARD_GRACE_DAYS.min || grace > CARD_GRACE_DAYS.max) {
        fail('grace_days', i18n.t('validation.cardGraceDays', CARD_GRACE_DAYS))
    }
})

export type AccountFormValues = z.infer<typeof accountSchema>

export type AccountFormData = Omit<AccountFormValues, 'currency'> & {
    currency_id?: number
    currency_code?: string
    /** Set by the dialog: an existing account that this save turns into a credit card. */
    convert?: boolean
}

/** What the API gets: the card terms only for a card. */
export function toAccountPayload(data: AccountFormValues): AccountFormData {
    const { currency, limit, statement_day, grace_days, ...rest } = data
    const card = data.type === 'credit' ? { limit, statement_day, grace_days: grace_days ?? 0 } : {}
    return { ...rest, ...card, ...decodeAccountCurrency(currency) }
}

export function encodeAccountCurrency(value: { id?: number; code?: string }): string {
    if (value.code) {
        return `code:${value.code}`
    }

    if (value.id) {
        return `id:${value.id}`
    }

    return ''
}

export function decodeAccountCurrency(currency: string): { currency_id?: number; currency_code?: string } {
    if (currency.startsWith('code:')) {
        return { currency_code: currency.slice(5) }
    }

    if (currency.startsWith('id:')) {
        const id = Number(currency.slice(3))
        return Number.isFinite(id) && id > 0 ? { currency_id: id } : {}
    }

    return {}
}
