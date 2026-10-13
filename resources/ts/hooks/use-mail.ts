import { useQuery } from '@tanstack/react-query'
import { mailApi } from '@/api/mail'
import { useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['mail']

/** Admin only: pass `enabled` false for anyone else. */
export function useMailStatus(enabled = true) {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: mailApi.status,
        enabled,
    })
}

export function useSendTestMail() {
    return useResourceMutation({
        mutationFn: (to?: string) => mailApi.test(to),
        successMessage: (data) => i18n.t('system.mail.testSent', { ns: 'settings', to: data.to }),
    })
}
