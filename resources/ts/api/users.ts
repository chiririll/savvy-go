import { api } from './client'
import { createCrudApi } from './crud'
import type { User } from '@/types/users'
import type { UserFormData } from '@/schemas'

export const usersApi = {
    ...createCrudApi<User, UserFormData>('/users'),

    /** An admin vouching for (or withdrawing) a user's address. */
    setEmailVerified: (id: number | string, emailVerified: boolean) =>
        api.patch<User, { emailVerified: boolean }>(`/users/${id}`, { emailVerified }),

    /** Returns a one-time link as `token`; with `send` the server emails it instead, in the language `locale`. */
    issuePasswordToken: (id: number | string, send = false, locale?: string) =>
        api.post<User, { send?: boolean; locale?: string }>(`/users/${id}/password-token`, send ? { send: true, locale } : {}),
}
