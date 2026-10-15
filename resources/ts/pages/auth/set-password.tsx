import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form'
import { Loader2 } from 'lucide-react'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { PasswordTokenPreview } from '@/types/users'
import { toast } from 'sonner'

type FormValues = {
    password: string
    password_confirmation: string
}

export default function SetPasswordPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const { token } = useParams<{ token: string }>()
    const applySession = useAuthStore((state) => state.applySession)
    const [preview, setPreview] = useState<PasswordTokenPreview | null>(null)
    const [loading, setLoading] = useState(true)
    const [invalid, setInvalid] = useState(false)
    const [submitting, setSubmitting] = useState(false)

    useEffect(() => {
        if (!token) {
            setInvalid(true)
            setLoading(false)
            return
        }

        authApi.previewPasswordToken(token)
            .then(setPreview)
            .catch(() => setInvalid(true))
            .finally(() => setLoading(false))
    }, [token])

    const schema = useMemo(() => z.object({
        password: z.string().min(8, t('setPassword.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('setPassword.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            password: '',
            password_confirmation: '',
        },
    })

    const onSubmit = async (data: FormValues) => {
        if (!token) return
        setSubmitting(true)
        try {
            const response = await authApi.acceptPasswordToken(token, data.password, data.password_confirmation)
            if ('requires_2fa' in response && response.requires_2fa) {
                toast.success(t('setPassword.updated'))
                navigate('/login', { replace: true, state: { twoFactorToken: response.two_factor_token, rememberMe: true } })
                return
            }
            if ('user' in response) {
                applySession(response)
            }
            toast.success(t('setPassword.updated'))
            navigate('/', { replace: true })
        } catch (error: unknown) {
            const message = error && typeof error === 'object' && 'message' in error
                ? (error as { message: string }).message
                : t('setPassword.failed')
            toast.error(message)
        } finally {
            setSubmitting(false)
        }
    }

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="flex justify-center mb-4">
                        <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                            <Logo className="size-7" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">
                        {invalid
                            ? t('setPassword.invalidTitle')
                            : preview?.isInactive
                                ? t('setPassword.inviteTitle')
                                : t('setPassword.resetTitle')}
                    </CardTitle>
                    <CardDescription>
                        {invalid
                            ? t('setPassword.invalidDescription')
                            : t('setPassword.description', { email: preview?.email ?? '' })}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {!invalid && (
                        <Form {...form}>
                            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                                <FormField
                                    control={form.control}
                                    name="password"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>{t('password')}</FormLabel>
                                            <FormControl>
                                                <Input
                                                    type="password"
                                                    placeholder={t('setPassword.passwordPlaceholder')}
                                                    autoComplete="new-password"
                                                    {...field}
                                                />
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
                                                <Input
                                                    type="password"
                                                    placeholder={t('setup.confirmPlaceholder')}
                                                    autoComplete="new-password"
                                                    {...field}
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <Button type="submit" className="w-full" disabled={submitting}>
                                    {submitting && <Loader2 className="mr-2 size-4 animate-spin" />}
                                    {t('setPassword.submit')}
                                </Button>
                            </form>
                        </Form>
                    )}
                    {invalid && (
                        <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
                            {t('setPassword.backToLogin')}
                        </Button>
                    )}
                </CardContent>
            </Card>
            <LanguageSwitcher variant="auth" />
        </div>
    )
}
