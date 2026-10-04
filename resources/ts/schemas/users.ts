import { z } from 'zod'
import i18n from '@/lib/i18n'

export const roleSchema = z.enum(['admin', 'user', 'guest'])

export const createUserSchema = z.object({
    name: z.string().min(1, i18n.t('validation.nameRequired')).max(255),
    email: z.string().min(1, i18n.t('validation.emailRequired')).email(i18n.t('validation.emailInvalid')),
    setPassword: z.boolean(),
    password: z.string().optional().or(z.literal('')),
    role: roleSchema.default('user'),
}).superRefine((data, ctx) => {
    if (data.setPassword && (!data.password || data.password.length < 8)) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n.t('validation.passwordMin8'),
            path: ['password'],
        })
    }
})

export const updateUserSchema = z.object({
    name: z.string().min(1, i18n.t('validation.nameRequired')).max(255),
    email: z.string().min(1, i18n.t('validation.emailRequired')).email(i18n.t('validation.emailInvalid')),
    password: z.string().min(8, i18n.t('validation.passwordMin8')).optional().or(z.literal('')),
    role: roleSchema.optional(),
})

export type CreateUserFormData = z.infer<typeof createUserSchema>
export type UpdateUserFormData = z.infer<typeof updateUserSchema>
export type UserFormData = {
    name: string
    email: string
    password?: string
    role?: z.infer<typeof roleSchema>
}
