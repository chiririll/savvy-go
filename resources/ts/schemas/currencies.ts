import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'

export const currencySchema = z.object({
    code: z.string()
        .min(1, i18n.t('validation.currencyCodeRequired'))
        .max(10, i18n.t('validation.maxChars', { count: 10 }))
        .toUpperCase(),

    name: z.string()
        .min(2, i18n.t('validation.minChars', { count: 2 }))
        .max(255, i18n.t('validation.maxChars', { count: 255 })),

    symbol: z.string()
        .min(1, i18n.t('validation.symbolRequired'))
        .max(5, i18n.t('validation.maxChars', { count: 5 })),

    decimals: z.coerce.number()
        .int()
        .min(0, i18n.t('validation.minZero'))
        .max(8, i18n.t('validation.maxEight')),

    rate: z.coerce.number()
        .positive(i18n.t('validation.ratePositive'))
        .optional(),
})

export type CurrencyFormData = z.infer<typeof currencySchema>
