import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
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
import { Loader2, Sparkles } from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'
import { authApi } from '@/api'
import { getApiErrorMessage } from '@/lib/api-error'
import { toast } from 'sonner'

type SetupFormValues = {
    name: string
    email: string
    password: string
    password_confirmation: string
}

export default function SetupPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const register = useAuthStore((state) => state.register)
    const [isLoading, setIsLoading] = useState(false)
    const [checkingStatus, setCheckingStatus] = useState(true)

    useEffect(() => {
        authApi.status().then((status) => {
            if (!status.needs_registration) {
                navigate('/login', { replace: true })
            }
        }).finally(() => setCheckingStatus(false))
    }, [navigate])

    const setupSchema = useMemo(() => z.object({
        name: z.string().min(1, t('setup.validation.name')),
        email: z.string().email(t('setup.validation.email')),
        password: z.string().min(6, t('setup.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('setup.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<SetupFormValues>({
        resolver: zodResolver(setupSchema),
        defaultValues: {
            name: '',
            email: '',
            password: '',
            password_confirmation: '',
        },
    })

    const onSubmit = async (data: SetupFormValues) => {
        setIsLoading(true)
        try {
            await register({
                name: data.name,
                email: data.email,
                password: data.password,
            })
            // AuthProvider caches the status forever; drop the stale needs_registration=true.
            await queryClient.invalidateQueries({ queryKey: ['auth', 'status'] })
            toast.success(t('setup.created'))
            navigate('/setup-2fa')
        } catch (error: unknown) {
            toast.error(getApiErrorMessage(error, t('setup.failed')))
        } finally {
            setIsLoading(false)
        }
    }

    if (checkingStatus) {
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
                    <CardTitle className="text-2xl flex items-center justify-center gap-2">
                        <Sparkles className="size-5 text-primary" />
                        {t('setup.title')}
                    </CardTitle>
                    <CardDescription>
                        {t('setup.description')}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                            <FormField
                                control={form.control}
                                name="name"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>{t('setup.name')}</FormLabel>
                                        <FormControl>
                                            <Input
                                                placeholder={t('setup.namePlaceholder')}
                                                autoComplete="name"
                                                {...field}
                                            />
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
                                            <Input
                                                type="email"
                                                placeholder={t('emailPlaceholder')}
                                                autoComplete="email"
                                                {...field}
                                            />
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
                                            <Input
                                                type="password"
                                                placeholder={t('setup.passwordPlaceholder')}
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

                            <Button type="submit" className="w-full" disabled={isLoading}>
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('setup.submit')}
                            </Button>
                        </form>
                    </Form>
                    <LanguageSwitcher variant="auth" className="pt-4" />
                </CardContent>
            </Card>
        </div>
    )
}
