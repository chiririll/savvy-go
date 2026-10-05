import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { spacesApi } from '@/api/spaces'
import { useSpaces } from '@/hooks/use-spaces'
import { useUser } from '@/stores/auth'
import { useSpaceStore } from '@/stores/space'

function CenteredLoader() {
    return (
        <div className="min-h-screen flex items-center justify-center">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
    )
}

/** Shown to a user who belongs to no space yet. */
function NoSpace() {
    const { t } = useTranslation('common')
    const user = useUser()
    const queryClient = useQueryClient()
    const create = useMutation({
        mutationFn: () => spacesApi.create(user?.name || t('spaces.defaultName')),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['spaces'] }),
    })
    const isGuest = user?.role === 'guest'

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="max-w-sm space-y-4 text-center">
                <h1 className="text-xl font-semibold">{t('spaces.noneTitle')}</h1>
                <p className="text-muted-foreground">{isGuest ? t('spaces.noneGuest') : t('spaces.noneText')}</p>
                {!isGuest && (
                    <Button onClick={() => create.mutate()} disabled={create.isPending}>
                        {t('spaces.create')}
                    </Button>
                )}
            </div>
        </div>
    )
}

/**
 * Renders the app once the current space is known, so every data request can
 * go to /spaces/{id}. A remembered space the user no longer belongs to falls
 * back to their first one.
 */
export function SpaceGate({ children }: { children: React.ReactNode }) {
    const { data: spaces, isPending } = useSpaces()
    const currentId = useSpaceStore((state) => state.currentId)
    const setCurrentId = useSpaceStore((state) => state.setCurrentId)
    const valid = !!spaces?.some((space) => space.id === currentId)

    useEffect(() => {
        if (spaces && !valid) {
            setCurrentId(spaces[0]?.id ?? null)
        }
    }, [spaces, valid, setCurrentId])

    if (isPending) return <CenteredLoader />
    if (!spaces?.length) return <NoSpace />
    if (!valid) return <CenteredLoader />
    return <>{children}</>
}
