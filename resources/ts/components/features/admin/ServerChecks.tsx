import { Link } from 'react-router-dom'
import { AlertTriangle, CircleAlert, Info, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { checkText } from '@/lib/labels'
import { cn } from '@/lib/utils'
import { useServerChecks } from '@/hooks/use-server-checks'
import type { CheckId, CheckSeverity } from '@/types/checks'

/** The admin page that fixes each check. */
export const CHECK_LINKS: Record<CheckId, string> = {
    app_url_missing: '/admin/system',
    mail_off: '/admin/system',
    mail_invalid: '/admin/system',
    backup_missing: '/admin/backups',
    backup_stale: '/admin/backups',
    disk_filling: '/admin/monitoring',
    config_read_only: '/admin/system',
}

const ICONS: Record<CheckSeverity, LucideIcon> = {
    error: CircleAlert,
    warning: AlertTriangle,
    info: Info,
}

const TONES: Record<CheckSeverity, string> = {
    error: 'border-red-500/40 text-red-600 dark:text-red-400',
    warning: 'border-amber-500/40 text-amber-600 dark:text-amber-400',
    info: '',
}

/** Problems the server found with itself, or a note that there are none. */
export function ServerChecks({ className }: { className?: string }) {
    const { t } = useTranslation('settings')
    const { data } = useServerChecks()

    if (!data) return null
    if (data.length === 0) {
        return (
            <p className={cn('rounded-lg border px-4 py-10 text-center text-sm text-muted-foreground', className)}>
                {t('checks.allGood')}
            </p>
        )
    }

    return (
        <section aria-label={t('checks.title')} className={cn('space-y-3', className)}>
            {data.map((check) => {
                const Icon = ICONS[check.severity]
                const text = checkText(t, check.id, check.value)
                return (
                    <Alert key={check.id} className={TONES[check.severity]}>
                        <Icon />
                        <AlertTitle>{text.title}</AlertTitle>
                        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
                            <span>{text.description}</span>
                            <Button asChild size="sm" variant="outline">
                                <Link to={CHECK_LINKS[check.id]}>{text.action}</Link>
                            </Button>
                        </AlertDescription>
                    </Alert>
                )
            })}
        </section>
    )
}
