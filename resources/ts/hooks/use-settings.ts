import { useQuery } from '@tanstack/react-query'
import { settingsApi } from '@/api'
import { Settings } from '@/types'
import { useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['settings']

/** Pass `enabled` false while there is nothing to ask yet (no session). */
export function useSettings(enabled = true) {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: settingsApi.get,
        enabled,
    })
}

/** Instance settings alone, for what runs before the current space is validated. */
export function useInstanceSettings(enabled = true) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'instance'],
        queryFn: settingsApi.getInstance,
        enabled,
    })
}

export function useUpdateSettings() {
    return useResourceMutation({
        mutationFn: (data: Partial<Settings>) => settingsApi.update(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.settings.updated'),
    })
}
