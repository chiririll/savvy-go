import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom'
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
import {
    InputOTP,
    InputOTPGroup,
    InputOTPSlot,
    InputOTPSeparator,
} from '@/components/ui/input-otp'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, ArrowLeft, Key, Fingerprint } from 'lucide-react'
import { browserSupportsWebAuthn } from '@simplewebauthn/browser'
import { useAuthStore } from '@/stores/auth'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'
import { SsoButtons } from '@/components/features/sso'
import { authApi } from '@/api'
import { passkeyErrorMessage, isPasskeyDomainSupported } from '@/hooks'
import { getApiErrorMessage } from '@/lib/api-error'
import { toast } from 'sonner'

type LoginFormValues = {
    email: string
    password: string
    remember_me: boolean
}

type LoginLocationState = {
    twoFactorToken?: string
    rememberMe?: boolean
    sessionExpired?: boolean
    from?: string | { pathname?: string; search?: string }
}

function safeReturnPath(from: LoginLocationState['from']): string {
    if (typeof from === 'string') {
        return from.startsWith('/') && !from.startsWith('//') ? from : '/'
    }
    const pathname = from?.pathname
    if (typeof pathname === 'string' && pathname.startsWith('/') && !pathname.startsWith('//')) {
        return `${pathname}${from?.search ?? ''}`
    }
    return '/'
}

