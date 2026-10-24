import { Toaster } from 'sonner'
import { useTheme } from '@/hooks/use-theme'

/** Sonner's toasts do not follow the page theme on their own. */
export function AppToaster() {
    const { theme } = useTheme()
    return <Toaster position="top-right" richColors theme={theme} />
}
