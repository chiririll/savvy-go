import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'

export const categorySchema = z.object({
    name: z.string()
        .min(2, i18n.t('validation.minChars', { count: 2 }))
        .max(50, i18n.t('validation.maxChars', { count: 50 })),

    type: z.enum(['income', 'expense'], {
        message: i18n.t('validation.selectCategoryType'),
    }),

    icon: z.string()
        .min(1, i18n.t('validation.selectIcon')),

    color: z.string()
        .regex(/^#[0-9A-Fa-f]{6}$/, i18n.t('validation.invalidColor')),
})

export type CategoryFormData = z.infer<typeof categorySchema>
