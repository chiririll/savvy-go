import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { useSettings } from '@/hooks/use-settings'
import { useUser } from '@/stores/auth'
import { pages } from '@/app/pages'

/**
 * Brings an admin back to the server setup while the server says it is pending.
 * The server decides (it is set when the first admin is created), so a reload,
 * a closed tab or another device make no difference.
 */
export function ServerSetupGate({ children }: { children: ReactNode }) {
    const isAdmin = useUser()?.role === 'admin'
    const { data: settings, isPending } = useSettings(isAdmin)

    if (isAdmin && isPending) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }
    if (isAdmin && settings?.onboarding_server_done === false) {
        return <Navigate to={pages.serverSetup.path} replace />
    }
    return <>{children}</>
}
