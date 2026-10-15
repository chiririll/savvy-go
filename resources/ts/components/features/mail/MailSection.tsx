import { useState, type ReactNode } from 'react'
import { Mail } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsSection, SettingsRowSkeleton, ToggleRow } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useMailStatus, useSendTestMail, useUpdateMail } from '@/hooks'
import { mailProtocolLabel } from '@/lib/labels'
import { MAIL_PROTOCOLS, type MailInput, type MailProtocol, type MailStatus } from '@/types/mail'

/** Default port of each connection type, offered when the protocol changes. */
/** Stand-in shown for a stored password: the real one is never sent to the browser. */
const PASSWORD_DOTS = '••••••'

const DEFAULT_PORTS: Record<MailProtocol, number> = { starttls: 587, smtps: 465, smtp: 25 }

interface Draft {
    enabled: boolean
    host: string
    port: string
    protocol: MailProtocol
    user: string
    /** Empty keeps the stored password. */
    password: string
    from: string
    skipVerify: boolean
    timeout: string
}

const draftOf = (mail: MailStatus): Draft => ({
    enabled: mail.enabled,
    host: mail.host,
    port: String(mail.port),
    protocol: mail.protocol,
    user: mail.user,
    password: '',
    from: mail.from,
    skipVerify: mail.skipVerify,
    timeout: String(mail.timeout),
})

function Field({ id, label, hint, children }: { id: string; label: string; hint?: string; children: ReactNode }) {
    return (
        <div className="grid gap-1.5 px-4 py-3">
            <Label htmlFor={id} className="text-sm font-medium">{label}</Label>
            {children}
            {hint && <p className="text-sm leading-relaxed text-muted-foreground">{hint}</p>}
        </div>
    )
}

/** The mail server, kept in the [mailer] section of config.toml: edit it here, then try it with a test email. */
export function MailSection() {
    const { data: mail, isLoading } = useMailStatus()
    return isLoading || !mail ? <MailShell><SettingsRowSkeleton /></MailShell> : <MailForm mail={mail} />
}

function MailShell({ children }: { children: ReactNode }) {
    const { t } = useTranslation('settings')
    return (
        <SettingsSection icon={Mail} title={t('system.mail.title')} description={t('system.mail.description')}>
            <div className="divide-y rounded-lg border">{children}</div>
        </SettingsSection>
    )
}

