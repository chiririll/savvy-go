import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface SpaceState {
    /** The space data requests go to; remembered per browser. */
    currentId: number | null
    setCurrentId: (id: number | null) => void
    /** The user the remembered space belongs to; another user starts clean. */
    ownerId: number | null
    setOwnerId: (id: number | null) => void
}

export const useSpaceStore = create<SpaceState>()(
    persist(
        (set) => ({
            currentId: null,
            setCurrentId: (currentId) => set({ currentId }),
            ownerId: null,
            setOwnerId: (ownerId) => set({ ownerId }),
        }),
        { name: 'savvy-space' }
    )
)

export const useCurrentSpaceId = () => useSpaceStore((state) => state.currentId)

/** The current space id outside React (the API client). */
export const currentSpaceId = () => useSpaceStore.getState().currentId
