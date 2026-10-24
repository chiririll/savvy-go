import { lazy, Suspense } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { AppToaster } from '@/components/providers/AppToaster'
import { AbilityProvider } from '@/components/providers/AbilityProvider'
import { ReadOnlyProvider } from '@/components/providers/ReadOnlyProvider'

const ConfirmDialog = lazy(() => import('@/components/providers/ConfirmDialog'))

export function Providers({ children }: { children: React.ReactNode }) {
    return (
        <QueryClientProvider client={queryClient}>
            <AbilityProvider>
                <ReadOnlyProvider>
                    {children}
                </ReadOnlyProvider>
            </AbilityProvider>
            <Suspense fallback={null}>
                <ConfirmDialog />
            </Suspense>
            <AppToaster />
        </QueryClientProvider>
    )
}
