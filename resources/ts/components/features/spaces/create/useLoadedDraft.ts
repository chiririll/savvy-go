import { useEffect, useState } from 'react'
import { loadSpaceDraft } from '@/stores/space-draft'

/** The stored draft is read once per opening; until then no step can be shown. */
export function useLoadedDraft(open: boolean) {
    const [loaded, setLoaded] = useState(false)

    useEffect(() => {
        if (!open) return
        let cancelled = false
        setLoaded(false)
        void loadSpaceDraft().then(() => !cancelled && setLoaded(true))
        return () => {
            cancelled = true
        }
    }, [open])

    return loaded
}
