import { useEffect } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore, useAuthLoading, useIsAuthenticated, useSessionExpired } from '@/stores/auth'
import { authApi } from '@/api'
import { Loader2 } from 'lucide-react'
import { SessionExpiredDialog } from './SessionExpiredDialog'
import { SpaceGate } from './SpaceGate'

const FOCUS_DEBOUNCE_MS = 1000

function FullScreenLoader() {
    return (
        <div className="min-h-screen flex items-center justify-center">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
    )
}

function nextWakeMs(expiresAt: string | null, refreshAt: string | null): number | null {
    const times = [refreshAt, expiresAt]
        .filter((value): value is string => Boolean(value))
        .map((value) => new Date(value).getTime())
        .filter((value) => !Number.isNaN(value))
        .sort((a, b) => a - b)

    return times[0] ?? null
}

export function AuthProvider() {
    const location = useLocation()
    const isLoading = useAuthLoading()
    const isAuthenticated = useIsAuthenticated()
    const sessionExpired = useSessionExpired()
    const expiresAt = useAuthStore((state) => state.expiresAt)
    const refreshAt = useAuthStore((state) => state.refreshAt)
    const checkAuth = useAuthStore((state) => state.checkAuth)

    const { data: status, isPending: statusPending } = useQuery({
        queryKey: ['auth', 'status'],
        queryFn: authApi.status,
        staleTime: Infinity,
        retry: false,
    })

    const needsRegistration = status?.needs_registration ?? false

    useEffect(() => {
        if (!statusPending && !needsRegistration) {
            checkAuth()
        }
    }, [statusPending, needsRegistration, checkAuth])

    useEffect(() => {
        if (!isAuthenticated || sessionExpired) {
            return
        }

        const wakeAt = nextWakeMs(expiresAt, refreshAt)
        let timeout: number | undefined
        if (wakeAt !== null) {
            const delay = Math.min(Math.max(0, wakeAt - Date.now()), 2147483647)
            timeout = window.setTimeout(() => {
                void checkAuth()
            }, delay)
        }

        let debounce: number | undefined
        const onVisible = () => {
            if (document.visibilityState !== 'visible') {
                return
            }
            window.clearTimeout(debounce)
            debounce = window.setTimeout(() => {
                void checkAuth()
            }, FOCUS_DEBOUNCE_MS)
        }

        document.addEventListener('visibilitychange', onVisible)
        window.addEventListener('focus', onVisible)

        return () => {
            window.clearTimeout(timeout)
            window.clearTimeout(debounce)
            document.removeEventListener('visibilitychange', onVisible)
            window.removeEventListener('focus', onVisible)
        }
    }, [isAuthenticated, sessionExpired, expiresAt, refreshAt, checkAuth])

    if (statusPending) {
        return <FullScreenLoader />
    }

    if (needsRegistration) {
        return <Navigate to="/setup" state={{ from: location }} replace />
    }

    if (isLoading) {
        return <FullScreenLoader />
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" state={{ from: location }} replace />
    }

    return (
        <>
            <SpaceGate>
                <Outlet />
            </SpaceGate>
            <SessionExpiredDialog />
        </>
    )
}
