import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UiState {
    sidebarOpen: boolean
    settingsOpen: boolean
    adminOpen: boolean
    setSidebarOpen: (open: boolean) => void
    setSettingsOpen: (open: boolean) => void
    setAdminOpen: (open: boolean) => void
}

export const useUiStore = create<UiState>()(
    persist(
        (set) => ({
            sidebarOpen: true,
            settingsOpen: false,
            adminOpen: false,
            setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
            setSettingsOpen: (settingsOpen) => set({ settingsOpen }),
            setAdminOpen: (adminOpen) => set({ adminOpen }),
        }),
        { name: 'savvy-ui' }
    )
)

export const useSidebarOpen = () => useUiStore((state) => state.sidebarOpen)
export const useSettingsOpen = () => useUiStore((state) => state.settingsOpen)
