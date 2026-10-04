import { ReactNode, useMemo, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
    InputOTP,
    InputOTPGroup,
    InputOTPSlot,
    InputOTPSeparator,
} from '@/components/ui/input-otp'
import {
    useChangePassword,
    useLogoutOthers,
    useTwoFactorStatus,
    useEnableTwoFactor,
    useConfirmTwoFactor,
    useDisableTwoFactor,
    useRegenerateRecoveryCodes,
    useWebauthnCredentials,
    useRegisterPasskey,
    useDeletePasskey,
    isPasskeyDomainSupported,
} from '@/hooks'
import { browserSupportsWebAuthn } from '@simplewebauthn/browser'
import { ShieldCheck, Key, Copy, RefreshCw, Fingerprint, Trash2, Plus, Lock, LogOut } from 'lucide-react'
import { useUser } from '@/stores/auth'
import { toast } from 'sonner'
import { QRCode } from 'react-qrcode-logo'
import { useTheme } from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { intlLocale } from '@/lib/i18n'

type SetupStep = 'qr' | 'verify' | 'recovery'

function Section({
    icon: Icon,
    title,
    description,
    children,
}: {
    icon: LucideIcon
    title: string
    description: string
    children: ReactNode
}) {
    return (
        <section className="rounded-xl border bg-card shadow-sm">
            <header className="flex items-start gap-3 border-b px-5 py-4">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50 text-muted-foreground">
                    <Icon className="size-4" />
                </div>
                <div className="space-y-0.5">
                    <h3 className="text-sm font-semibold leading-none">{title}</h3>
                    <p className="text-xs text-muted-foreground">{description}</p>
                </div>
            </header>
            <div className="p-5">{children}</div>
        </section>
    )
}

function ActionRow({
    label,
    description,
    badge,
    children,
}: {
    label: string
    description: ReactNode
    badge?: ReactNode
    children: ReactNode
}) {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="min-w-0 space-y-1">
                <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{label}</span>
                    {badge}
                </div>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            {children}
        </div>
    )
}

type PasswordFormValues = {
    current_password: string
    password: string
    password_confirmation: string
}

function ChangePasswordSection() {
    const { t } = useTranslation('settings')
    const user = useUser()
    const changePassword = useChangePassword()

    const schema = useMemo(() => z.object({
        current_password: z.string().min(1, t('security.password.validation.currentRequired')),
        password: z.string().min(8, t('security.password.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('security.password.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<PasswordFormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            current_password: '',
            password: '',
            password_confirmation: '',
        },
    })

    const onSubmit = async (data: PasswordFormValues) => {
        try {
            await changePassword.mutateAsync(data)
            form.reset()
        } catch {
            // toasted by the mutation
        }
    }

    return (
        <Section
            icon={Lock}
            title={t('security.password.title')}
            description={t('security.password.description')}
        >
            {user?.isSsoOnly ? (
                <p className="text-sm text-muted-foreground">{t('security.password.ssoOnly')}</p>
            ) : (
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                        <FormField
                            control={form.control}
                            name="current_password"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('security.password.current')}</FormLabel>
                                    <FormControl>
                                        <Input type="password" autoComplete="current-password" {...field} />
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
                                    <FormLabel>{t('security.password.new')}</FormLabel>
                                    <FormControl>
                                        <Input type="password" autoComplete="new-password" {...field} />
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
                                    <FormLabel>{t('security.password.confirm')}</FormLabel>
                                    <FormControl>
                                        <Input type="password" autoComplete="new-password" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <Button type="submit" disabled={changePassword.isPending}>
                            {t('security.password.submit')}
                        </Button>
                    </form>
                </Form>
            )}
        </Section>
    )
}

function SessionsSection({
    onConfirm,
}: {
    onConfirm: () => void
}) {
    const { t } = useTranslation('settings')

    return (
        <Section
            icon={LogOut}
            title={t('security.sessions.title')}
            description={t('security.sessions.description')}
        >
            <div className="space-y-4">
                <p className="text-sm leading-relaxed text-muted-foreground">
                    {t('security.sessions.logoutOthersDescription')}
                </p>
                <Button variant="outline" onClick={onConfirm}>
                    <LogOut className="mr-2 size-4" />
                    {t('security.sessions.logoutOthers')}
                </Button>
            </div>
        </Section>
    )
}

function RowSkeleton() {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-64" />
            </div>
            <Skeleton className="h-9 w-24" />
        </div>
    )
}

