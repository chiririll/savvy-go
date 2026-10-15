import { api } from './client'
import type { MailInput, MailStatus } from '@/types/mail'

export const mailApi = {
    status: () => api.get<MailStatus>('/admin/mail'),

    /** Saves the whole configuration to config.toml and starts using it. */
    update: (input: MailInput) => api.put<MailStatus, MailInput>('/admin/mail', input),

    /** Checks the connection, then sends a test message to `to` (default: the signed-in admin). */
    test: (to?: string) => api.post<{ to: string }, { to?: string }>('/admin/mail/test', { to }),
}
