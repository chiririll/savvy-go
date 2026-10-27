import { useQuery } from '@tanstack/react-query'
import { checksApi } from '@/api/checks'

export const SERVER_CHECKS_KEY = ['admin', 'checks']

/**
 * What a server admin should know about; `enabled` keeps other users from asking.
 * Fixes happen on other pages (mail, backups, app URL), so entering a page or
 * returning to the tab must re-read, whatever the global staleTime says.
 */
export function useServerChecks(enabled = true) {
    return useQuery({
        queryKey: SERVER_CHECKS_KEY,
        queryFn: checksApi.list,
        enabled,
        staleTime: 0,
        refetchOnMount: 'always',
        refetchInterval: 60_000,
        refetchOnWindowFocus: 'always',
    })
}
