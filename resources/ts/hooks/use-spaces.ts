import { useQuery } from '@tanstack/react-query'
import { spacesApi } from '@/api/spaces'
import { useUser } from '@/stores/auth'
import { useCurrentSpaceId } from '@/stores/space'

export function useSpaces() {
    const user = useUser()
    return useQuery({
        queryKey: ['spaces', user?.id],
        queryFn: spacesApi.list,
        enabled: !!user,
    })
}

/** The space data requests go to. */
export function useCurrentSpace() {
    const id = useCurrentSpaceId()
    return useSpaces().data?.find((space) => space.id === id) ?? null
}
