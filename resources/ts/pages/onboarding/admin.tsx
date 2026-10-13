import { useEffect, useMemo, useState } from 'react'
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
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'
import type { ApiError } from '@/types'

type AdminSetupValues = {
    name: string
    email: string
    password: string
    password_confirmation: string
}

// Until server setup (task 04) exists, the new admin goes on to 2FA as before.
const NEXT_PATH = '/setup-2fa'

/** Flow 1: the bootstrap password, then the first admin. */
export default function AdminSetupPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const register = useAuthStore((state) => state.register)
    const [step, setStep] = useState<'bootstrap' | 'admin'>('bootstrap')
    const [isLoading, setIsLoading] = useState(false)
    // Not a form field: a field of the first screen would lose its value when the screen unmounts.
    const [bootstrapPassword, setBootstrapPassword] = useState('')
    const [bootstrapError, setBootstrapError] = useState<string | null>(null)

    const { data: status } = useQuery({ queryKey: ['auth', 'status'], queryFn: authApi.status, retry: false })

    useEffect(() => {
        if (status && !status.needs_registration) {
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

    const goToAdmin = async () => {
        const password = bootstrapPassword.trim()
        if (password === '') {
            setBootstrapError(t('setup.validation.bootstrapPassword'))
            return
        }
        setIsLoading(true)
        try {
            await authApi.verifyBootstrap(password)
            setBootstrapError(null)
            setStep('admin')
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

    const onSubmit = async (data: AdminSetupValues) => {
        setIsLoading(true)
        try {
            await register({
                bootstrap_password: bootstrapPassword.trim(),
                name: data.name,
                email: data.email,
                password: data.password,
            })
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
            title={step === 'bootstrap' ? t('setup.title') : t('setup.adminTitle')}
            description={step === 'bootstrap' ? t('setup.description') : t('setup.adminDescription')}
        >
            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
                            {/* Restoring a server backup (task 08) becomes the second choice here. */}
                            <Button type="button" className="w-full" disabled={isLoading} onClick={goToAdmin}>
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('setup.createAdmin')}
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
                            <Button type="button" variant="ghost" className="w-full" disabled={isLoading} onClick={() => setStep('bootstrap')}>
                                {t('setup.back')}
                            </Button>
                        </>
                    )}
                </form>
            </Form>
        </OnboardingLayout>
    )
}
