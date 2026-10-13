import { useState, type ReactNode } from 'react'
import { Mail } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsSection, SettingsRowSkeleton } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMailStatus, useSendTestMail } from '@/hooks'
import { mailProtocolLabel } from '@/lib/labels'

function Row({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
            <span className="text-sm font-medium">{label}</span>
            <span className="min-w-0 break-words text-sm text-muted-foreground sm:text-right">{children}</span>
        </div>
    )
}

/** The mail server from config.toml: read-only, with a button to try it. */
export function MailSection() {
    const { t } = useTranslation('settings')
    const { data: mail, isLoading } = useMailStatus()
    const sendTest = useSendTestMail()
    const [to, setTo] = useState('')

    const status = !mail ? '' : mail.enabled ? t('system.mail.working') : mail.problem ? t('system.mail.problem', { problem: mail.problem }) : t('system.mail.off')

    return (
        <SettingsSection icon={Mail} title={t('system.mail.title')} description={t('system.mail.description')}>
            <div className="divide-y rounded-lg border">
                {isLoading || !mail ? (
                    <SettingsRowSkeleton />
                ) : (
                    <>
                        <Row label={t('system.mail.status')}>
                            <span className={mail.enabled ? 'text-foreground' : mail.problem ? 'text-destructive' : undefined}>{status}</span>
                        </Row>
                        {(mail.host || mail.enabled) && (
                            <>
                                <Row label={t('system.mail.server')}>
                                    {mail.host}:{mail.port} · {mailProtocolLabel(t, mail.protocol)}
                                </Row>
                                <Row label={t('system.mail.user')}>{mail.user || t('system.mail.noUser')}</Row>
                                <Row label={t('system.mail.password')}>
                                    {mail.passwordSet ? t('system.mail.passwordSet') : t('system.mail.passwordNotSet')}
                                </Row>
                                <Row label={t('system.mail.from')}>{mail.from}</Row>
                                {mail.protocol !== 'smtp' && (
                                    <Row label={t('system.mail.certificate')}>
                                        {mail.skipVerify ? t('system.mail.unverified') : t('system.mail.verified')}
                                    </Row>
                                )}
                            </>
                        )}
                        {mail.enabled && !mail.appUrlSet && (
                            <p className="px-4 py-3 text-sm text-destructive">{t('system.mail.noAppUrl')}</p>
                        )}
                        <p className="px-4 py-3 text-sm leading-relaxed text-muted-foreground">{t('system.mail.configHint')}</p>
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
                                <Button type="submit" disabled={sendTest.isPending}>{t('system.mail.testSend')}</Button>
                            </form>
                        )}
                    </>
                )}
            </div>
        </SettingsSection>
    )
}
