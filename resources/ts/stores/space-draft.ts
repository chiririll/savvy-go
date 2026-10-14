import { create } from 'zustand'
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware'
import { emptyDraft, type SpaceDraft } from '@/components/features/spaces/create/draft'
import { useAuthStore } from '@/stores/auth'

/** Bump when the shape of SpaceDraft changes: an older draft is dropped. */
export const DRAFT_VERSION = 1
export const DRAFT_KEY = 'savvy-space-draft'

/** One draft per user of this browser. */
const keyFor = () => `${DRAFT_KEY}:${useAuthStore.getState().user?.id ?? 'anonymous'}`

/** Storage that never throws: without it the modal works, just without resume. */
const safeStorage: StateStorage = {
    getItem: (name) => {
        try {
            return localStorage.getItem(name === DRAFT_KEY ? keyFor() : name)
        } catch {
            return null
        }
    },
    setItem: (name, value) => {
        try {
            localStorage.setItem(name === DRAFT_KEY ? keyFor() : name, value)
        } catch {
            // no storage, no resume
        }
    },
    removeItem: (name) => {
        try {
            localStorage.removeItem(name === DRAFT_KEY ? keyFor() : name)
        } catch {
            // no storage, no resume
        }
    },
}

interface DraftState {
    draft: SpaceDraft
    update: (patch: Partial<SpaceDraft>) => void
    /** Forgets the draft, after a successful create. */
    clear: () => void
}

export const useSpaceDraftStore = create<DraftState>()(
    persist(
        (set) => ({
            draft: emptyDraft(),
            update: (patch) => set((state) => ({ draft: { ...state.draft, ...patch } })),
            clear: () => {
                set({ draft: emptyDraft() })
                safeStorage.removeItem(DRAFT_KEY)
            },
        }),
        {
            name: DRAFT_KEY,
            version: DRAFT_VERSION,
            storage: createJSONStorage(() => safeStorage),
            partialize: (state) => ({ draft: state.draft }),
            // A different version drops the draft instead of guessing.
            migrate: () => ({ draft: emptyDraft() }),
            skipHydration: true,
        }
    )
)

/** Loads the draft of the signed-in user; call when the modal opens. */
export async function loadSpaceDraft(): Promise<void> {
    try {
        // Rehydrating storage that holds nothing keeps the previous user's state.
        if (safeStorage.getItem(DRAFT_KEY) !== null) {
            await useSpaceDraftStore.persist.rehydrate()
            return
        }
    } catch {
        // broken storage: start empty
    }
    useSpaceDraftStore.setState({ draft: emptyDraft() })
}
