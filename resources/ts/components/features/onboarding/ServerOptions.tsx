import type { ComponentType } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Check, ChevronRight, KeyRound, Loader2, Mail } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useIdentityProviders, useMailStatus } from '@/hooks'
import { serverSetupPaths } from './server-setup-paths'

interface OptionProps {
    to: string
    icon: ComponentType<{ className?: string }>
    title: string
    description: string
    done: boolean
}

function Option({ to, icon: Icon, title, description, done }: OptionProps) {
    const { t } = useTranslation('auth')
    return (
        <Link
            to={to}
            className="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
            <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50 text-muted-foreground">
                <Icon className="size-4" />
            </div>
            <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <span className="text-sm font-medium">{title}</span>
                    {done && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-1.5 py-0.5 text-[11px] font-medium text-primary">
                            <Check className="size-3" />
                            {t('serverSetup.done')}
                        </span>
                    )}
                </div>
                <p className="text-xs text-muted-foreground">{description}</p>
            </div>
            <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
        </Link>
    )
}

interface ServerOptionsProps {
    busy: boolean
    onFinish: () => void
}

/** Server setup, the optional steps: each opens on its own screen and comes back here. */
export function ServerOptions({ busy, onFinish }: ServerOptionsProps) {
    const { t } = useTranslation('auth')
    const { data: mail } = useMailStatus()
    const { data: providers } = useIdentityProviders()
    const mailDone = mail?.enabled === true
    const ssoDone = (providers?.length ?? 0) > 0

    return (
        <div className="space-y-4">
            <div className="space-y-2">
                <Option
                    to={serverSetupPaths.mail}
                    icon={Mail}
                    title={t('serverSetup.mail.title')}
                    description={t('serverSetup.mail.description')}
                    done={mailDone}
                />
                <Option
                    to={serverSetupPaths.sso}
                    icon={KeyRound}
                    title={t('serverSetup.sso.title')}
                    description={t('serverSetup.sso.description')}
                    done={ssoDone}
                />
            </div>
            <div className="flex flex-col gap-2">
                <Button type="button" className="w-full" disabled={busy} onClick={onFinish}>
                    {busy && <Loader2 className="mr-2 size-4 animate-spin" />}
                    {mailDone || ssoDone ? t('serverSetup.finish') : t('serverSetup.skip')}
                </Button>
                <Button asChild variant="ghost" className="w-full">
                    <Link to={serverSetupPaths.url}>{t('serverSetup.back')}</Link>
                </Button>
            </div>
        </div>
    )
}
