import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UiState {
    sidebarOpen: boolean
    spaceOpen: boolean
    accountOpen: boolean
    adminOpen: boolean
    setSidebarOpen: (open: boolean) => void
    setSpaceOpen: (open: boolean) => void
    setAccountOpen: (open: boolean) => void
    setAdminOpen: (open: boolean) => void
}

export const useUiStore = create<UiState>()(
    persist(
        (set) => ({
            sidebarOpen: true,
            spaceOpen: false,
            accountOpen: false,
            adminOpen: false,
            setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
            setSpaceOpen: (spaceOpen) => set({ spaceOpen }),
            setAccountOpen: (accountOpen) => set({ accountOpen }),
            setAdminOpen: (adminOpen) => set({ adminOpen }),
        }),
        { name: 'savvy-ui' }
    )
)

export const useSidebarOpen = () => useUiStore((state) => state.sidebarOpen)
