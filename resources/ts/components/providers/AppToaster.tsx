import type { CSSProperties } from 'react'
import { Toaster } from 'sonner'
import { useTheme } from '@/hooks/use-theme'

// Sonner's rich colours are fixed pastels that miss WCAG AA, so the toasts take the status tokens instead.
const tint = (token: string) => `color-mix(in oklab, var(--${token}) 10%, var(--background))`
const border = (token: string) => `color-mix(in oklab, var(--${token}) 30%, var(--background))`
const status = (name: string, token: string) => ({
    [`--${name}-bg`]: tint(token),
    [`--${name}-border`]: border(token),
    [`--${name}-text`]: `var(--${token})`,
})

const style = {
    ...status('success', 'success'),
    ...status('error', 'destructive'),
    ...status('warning', 'warning'),
    ...status('info', 'info'),
} as CSSProperties

export function AppToaster() {
    const { theme } = useTheme()
    return <Toaster position="top-right" richColors theme={theme} style={style} />
}
