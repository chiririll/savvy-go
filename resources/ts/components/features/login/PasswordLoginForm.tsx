import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'

type LoginFormValues = {
    email: string
    password: string
    remember_me: boolean
}

interface Props {
    afterLoginPath: string
    /** The password was right but a second factor is needed. */
    onTwoFactor: (token: string, rememberMe: boolean) => void
}

export function PasswordLoginForm({ afterLoginPath, onTwoFactor }: Props) {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const login = useAuthStore((state) => state.login)
    const [isLoading, setIsLoading] = useState(false)

    const loginSchema = useMemo(() => z.object({
        email: z.string().email(t('validation.email')),
        password: z.string().min(1, t('validation.passwordRequired')),
        remember_me: z.boolean(),
    }), [t])

    const form = useForm<LoginFormValues>({
        resolver: zodResolver(loginSchema),
        defaultValues: {
            email: '',
            password: '',
            remember_me: true,
        },
    })

    const onSubmit = async (data: LoginFormValues) => {
        setIsLoading(true)
        try {
            const result = await login(data)
            if (result.success) {
                toast.success(t('welcomeToast'))
                navigate(afterLoginPath)
            } else if (result.requires_2fa) {
                onTwoFactor(result.two_factor_token, data.remember_me)
            }
        } catch (error: unknown) {
            toast.error(getApiErrorMessage(error, t('invalidCredentials')))
        } finally {
            setIsLoading(false)
        }
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
                                    autoComplete="username"
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
                                    placeholder={t('passwordPlaceholder')}
                                    autoComplete="current-password"
                                    {...field}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="remember_me"
                    render={({ field }) => (
                        <FormItem className="flex flex-row items-center gap-2 space-y-0">
                            <FormControl>
                                <Checkbox
                                    checked={field.value}
                                    onCheckedChange={(checked) => field.onChange(checked === true)}
                                />
                            </FormControl>
                            <FormLabel className="font-normal">{t('rememberMe')}</FormLabel>
                        </FormItem>
                    )}
                />

                <Button type="submit" className="w-full" disabled={isLoading}>
                    {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                    {t('signIn')}
                </Button>
            </form>
        </Form>
    )
}
