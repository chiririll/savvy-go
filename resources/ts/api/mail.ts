import { api } from './client'
import type { MailStatus } from '@/types/mail'

export const mailApi = {
    status: () => api.get<MailStatus>('/admin/mail'),

    /** Checks the connection, then sends a test message to `to` (default: the signed-in admin). */
    test: (to?: string) => api.post<{ to: string }, { to?: string }>('/admin/mail/test', { to }),
}
