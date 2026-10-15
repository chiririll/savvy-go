import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { LazyCreateSpaceModal } from '@/components/features/spaces/LazyCreateSpaceModal'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
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
export function NoSpace() {
    const { t } = useTranslation('common')
    const user = useUser()
    const isReadOnly = useReadOnly()
    const isGuest = user?.role === 'guest'
    const canCreate = !isGuest && !isReadOnly
    const [creating, setCreating] = useState(canCreate)

    return (
        <div className="min-h-screen flex items-center justify-center p-4">
            <div className="max-w-sm space-y-4 text-center">
                <h1 className="text-xl font-semibold">{t('spaces.noneTitle')}</h1>
                <p className="text-muted-foreground">{isGuest ? t('spaces.noneGuest') : t('spaces.noneText')}</p>
                {canCreate && (
                    <>
                        <Button onClick={() => setCreating(true)}>{t('spaces.create')}</Button>
                        {/* Without a space there is nothing to go back to, so it cannot be closed. */}
                        <LazyCreateSpaceModal open={creating} onOpenChange={setCreating} dismissible={false} />
                    </>
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