function MailForm({ mail }: { mail: MailStatus }) {
    const { t } = useTranslation('settings')
    const update = useUpdateMail()
    const sendTest = useSendTestMail()
    const [draft, setDraft] = useState(() => draftOf(mail))
    const [to, setTo] = useState('')
    // While false and a password is stored, the field shows dots instead of an (empty) draft.
    const [editingPassword, setEditingPassword] = useState(false)

    const locked = !mail.writable
    const set = <K extends keyof Draft>(key: K, value: Draft[K]) => setDraft((d) => ({ ...d, [key]: value }))

    const input = (): MailInput => ({
        enabled: draft.enabled,
        host: draft.host.trim(),
        port: Number(draft.port),
        protocol: draft.protocol,
        user: draft.user.trim(),
        // Left empty, the stored password stays. Without a login there is no use for one, so it is cleared.
        ...(draft.password !== '' || draft.user.trim() === '' ? { password: draft.password } : {}),
        from: draft.from.trim(),
        skipVerify: draft.skipVerify,
        timeout: Number(draft.timeout),
    })

    const status = mail.enabled ? t('system.mail.working') : mail.problem ? t('system.mail.problem', { problem: mail.problem }) : t('system.mail.off')

    return (
        <MailShell>
            <p className={`px-4 py-3 text-sm ${mail.enabled ? '' : mail.problem ? 'text-destructive' : 'text-muted-foreground'}`}>{status}</p>
            <form
                className="divide-y"
                onSubmit={(e) => {
                    e.preventDefault()
                    update.mutate(input(), {
                        onSuccess: (saved) => {
                            setDraft(draftOf(saved))
                            setEditingPassword(false)
                        },
                    })
                }}
            >
                <ToggleRow
                    id="mail-enabled"
                    label={t('system.mail.enabled')}
                    description={t('system.mail.enabledDescription')}
                    checked={draft.enabled}
                    disabled={locked}
                    onChange={(v) => set('enabled', v)}
                />
                <Field id="mail-host" label={t('system.mail.host')}>
                    <Input id="mail-host" value={draft.host} disabled={locked} placeholder="smtp.example.com" onChange={(e) => set('host', e.target.value)} />
                </Field>
                <Field id="mail-protocol" label={t('system.mail.protocol')}>
                    <Select
                        value={draft.protocol}
                        disabled={locked}
                        onValueChange={(v) => {
                            const protocol = v as MailProtocol
                            // Follow the usual port, unless the admin typed a different one.
                            const usual = Object.values(DEFAULT_PORTS).map(String).includes(draft.port)
                            setDraft((d) => ({ ...d, protocol, port: usual ? String(DEFAULT_PORTS[protocol]) : d.port }))
                        }}
                    >
                        <SelectTrigger id="mail-protocol" className="w-full">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {MAIL_PROTOCOLS.map((p) => (
                                <SelectItem key={p} value={p}>{mailProtocolLabel(t, p)}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </Field>
                <Field id="mail-port" label={t('system.mail.port')}>
                    <Input id="mail-port" type="number" min={1} max={65535} value={draft.port} disabled={locked} onChange={(e) => set('port', e.target.value)} />
                </Field>
                <Field id="mail-user" label={t('system.mail.user')} hint={t('system.mail.userHint')}>
                    <Input id="mail-user" value={draft.user} disabled={locked} autoComplete="off" onChange={(e) => set('user', e.target.value)} />
                </Field>
                <Field id="mail-password" label={t('system.mail.password')} hint={t('system.mail.passwordHint')}>
                    <Input
                        id="mail-password"
                        type="password"
                        value={mail.passwordSet && !editingPassword ? PASSWORD_DOTS : draft.password}
                        disabled={locked}
                        autoComplete="new-password"
                        placeholder={mail.passwordSet ? t('system.mail.passwordKeep') : ''}
                        // Focusing a stored password clears the dots; leaving it empty brings them back.
                        onFocus={() => setEditingPassword(true)}
                        onBlur={() => draft.password === '' && setEditingPassword(false)}
                        onChange={(e) => set('password', e.target.value)}
                    />
                </Field>
                <Field id="mail-from" label={t('system.mail.from')} hint={t('system.mail.fromHint')}>
                    <Input id="mail-from" value={draft.from} disabled={locked} placeholder="Go Savvy <savvy@example.com>" onChange={(e) => set('from', e.target.value)} />
                </Field>
                {draft.protocol !== 'smtp' && (
                    <ToggleRow
                        id="mail-skip-verify"
                        label={t('system.mail.skipVerify')}
                        description={t('system.mail.skipVerifyDescription')}
                        checked={draft.skipVerify}
                        disabled={locked}
                        onChange={(v) => set('skipVerify', v)}
                    />
                )}
                <Field id="mail-timeout" label={t('system.mail.timeout')} hint={t('system.mail.timeoutHint')}>
                    <Input id="mail-timeout" type="number" min={1} value={draft.timeout} disabled={locked} onChange={(e) => set('timeout', e.target.value)} />
                </Field>
                {locked && <p className="px-4 py-3 text-sm leading-relaxed text-muted-foreground">{t('system.mail.configReadOnly')}</p>}
                {!locked && (
                    <div className="px-4 py-3.5">
                        <Button type="submit" disabled={update.isPending}>{t('actions.save', { ns: 'common' })}</Button>
                    </div>
                )}
            </form>
            {mail.enabled && !mail.appUrlSet && <p className="px-4 py-3 text-sm text-destructive">{t('system.mail.noAppUrl')}</p>}
            {mail.enabled && (
                <form
                    className="flex flex-col gap-2 px-4 py-3.5 sm:flex-row sm:items-end"
                    onSubmit={(e) => {
                        e.preventDefault()
                        sendTest.mutate(to.trim() || undefined)
                    }}
                >
                    <div className="grid flex-1 gap-1.5">
                        <Label htmlFor="mail-test-to" className="text-sm font-medium">{t('system.mail.testTitle')}</Label>
                        <Input
                            id="mail-test-to"
                            type="email"
                            placeholder={t('system.mail.testPlaceholder')}
                            value={to}
                            onChange={(e) => setTo(e.target.value)}
                        />
                    </div>
                    <Button type="submit" variant="outline" disabled={sendTest.isPending}>{t('system.mail.testSend')}</Button>
                </form>
            )}
        </MailShell>
    )
}
