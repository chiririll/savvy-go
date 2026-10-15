import { create } from 'zustand'
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware'
import { useAuthStore } from '@/stores/auth'

/**
 * Whether `stored` has the shape of `empty`: the same fields, each of the same kind
 * (text, number, flag, list, record). Where `empty` has null the kind is not known
 * and anything goes. Items of a list are not looked into.
 */
export function sameShape(empty: unknown, stored: unknown): boolean {
    if (empty === null) return true
    if (Array.isArray(empty)) return Array.isArray(stored)
    if (typeof empty === 'object') {
        if (typeof stored !== 'object' || stored === null || Array.isArray(stored)) return false
        const keys = Object.keys(empty)
        if (keys.length !== Object.keys(stored).length) return false
        return keys.every((k) => k in stored && sameShape((empty as Record<string, unknown>)[k], (stored as Record<string, unknown>)[k]))
    }
    return typeof empty === typeof stored
}

interface Options<T extends object> {
    /** Storage name; the id of the signed-in user is added to it. */
    key: string
    empty: () => T
    /** For what the shape cannot tell, such as a step that must be one of the known ones. */
    accept?: (stored: T) => boolean
}

export interface DraftState<T extends object> {
    draft: T
    update: (patch: Partial<T>) => void
    /** Forgets the draft, after what it was for is done. */
    clear: () => void
}

/**
 * A draft that lives in this browser only: saved on every change, one per user,
 * and checked against the code when it is loaded (a draft of another shape is
 * dropped). It works without storage, just without resume. Nothing goes to the server.
 */
export function createDraftStore<T extends object>({ key, empty, accept }: Options<T>) {
    const keyFor = () => `${key}:${useAuthStore.getState().user?.id ?? 'anonymous'}`

    const read = (name: string): string | null => {
        try {
            return localStorage.getItem(name === key ? keyFor() : name)
        } catch {
            return null
        }
    }

    // Storage that never throws.
    const storage: StateStorage = {
        getItem: read,
        setItem: (name, value) => {
            try {
                localStorage.setItem(name === key ? keyFor() : name, value)
            } catch {
                // no storage, no resume
            }
        },
        removeItem: (name) => {
            try {
                localStorage.removeItem(name === key ? keyFor() : name)
            } catch {
                // no storage, no resume
            }
        },
    }

    const useStore = create<DraftState<T>>()(
        persist(
            (set) => ({
                draft: empty(),
                update: (patch) => set((state) => ({ draft: { ...state.draft, ...patch } })),
                clear: () => {
                    set({ draft: empty() })
                    storage.removeItem(key)
                },
            }),
            {
                name: key,
                storage: createJSONStorage(() => storage),
                partialize: (state) => ({ draft: state.draft }),
                skipHydration: true,
            }
        )
    )

    /** Loads the draft of the signed-in user; call when the screen that edits it opens. */
    const load = async (): Promise<void> => {
        try {
            const raw = read(key)
            // Setting only a draft that fits keeps the previous user's state from showing through.
            const stored = raw === null ? null : (JSON.parse(raw) as { state?: { draft?: unknown } }).state?.draft
            if (raw !== null && sameShape(empty(), stored) && (accept?.(stored as T) ?? true)) {
                useStore.setState({ draft: stored as T })
                return
            }
        } catch {
            // broken storage: start empty
        }
        useStore.setState({ draft: empty() })
    }

    return { useStore, load, key }
}
