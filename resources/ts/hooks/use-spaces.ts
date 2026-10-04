import { useQuery } from '@tanstack/react-query'
import { spacesApi } from '@/api/spaces'
import { useUser } from '@/stores/auth'

export function useSpaces() {
    const user = useUser()
    return useQuery({
        queryKey: ['spaces', user?.id],
        queryFn: spacesApi.list,
        enabled: !!user,
    })
}

/** The space data requests work in: the server uses the caller's first space. */
export function useCurrentSpace() {
    return useSpaces().data?.[0] ?? null
}
