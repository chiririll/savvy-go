import { api } from './client'
import { createCrudApi } from './crud'
import type { User } from '@/types/users'
import type { UserFormData } from '@/schemas'

export const usersApi = {
    ...createCrudApi<User, UserFormData>('/users'),

    /** Returns a one-time link as `token`; with `send` the server emails it instead. */
    issuePasswordToken: (id: number | string, send = false) =>
        api.post<User, { send?: boolean }>(`/users/${id}/password-token`, send ? { send: true } : {}),
}
