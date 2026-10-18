import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { AppUrlStep } from '@/components/features/onboarding/AppUrlStep'
import { OnboardingLayout } from '@/components/features/onboarding/OnboardingLayout'
import { ServerMailStep } from '@/components/features/onboarding/ServerMailStep'
import { ServerOptions } from '@/components/features/onboarding/ServerOptions'
import { ServerSsoStep } from '@/components/features/onboarding/ServerSsoStep'
import { serverSetupPaths } from '@/components/features/onboarding/server-setup-paths'
import { useInstanceSettings, useUpdateSettings } from '@/hooks'
import { useAuthLoading, useAuthStore, useUser } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'
import { pages } from '@/app/pages'

// The new admin goes on to 2FA; that page sends anyone else home.
const NEXT_PATH = pages.setup2fa.path

function PageLoader() {
    return (
        <div className="min-h-screen flex items-center justify-center bg-background">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
    )
}

/**
 * Flow 2, shown to the admin until it is finished or skipped. Step 1 is required (the
 * public URL). Step 2 is a list of optional things; each opens on its own screen and comes back.
 * Routes: /setup/server, /setup/server/more, /setup/server/mail, /setup/server/sso.
 */
export default function ServerSetupPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const user = useUser()
    const authLoading = useAuthLoading()
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const { data: settings } = useInstanceSettings(Boolean(user))
    const update = useUpdateSettings()
    // Saving the flag makes onboarding_server_done true, which would send us home before navigate runs.
    const leaving = useRef(false)

    // Opened cold (reload, closed tab, another device): the session is not loaded yet.
    useEffect(() => {
        if (!user) void checkAuth()
    }, [user, checkAuth])

    if (!user) {
        return authLoading ? <PageLoader /> : <Navigate to="/login" replace />
    }
    if (user.role !== 'admin') {
        return <Navigate to="/" replace />
    }
    if (!settings) {
        return <PageLoader />
    }
    // Only while the server says it is pending (false): done (true) and never-had-it (null) go home.
    if (settings.onboarding_server_done !== false && !leaving.current) {
        return <Navigate to="/" replace />
    }

    const fail = (error: unknown) => toast.error(getApiErrorMessage(error, t('serverSetup.failed')))

    // The URL is the one required step: save it if it changed, then show the optional ones.
    const submitUrl = async (url: string | null) => {
        if (url !== null) {
            try {
                await update.mutateAsync({ app_url: url })
            } catch (error: unknown) {
                fail(error)
                return
            }
        }
        navigate(serverSetupPaths.options)
    }

    // Finishing and skipping both end the flow: anything left is done in the admin panel.
    const leave = async () => {
        leaving.current = true
        try {
            await update.mutateAsync({ onboarding_server_done: true })
            navigate(NEXT_PATH)
        } catch (error: unknown) {
            leaving.current = false
            fail(error)
        }
    }

    return (
        <Routes>
            <Route
                index
                element={
                    <OnboardingLayout title={t('serverSetup.url.title')} description={t('serverSetup.description')}>
                        <AppUrlStep
                            value={settings.app_url ?? ''}
                            writable={settings.config_writable !== false}
                            saving={update.isPending}
                            onContinue={submitUrl}
                        />
                    </OnboardingLayout>
                }
            />
            <Route
                path="more"
                element={
                    <OnboardingLayout
                        title={t('serverSetup.optionsTitle')}
                        description={t('serverSetup.optionsDescription')}
                    >
                        <ServerOptions busy={update.isPending} onFinish={leave} />
                    </OnboardingLayout>
                }
            />
            <Route
                path="mail"
                element={
                    <OnboardingLayout
                        title={t('serverSetup.mail.title')}
                        description={t('serverSetup.mail.description')}
                        className="max-w-2xl"
                    >
                        <ServerMailStep />
                    </OnboardingLayout>
                }
            />
            <Route
                path="sso"
                element={
                    <OnboardingLayout
                        title={t('serverSetup.sso.title')}
                        description={t('serverSetup.sso.description')}
                        className="max-w-2xl"
                    >
                        <ServerSsoStep />
                    </OnboardingLayout>
                }
            />
            <Route path="*" element={<Navigate to={serverSetupPaths.url} replace />} />
        </Routes>
    )
}
