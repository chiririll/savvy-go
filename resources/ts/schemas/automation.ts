import { z } from 'zod'
import i18n from '@/lib/i18n/i18n'

const conditionSchema = z.object({
    field: z.string().min(1, i18n.t('validation.fieldRequired')),
    op: z.string().min(1, i18n.t('validation.operatorRequired')),
    value: z.unknown(),
})

const conditionGroupSchema = z.object({
    match: z.enum(['all', 'any']),
    conditions: z.array(conditionSchema).min(1, i18n.t('validation.oneCondition')),
})

const actionSchema = z.object({
    type: z.string().min(1, i18n.t('validation.actionTypeRequired')),
}).passthrough()

export const automationRuleSchema = z.object({
    name: z.string().min(1, i18n.t('validation.nameRequired')).max(255),
    description: z.string().nullable().optional(),
    trigger_type: z.enum([
        'on_transaction_create',
        'on_transaction_update',
    ]),
    priority: z.coerce.number().min(1).max(100).default(50),
    conditions: conditionGroupSchema,
    actions: z.array(actionSchema).min(1, i18n.t('validation.oneAction')),
    is_active: z.boolean().default(true),
    stop_processing: z.boolean().default(false),
})

export type AutomationRuleFormData = z.infer<typeof automationRuleSchema>