export default function LoginPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const location = useLocation()
    const [searchParams] = useSearchParams()
    const login = useAuthStore((state) => state.login)
    const loginWith2FA = useAuthStore((state) => state.loginWith2FA)
    const loginWithPasskey = useAuthStore((state) => state.loginWithPasskey)
    const [isLoading, setIsLoading] = useState(false)
    const [checkingStatus, setCheckingStatus] = useState(true)
    const [passwordLoginEnabled, setPasswordLoginEnabled] = useState(true)
    const [twoFactorToken, setTwoFactorToken] = useState<string | null>(null)
    const [otpValue, setOtpValue] = useState('')
    const [useRecoveryCode, setUseRecoveryCode] = useState(false)
    const [recoveryCode, setRecoveryCode] = useState('')
    const [passkeyLoading, setPasskeyLoading] = useState(false)
    const [rememberMe, setRememberMe] = useState(false)
    const [passkeySupported] = useState(() => browserSupportsWebAuthn() && isPasskeyDomainSupported())
    const loginState = (location.state as LoginLocationState | null) ?? null
    const sessionExpiredNotice = Boolean(loginState?.sessionExpired)
    const afterLoginPath = safeReturnPath(loginState?.from)

    useEffect(() => {
        authApi.status().then((status) => {
            if (status.needs_registration) {
                navigate('/setup', { replace: true })
            }
            setPasswordLoginEnabled(status.password_login_enabled)
        }).finally(() => setCheckingStatus(false))
    }, [navigate])

    // Resume the 2FA step when arriving from the SSO callback.
    useEffect(() => {
        const incoming = location.state as LoginLocationState | null
        if (incoming?.twoFactorToken) {
            setTwoFactorToken(incoming.twoFactorToken)
            if (incoming.rememberMe) {
                setRememberMe(true)
            }
            navigate(location.pathname, { replace: true, state: null })
        }
    }, [location.state, location.pathname, navigate])

    // Surface SSO errors bounced back to the login screen.
    useEffect(() => {
        if (searchParams.get('sso_error')) {
            toast.error(t('ssoFailed'))
        }
    }, [searchParams, t])

    const handlePasskeyLogin = async (stepUpToken?: string) => {
        setPasskeyLoading(true)
        try {
            await loginWithPasskey({ twoFactorToken: stepUpToken })
            toast.success(t('welcomeToast'))
            navigate(afterLoginPath)
        } catch (error: unknown) {
            toast.error(passkeyErrorMessage(error, t('passkeyFailed')))
        } finally {
            setPasskeyLoading(false)
        }
    }

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
                setRememberMe(data.remember_me)
                setTwoFactorToken(result.two_factor_token)
            }
        } catch (error: unknown) {
            toast.error(getApiErrorMessage(error, t('invalidCredentials')))
        } finally {
            setIsLoading(false)
        }
    }

    const onVerify2FA = async () => {
        const code = useRecoveryCode ? recoveryCode : otpValue
        if (!twoFactorToken || (!useRecoveryCode && otpValue.length !== 6) || (useRecoveryCode && !recoveryCode)) return

        setIsLoading(true)
        try {
            await loginWith2FA(twoFactorToken, code, rememberMe)
            toast.success(t('welcomeToast'))
            navigate(afterLoginPath)
        } catch (error: unknown) {
            toast.error(getApiErrorMessage(error, t('twoFactor.invalidCode')))
            if (useRecoveryCode) {
                setRecoveryCode('')
            } else {
                setOtpValue('')
            }
        } finally {
            setIsLoading(false)
        }
    }

    const handleBack = () => {
        setTwoFactorToken(null)
        setOtpValue('')
        setRecoveryCode('')
        setUseRecoveryCode(false)
    }

    if (checkingStatus) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    // 2FA Verification Step
    if (twoFactorToken) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background p-4">
                <Card className="w-full max-w-md">
                    <CardHeader className="text-center">
                        <div className="flex justify-center mb-4">
                            <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                                <Logo className="size-7" />
                            </div>
                        </div>
                        <CardTitle className="text-2xl">{t('twoFactor.title')}</CardTitle>
                        <CardDescription>
                            {useRecoveryCode
                                ? t('twoFactor.recoveryDescription')
                                : t('twoFactor.otpDescription')}
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-6">
                        {useRecoveryCode ? (
                            <Input
                                type="text"
                                placeholder={t('twoFactor.recoveryPlaceholder')}
                                value={recoveryCode}
                                onChange={(e) => setRecoveryCode(e.target.value)}
                                className="text-center font-mono text-lg"
                                autoFocus
                            />
                        ) : (
                            <div className="flex justify-center">
                                <InputOTP
                                    maxLength={6}
                                    value={otpValue}
                                    onChange={setOtpValue}
                                    onComplete={onVerify2FA}
                                >
                                    <InputOTPGroup>
                                        <InputOTPSlot index={0} />
                                        <InputOTPSlot index={1} />
                                        <InputOTPSlot index={2} />
                                    </InputOTPGroup>
                                    <InputOTPSeparator />
                                    <InputOTPGroup>
                                        <InputOTPSlot index={3} />
                                        <InputOTPSlot index={4} />
                                        <InputOTPSlot index={5} />
                                    </InputOTPGroup>
                                </InputOTP>
                            </div>
                        )}

                        <div className="flex flex-col gap-2">
                            <Button
                                onClick={onVerify2FA}
                                className="w-full"
                                disabled={(!useRecoveryCode && otpValue.length !== 6) || (useRecoveryCode && !recoveryCode) || isLoading}
                            >
                                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                                {t('twoFactor.verify')}
                            </Button>
                            {passkeySupported && (
                                <Button
                                    variant="outline"
                                    onClick={() => handlePasskeyLogin(twoFactorToken)}
                                    className="w-full"
                                    disabled={passkeyLoading || isLoading}
                                >
                                    {passkeyLoading
                                        ? <Loader2 className="mr-2 size-4 animate-spin" />
                                        : <Fingerprint className="mr-2 size-4" />}
                                    {t('twoFactor.usePasskey')}
                                </Button>
                            )}
                            <Button
                                variant="ghost"
                                onClick={() => {
                                    setUseRecoveryCode(!useRecoveryCode)
                                    setOtpValue('')
                                    setRecoveryCode('')
                                }}
                                className="w-full"
                            >
                                <Key className="mr-2 size-4" />
                                {useRecoveryCode ? t('twoFactor.useAuthenticator') : t('twoFactor.useRecovery')}
                            </Button>
                            <Button
                                variant="ghost"
                                onClick={handleBack}
                                className="w-full"
                            >
                                <ArrowLeft className="mr-2 size-4" />
                                {t('twoFactor.backToLogin')}
                            </Button>
                            <LanguageSwitcher variant="auth" className="pt-2" />
                        </div>
                    </CardContent>
                </Card>
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
                    <CardTitle className="text-2xl">{t('welcomeBack')}</CardTitle>
                    <CardDescription>
                        {t('signInDescription')}
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    {sessionExpiredNotice && (
                        <Alert>
                            <AlertDescription>{t('sessionExpired.banner')}</AlertDescription>
                        </Alert>
                    )}
                    {passkeySupported && (
                        <Button
                            variant="outline"
                            className="w-full"
                            onClick={() => handlePasskeyLogin()}
                            disabled={passkeyLoading}
                        >
                            {passkeyLoading
                                ? <Loader2 className="mr-2 size-4 animate-spin" />
                                : <Fingerprint className="mr-2 size-4" />}
                            {t('signInPasskey')}
                        </Button>
                    )}
                    <SsoButtons showDivider={passwordLoginEnabled} />
                    {passwordLoginEnabled && (
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
                    )}
                    <LanguageSwitcher variant="auth" className="pt-2" />
                </CardContent>
            </Card>
        </div>
    )
}
