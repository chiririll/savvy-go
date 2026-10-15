import { useQuery } from '@tanstack/react-query'
import { checksApi } from '@/api/checks'

/** What a server admin should know about; `enabled` keeps other users from asking. */
export function useServerChecks(enabled = true) {
    return useQuery({
        queryKey: ['admin', 'checks'],
        queryFn: checksApi.list,
        enabled,
        staleTime: 30_000,
        refetchInterval: 60_000,
        refetchOnWindowFocus: true,
    })
}
