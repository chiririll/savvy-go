import { z } from 'zod'
import i18n from '@/lib/i18n'

export const roleMappingRuleSchema = z.object({
    claim: z.string().min(1, i18n.t('validation.claimRequired')),
    operator: z.enum(['equals', 'contains', 'one_of']),
    value: z.string().min(1, i18n.t('validation.valueRequired')),
    role: z.enum(['admin', 'user', 'guest']),
})

export const identityProviderSchema = z.object({
    name: z.string().min(1, i18n.t('validation.nameRequired')).max(255),
    slug: z
        .string()
        .min(1, i18n.t('validation.slugRequired'))
        .max(255)
        .regex(/^[a-z0-9-]+$/, i18n.t('validation.slugFormat')),
    preset: z.string().min(1, i18n.t('validation.pickProvider')),
    enabled: z.boolean().default(false),
    fields: z.record(z.string(), z.string()).default({}),
    role_mapping: z.array(roleMappingRuleSchema).default([]),
    default_role: z.enum(['user', 'guest']).default('user'),
    allow_jit: z.boolean().default(true),
    sync_role_on_login: z.boolean().default(false),
    link_by_email: z.boolean().default(true),
})

export type IdentityProviderFormData = z.infer<typeof identityProviderSchema>
export type IdentityProviderFormValues = IdentityProviderFormData
