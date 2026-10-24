import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { oauthClientsApi, oauthRegistrationApi, type OAuthRegistration } from '@/api/oauth'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'

const QUERY_KEY = ['oauth-registration']

/**
 * These change outside this page (an app registers itself), and interval polling
 * pauses while the tab is hidden. Entering the page or returning to the tab must
 * always read fresh data, whatever the global 5 minute staleTime says.
 */
const LIVE = { staleTime: 0, refetchOnMount: 'always', refetchOnWindowFocus: 'always' } as const

/** While the window is open the state is re-read so the screen notices it closing. */
export function useOAuthRegistration() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: oauthRegistrationApi.get,
        ...LIVE,
        refetchInterval: (query) => (query.state.data?.open ? 5000 : false),
    })
}

export function useSetOAuthRegistration() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: (open: boolean) => (open ? oauthRegistrationApi.open() : oauthRegistrationApi.close()),
        onSuccess: (data: OAuthRegistration) => queryClient.setQueryData(QUERY_KEY, data),
        onError: (error: unknown, open: boolean) => {
            toast.error(
                getApiErrorMessage(
                    error,
                    open ? i18n.t('settings:admin.mcp.openFailed') : i18n.t('settings:admin.mcp.closeFailed'),
                ),
            )
        },
    })
}

const CLIENTS_KEY = ['oauth-clients']

/** New apps show up while the admin has the page open. */
export function useOAuthClients() {
    return useQuery({ queryKey: CLIENTS_KEY, queryFn: oauthClientsApi.list, ...LIVE, refetchInterval: 10000 })
}

export function useRemoveOAuthClient() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: oauthClientsApi.remove,
        onSuccess: () => queryClient.invalidateQueries({ queryKey: CLIENTS_KEY }),
        onError: (error: unknown) =>
            toast.error(getApiErrorMessage(error, i18n.t('settings:admin.mcp.apps.removeFailed'))),
    })
}
