import { ComponentProps, lazy, Suspense, useState } from 'react'

// The create-space flow pulls in zod, react-hook-form and the entities step; keep it out of the entry chunk.
const CreateSpaceModal = lazy(() => import('./CreateSpaceModal').then((m) => ({ default: m.CreateSpaceModal })))

/**
 * CreateSpaceModal, loaded on first open. It stays mounted afterwards so the close animation
 * and its internal state work as they would without the lazy load.
 */
export function LazyCreateSpaceModal(props: ComponentProps<typeof CreateSpaceModal>) {
    const [wasOpened, setWasOpened] = useState(props.open)

    if (props.open && !wasOpened) {
        setWasOpened(true)
    }

    if (!wasOpened) {
        return null
    }

    return (
        <Suspense fallback={null}>
            <CreateSpaceModal {...props} />
        </Suspense>
    )
}
