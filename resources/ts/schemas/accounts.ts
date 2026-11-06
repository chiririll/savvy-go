import { z } from 'zod'
import i18n from '@/lib/i18n'
import { FORM_ACCOUNT_TYPES } from '@/types/accounts'

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
})

export type AccountFormValues = z.infer<typeof accountSchema>

export type AccountFormData = Omit<AccountFormValues, 'currency'> & {
    currency_id?: number
    currency_code?: string
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
