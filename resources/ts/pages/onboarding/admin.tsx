import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { BootstrapInstructions } from '@/components/features/onboarding/BootstrapInstructions'
import { OnboardingLayout } from '@/components/features/onboarding/OnboardingLayout'
import { authApi, onboardingApi } from '@/api'
import { isLaravelFile } from '@/components/features/backups/LaravelRestoreOptions'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'
import { pages } from '@/app/pages'
import { testId } from '@/lib/test-id'
import type { ApiError } from '@/types'

type AdminSetupValues = {
    name: string
    email: string
    password: string
    password_confirmation: string
}

// The new admin goes on to the server setup, then 2FA.
const NEXT_PATH = pages.serverSetup.path

/** Flow 1: the bootstrap password, then the first admin. */
export default function AdminSetupPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const register = useAuthStore((state) => state.register)
    const [step, setStep] = useState<'bootstrap' | 'admin' | 'restore'>('bootstrap')
    const [isLoading, setIsLoading] = useState(false)
    // Not a form field: a field of the first screen would lose its value when the screen unmounts.
    const [bootstrapPassword, setBootstrapPassword] = useState('')
    const [bootstrapError, setBootstrapError] = useState<string | null>(null)
    const [backupFile, setBackupFile] = useState<File | null>(null)
    const [appKey, setAppKey] = useState('')
    const [restoreError, setRestoreError] = useState<string | null>(null)

    const { data: status } = useQuery({ queryKey: ['auth', 'status'], queryFn: authApi.status, retry: false })

    // Registering is what makes needs_registration false: that must not send the new admin to the login page.
    const registered = useRef(false)

    useEffect(() => {
        if (status && !status.needs_registration && !registered.current) {
            navigate('/login', { replace: true })
        }
    }, [status, navigate])

    const schema = useMemo(() => z.object({
        name: z.string().min(1, t('setup.validation.name')),
        email: z.string().email(t('setup.validation.email')),
        password: z.string().min(6, t('setup.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('setup.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<AdminSetupValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: '', email: '', password: '', password_confirmation: '' },
    })

    const goTo = async (next: 'admin' | 'restore') => {
        const password = bootstrapPassword.trim()
        if (password === '') {
            setBootstrapError(t('setup.validation.bootstrapPassword'))
            return
        }
        setIsLoading(true)
        try {
            await authApi.verifyBootstrap(password)
            setBootstrapError(null)
            setStep(next)
        } catch (error: unknown) {
            const wrong = (error as ApiError).details?.bootstrap_password?.[0]
            if (wrong) {
                setBootstrapError(wrong)
            } else {
                toast.error(getApiErrorMessage(error, t('setup.failed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    const onRestore = async () => {
        if (!backupFile) {
            setRestoreError(t('setup.validation.backupFile'))
            return
        }
        setIsLoading(true)
        setRestoreError(null)
        try {
            await onboardingApi.restore(
                bootstrapPassword.trim(),
                backupFile,
                isLaravelFile(backupFile) && appKey.trim() !== '' ? appKey.trim() : undefined,
            )
            // The backup brought its own users and no session: drop the cached needs_registration=true and sign in.
            registered.current = true
            await queryClient.invalidateQueries({ queryKey: ['auth', 'status'] })
            toast.success(t('setup.restored'))
            navigate('/login', { replace: true })
        } catch (error: unknown) {
            const details = (error as ApiError).details
            if (details?.bootstrap_password?.[0]) {
                setBootstrapError(details.bootstrap_password[0])
                setStep('bootstrap')
            } else if (details?.file?.[0]) {
                setRestoreError(details.file[0])
            } else {
                toast.error(getApiErrorMessage(error, t('setup.restoreFailed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    const onSubmit = async (data: AdminSetupValues) => {
        setIsLoading(true)
        try {
            await register({
                bootstrap_password: bootstrapPassword.trim(),
                name: data.name,
                email: data.email,
                password: data.password,
            })
            registered.current = true
            // AuthProvider caches the status forever; drop the stale needs_registration=true.
            await queryClient.invalidateQueries({ queryKey: ['auth', 'status'] })
            toast.success(t('setup.created'))
            navigate(NEXT_PATH)
        } catch (error: unknown) {
            const wrongPassword = (error as ApiError).details?.bootstrap_password?.[0]
            if (wrongPassword) {
                // The password is on the first screen: take the user back to it.
                setBootstrapError(wrongPassword)
                setStep('bootstrap')
            } else {
                toast.error(getApiErrorMessage(error, t('setup.failed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    if (!status?.needs_registration) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    const hint = status.bootstrap_hint

    return (
        <OnboardingLayout
            title={step === 'bootstrap' ? t('setup.title') : step === 'restore' ? t('setup.restoreTitle') : t('setup.adminTitle')}
            description={step === 'bootstrap' ? t('setup.description') : step === 'restore' ? t('setup.restoreDescription') : t('setup.adminDescription')}
        >
            <Form {...form}>
                <form
                    onSubmit={(e) => {
                        // Enter in the password field submits the form: on the first screen that means "next",
                        // not validating admin fields the user has not seen yet.
                        if (step === 'bootstrap') {
                            e.preventDefault()
                            void goTo('admin')
                            return
                        }
                        if (step === 'restore') {
                            e.preventDefault()
                            void onRestore()
                            return
                        }
                        void form.handleSubmit(onSubmit)(e)
                    }}
                    className="space-y-4"
                >
                    {step === 'bootstrap' ? (
                        <>
                            {hint && <BootstrapInstructions hint={hint} />}
                            <div className="space-y-2">
                                <Label htmlFor="bootstrap-password">{t('setup.bootstrapPassword')}</Label>
                                <Input
                                    id="bootstrap-password"
                                    value={bootstrapPassword}
                                    onChange={(e) => setBootstrapPassword(e.target.value)}
                                    placeholder="XXXXX-XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"
                                    autoComplete="off"
                                    autoCapitalize="characters"
                                    spellCheck={false}
                                    aria-invalid={bootstrapError !== null}
                                />
                                {bootstrapError && <p className="text-sm text-destructive">{bootstrapError}</p>}
                            </div>
                            <Button type="button" className="w-full" disabled={isLoading} onClick={() => goTo('admin')} {...testId('setup-create-admin')}>
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('setup.createAdmin')}
                            </Button>
                            <Button type="button" variant="outline" className="w-full" disabled={isLoading} onClick={() => goTo('restore')} {...testId('setup-restore')}>
                                {t('setup.restore')}
                            </Button>
                        </>
                    ) : step === 'restore' ? (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="backup-file">{t('setup.backupFile')}</Label>
                                <Input
                                    id="backup-file"
                                    type="file"
                                    accept=".zip,.sqlite"
                                    onChange={(e) => {
                                        setBackupFile(e.target.files?.[0] ?? null)
                                        setRestoreError(null)
                                    }}
                                    aria-invalid={restoreError !== null}
                                />
                                <p className="text-xs text-muted-foreground">{t('setup.backupFileHint')}</p>
                                {restoreError && <p className="text-sm text-destructive">{restoreError}</p>}
                            </div>
                            {isLaravelFile(backupFile) && (
                                <div className="space-y-2">
                                    <Label htmlFor="restore-app-key">{t('setup.appKey')}</Label>
                                    <Input
                                        id="restore-app-key"
                                        value={appKey}
                                        onChange={(e) => setAppKey(e.target.value)}
                                        placeholder="base64:..."
                                        autoComplete="off"
                                        spellCheck={false}
                                    />
                                    <p className="text-xs text-muted-foreground">{t('setup.appKeyHint')}</p>
                                </div>
                            )}
                            <Button type="submit" className="w-full" disabled={isLoading}>
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('setup.restoreSubmit')}
                            </Button>
                            <Button type="button" variant="ghost" className="w-full" disabled={isLoading} onClick={() => setStep('bootstrap')} {...testId('setup-back')}>
                                {t('setup.back')}
                            </Button>
                        </>
                    ) : (
                        <>
                            <FormField
                                control={form.control}
                                name="name"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>{t('setup.name')}</FormLabel>
                                        <FormControl>
                                            <Input placeholder={t('setup.namePlaceholder')} autoComplete="name" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="email"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>{t('email')}</FormLabel>
                                        <FormControl>
                                            <Input type="email" placeholder={t('emailPlaceholder')} autoComplete="email" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="password"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>{t('password')}</FormLabel>
                                        <FormControl>
                                            <Input type="password" placeholder={t('setup.passwordPlaceholder')} autoComplete="new-password" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="password_confirmation"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>{t('setup.confirmPassword')}</FormLabel>
                                        <FormControl>
                                            <Input type="password" placeholder={t('setup.confirmPlaceholder')} autoComplete="new-password" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <Button type="submit" className="w-full" disabled={isLoading}>
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('setup.submit')}
                            </Button>
                            <Button type="button" variant="ghost" className="w-full" disabled={isLoading} onClick={() => setStep('bootstrap')} {...testId('setup-back')}>
                                {t('setup.back')}
                            </Button>
                        </>
                    )}
                </form>
            </Form>
        </OnboardingLayout>
    )
}