export default function SecuritySettingsPage() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const { theme } = useTheme()
    const isReadOnly = useReadOnly()
    const { data: status, isLoading } = useTwoFactorStatus()
    const enableMutation = useEnableTwoFactor()
    const confirmMutation = useConfirmTwoFactor()
    const disableMutation = useDisableTwoFactor()
    const regenerateMutation = useRegenerateRecoveryCodes()

    const passkeySupported = browserSupportsWebAuthn()
    const passkeyDomainValid = isPasskeyDomainSupported()
    const { data: passkeys, isLoading: passkeysLoading } = useWebauthnCredentials()
    const registerPasskey = useRegisterPasskey()
    const deletePasskey = useDeletePasskey()

    const [showEnableDialog, setShowEnableDialog] = useState(false)
    const [showDisableDialog, setShowDisableDialog] = useState(false)
    const [showRegenerateDialog, setShowRegenerateDialog] = useState(false)
    const [setupStep, setSetupStep] = useState<SetupStep>('qr')
    const [qrData, setQrData] = useState<{ secret: string; qr_code_url: string } | null>(null)
    const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
    const [otpValue, setOtpValue] = useState('')
    const [showAddPasskeyDialog, setShowAddPasskeyDialog] = useState(false)
    const [passkeyName, setPasskeyName] = useState('')
    const [showLogoutOthersDialog, setShowLogoutOthersDialog] = useState(false)
    const logoutOthers = useLogoutOthers()

    const formatDate = (value: string) => new Date(value).toLocaleDateString(intlLocale())

    const handleEnable = async () => {
        const data = await enableMutation.mutateAsync()
        setQrData(data)
        setSetupStep('qr')
        setShowEnableDialog(true)
    }

    const handleConfirm = async () => {
        const data = await confirmMutation.mutateAsync(otpValue)
        setRecoveryCodes(data.recovery_codes)
        setSetupStep('recovery')
        setOtpValue('')
    }

    const handleDisable = async () => {
        await disableMutation.mutateAsync(otpValue)
        setShowDisableDialog(false)
        setOtpValue('')
    }

    const handleRegenerate = async () => {
        const data = await regenerateMutation.mutateAsync(otpValue)
        setRecoveryCodes(data.recovery_codes)
        setShowRegenerateDialog(false)
        setSetupStep('recovery')
        setShowEnableDialog(true)
        setOtpValue('')
    }

    const handleAddPasskey = async () => {
        await registerPasskey.mutateAsync(passkeyName.trim() || null)
        setShowAddPasskeyDialog(false)
        setPasskeyName('')
    }

    const copySecret = () => {
        if (qrData?.secret) {
            navigator.clipboard.writeText(qrData.secret)
            toast.success(t('security.dialogs.secretCopied'))
        }
    }

    const copyRecoveryCodes = () => {
        navigator.clipboard.writeText(recoveryCodes.join('\n'))
        toast.success(t('security.dialogs.codesCopied'))
    }

    const closeEnableDialog = () => {
        setShowEnableDialog(false)
        setSetupStep('qr')
        setQrData(null)
        setRecoveryCodes([])
        setOtpValue('')
    }

    return (
        <Page title={t('security.title')}>
            <PageHeader title={t('security.heading')} description={t('security.description')} />

            <FormWrapper>
                <div className="grid items-start gap-6 lg:grid-cols-2">
                    <ChangePasswordSection />
                    <SessionsSection onConfirm={() => setShowLogoutOthersDialog(true)} />

                    <Section
                        icon={ShieldCheck}
                        title={t('security.twoFactor.title')}
                        description={t('security.twoFactor.description')}
                    >
                        <div className="rounded-lg border">
                            {isLoading ? (
                                <RowSkeleton />
                            ) : (
                                <ActionRow
                                    label={t('security.twoFactor.authenticator')}
                                    description={t('security.twoFactor.authenticatorDescription')}
                                >
                                    <Switch
                                        checked={status?.enabled ?? false}
                                        disabled={isReadOnly || enableMutation.isPending}
                                        onCheckedChange={(checked) =>
                                            checked ? handleEnable() : setShowDisableDialog(true)
                                        }
                                        aria-label={t('security.twoFactor.toggleAria')}
                                    />
                                </ActionRow>
                            )}
                        </div>
                    </Section>

                    {passkeySupported && (
                        <Section
                            icon={Fingerprint}
                            title={t('security.passkeys.title')}
                            description={t('security.passkeys.description')}
                        >
                            <div className="space-y-4">
                                {passkeysLoading ? (
                                    <div className="rounded-lg border">
                                        <RowSkeleton />
                                    </div>
                                ) : passkeys && passkeys.length > 0 ? (
                                    <div className="divide-y rounded-lg border">
                                        {passkeys.map((passkey) => (
                                            <div key={passkey.id} className="flex items-center justify-between gap-4 px-4 py-3.5">
                                                <div className="min-w-0 space-y-1">
                                                    <div className="truncate text-sm font-medium">
                                                        {passkey.name || t('security.passkeys.unnamed')}
                                                    </div>
                                                    <p className="text-sm leading-relaxed text-muted-foreground">
                                                        {t('security.passkeys.added', { date: formatDate(passkey.createdAt) })}
                                                        {' · '}
                                                        {passkey.lastUsedAt
                                                            ? t('security.passkeys.lastUsed', { date: formatDate(passkey.lastUsedAt) })
                                                            : t('security.passkeys.neverUsed')}
                                                    </p>
                                                </div>
                                                <AlertDialog>
                                                    <AlertDialogTrigger asChild>
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            disabled={isReadOnly || deletePasskey.isPending}
                                                            aria-label={t('security.passkeys.removeAria')}
                                                        >
                                                            <Trash2 className="size-4 text-destructive" />
                                                        </Button>
                                                    </AlertDialogTrigger>
                                                    <AlertDialogContent>
                                                        <AlertDialogHeader>
                                                            <AlertDialogTitle>
                                                                {t('security.passkeys.deleteTitle')}
                                                            </AlertDialogTitle>
                                                            <AlertDialogDescription>
                                                                {t('security.passkeys.deleteDescription')}
                                                            </AlertDialogDescription>
                                                        </AlertDialogHeader>
                                                        <AlertDialogFooter>
                                                            <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                                                            <AlertDialogAction
                                                                variant="destructive"
                                                                onClick={() => deletePasskey.mutate(passkey.id)}
                                                            >
                                                                {tCommon('actions.delete')}
                                                            </AlertDialogAction>
                                                        </AlertDialogFooter>
                                                    </AlertDialogContent>
                                                </AlertDialog>
                                            </div>
                                        ))}
                                    </div>
                                ) : (
                                    <div className="rounded-lg border border-dashed px-4 py-6 text-center text-sm text-muted-foreground">
                                        {t('security.passkeys.empty')}
                                    </div>
                                )}
                                {!passkeyDomainValid && (
                                    <p className="text-sm text-muted-foreground">
                                        {t('security.passkeys.domainHint')}
                                    </p>
                                )}
                                <Button
                                    onClick={() => setShowAddPasskeyDialog(true)}
                                    disabled={isReadOnly || registerPasskey.isPending || !passkeyDomainValid}
                                >
                                    <Plus className="mr-2 size-4" />
                                    {t('security.passkeys.add')}
                                </Button>
                            </div>
                        </Section>
                    )}

                    {status?.enabled && (
                        <Section
                            icon={Key}
                            title={t('security.recovery.title')}
                            description={t('security.recovery.description')}
                        >
                            <div className="rounded-lg border">
                                <ActionRow
                                    label={t('security.recovery.backup')}
                                    description={t('security.recovery.remaining', { count: status.recovery_codes_remaining })}
                                >
                                    <Button
                                        variant="outline"
                                        onClick={() => setShowRegenerateDialog(true)}
                                        disabled={isReadOnly}
                                    >
                                        <RefreshCw className="mr-2 size-4" />
                                        {t('security.recovery.regenerate')}
                                    </Button>
                                </ActionRow>
                            </div>
                        </Section>
                    )}
                </div>
            </FormWrapper>

            <Dialog open={showEnableDialog} onOpenChange={closeEnableDialog}>
                <DialogContent className="sm:max-w-md">
                    {setupStep === 'qr' && qrData && (
                        <>
                            <DialogHeader>
                                <DialogTitle>{t('security.dialogs.enableTitle')}</DialogTitle>
                                <DialogDescription>
                                    {t('security.dialogs.enableDescription')}
                                </DialogDescription>
                            </DialogHeader>
                            <div className="flex flex-col items-center gap-4 py-4">
                                <QRCode
                                    value={qrData.qr_code_url}
                                    size={240}
                                    ecLevel="M"
                                    bgColor="transparent"
                                    fgColor={theme === 'dark' ? '#ffffff' : '#0f172a'}
                                    qrStyle="dots"
                                    eyeRadius={6}
                                    quietZone={0}
                                />
                                <div className="w-full">
                                    <p className="text-sm text-muted-foreground mb-2 text-center">
                                        {t('security.dialogs.manualCode')}
                                    </p>
                                    <div className="flex items-center gap-2 bg-muted p-2 rounded-md">
                                        <code className="flex-1 text-sm font-mono break-all">
                                            {qrData.secret}
                                        </code>
                                        <Button variant="ghost" size="sm" onClick={copySecret}>
                                            <Copy className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </div>
                            </div>
                            <DialogFooter>
                                <Button onClick={() => setSetupStep('verify')}>
                                    {tCommon('actions.continue')}
                                </Button>
                            </DialogFooter>
                        </>
                    )}

                    {setupStep === 'verify' && (
                        <>
                            <DialogHeader>
                                <DialogTitle>{t('security.dialogs.verifyTitle')}</DialogTitle>
                                <DialogDescription>
                                    {t('security.dialogs.verifyDescription')}
                                </DialogDescription>
                            </DialogHeader>
                            <div className="flex justify-center py-6">
                                <InputOTP
                                    maxLength={6}
                                    value={otpValue}
                                    onChange={setOtpValue}
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
                            <DialogFooter className="gap-2 sm:gap-0">
                                <Button variant="outline" onClick={() => setSetupStep('qr')}>
                                    {tCommon('actions.back')}
                                </Button>
                                <Button
                                    onClick={handleConfirm}
                                    disabled={otpValue.length !== 6 || confirmMutation.isPending}
                                >
                                    {confirmMutation.isPending ? t('security.dialogs.verifying') : t('security.dialogs.verify')}
                                </Button>
                            </DialogFooter>
                        </>
                    )}

                    {setupStep === 'recovery' && recoveryCodes.length > 0 && (
                        <>
                            <DialogHeader>
                                <DialogTitle>{t('security.dialogs.recoveryTitle')}</DialogTitle>
                                <DialogDescription>
                                    {t('security.dialogs.recoveryDescription')}
                                </DialogDescription>
                            </DialogHeader>
                            <div className="py-4">
                                <div className="bg-muted p-4 rounded-md font-mono text-sm grid grid-cols-2 gap-2">
                                    {recoveryCodes.map((code, index) => (
                                        <div key={index} className="text-center">
                                            {code}
                                        </div>
                                    ))}
                                </div>
                                <Button
                                    variant="outline"
                                    className="w-full mt-4"
                                    onClick={copyRecoveryCodes}
                                >
                                    <Copy className="h-4 w-4 mr-2" />
                                    {t('security.dialogs.copyCodes')}
                                </Button>
                            </div>
                            <DialogFooter>
                                <Button onClick={closeEnableDialog}>
                                    {tCommon('actions.done')}
                                </Button>
                            </DialogFooter>
                        </>
                    )}
                </DialogContent>
            </Dialog>

            <Dialog open={showDisableDialog} onOpenChange={setShowDisableDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{t('security.dialogs.disableTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('security.dialogs.disableDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="flex justify-center py-6">
                        <InputOTP
                            maxLength={6}
                            value={otpValue}
                            onChange={setOtpValue}
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
                    <DialogFooter className="gap-2 sm:gap-0">
                        <Button variant="outline" onClick={() => {
                            setShowDisableDialog(false)
                            setOtpValue('')
                        }}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={handleDisable}
                            disabled={otpValue.length !== 6 || disableMutation.isPending}
                        >
                            {disableMutation.isPending ? t('security.dialogs.disabling') : t('security.dialogs.disable')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={showRegenerateDialog} onOpenChange={setShowRegenerateDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{t('security.recovery.regenerateTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('security.recovery.regenerateDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="flex justify-center py-6">
                        <InputOTP
                            maxLength={6}
                            value={otpValue}
                            onChange={setOtpValue}
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
                    <DialogFooter className="gap-2 sm:gap-0">
                        <Button variant="outline" onClick={() => {
                            setShowRegenerateDialog(false)
                            setOtpValue('')
                        }}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button
                            onClick={handleRegenerate}
                            disabled={otpValue.length !== 6 || regenerateMutation.isPending}
                        >
                            {regenerateMutation.isPending ? t('security.recovery.regenerating') : t('security.recovery.regenerate')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={showLogoutOthersDialog} onOpenChange={setShowLogoutOthersDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{t('security.sessions.confirmTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('security.sessions.confirmDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter className="gap-2 sm:gap-0">
                        <Button variant="outline" onClick={() => setShowLogoutOthersDialog(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            disabled={logoutOthers.isPending}
                            onClick={async () => {
                                try {
                                    await logoutOthers.mutateAsync()
                                    setShowLogoutOthersDialog(false)
                                } catch {
                                    // toasted by the mutation
                                }
                            }}
                        >
                            {t('security.sessions.confirm')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={showAddPasskeyDialog} onOpenChange={(open) => {
                setShowAddPasskeyDialog(open)
                if (!open) setPasskeyName('')
            }}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{t('security.passkeys.addTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('security.passkeys.addDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="py-2">
                        <Input
                            placeholder={t('security.passkeys.namePlaceholder')}
                            value={passkeyName}
                            onChange={(e) => setPasskeyName(e.target.value)}
                            maxLength={100}
                            autoFocus
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' && !registerPasskey.isPending) handleAddPasskey()
                            }}
                        />
                    </div>
                    <DialogFooter className="gap-2 sm:gap-0">
                        <Button variant="outline" onClick={() => {
                            setShowAddPasskeyDialog(false)
                            setPasskeyName('')
                        }}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button onClick={handleAddPasskey} disabled={registerPasskey.isPending}>
                            {registerPasskey.isPending ? t('security.passkeys.waiting') : tCommon('actions.continue')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </Page>
    )
}
