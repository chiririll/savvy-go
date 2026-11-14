import { z } from 'zod'
import i18n from '@/lib/i18n'

export const tagSchema = z.object({
    name: z.string()
        .trim()
        .min(1, i18n.t('validation.nameRequired'))
        .max(50, i18n.t('validation.maxChars', { count: 50 }))
        .regex(/^[a-zA-Zа-яА-ЯёЁ0-9_-]+$/, i18n.t('validation.tagFormat')),
})

export type TagFormData = z.infer<typeof tagSchema>
