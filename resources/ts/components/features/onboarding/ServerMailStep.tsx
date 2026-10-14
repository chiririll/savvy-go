import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { MailSection } from '@/components/features/mail'
import { serverSetupPaths } from './server-setup-paths'

/** Server setup: the mail form (saved to config.toml, live at once) and a test email. */
export function ServerMailStep() {
    const { t } = useTranslation('auth')
    return (
        <div className="space-y-4">
            <MailSection />
            <p className="text-sm text-muted-foreground">{t('serverSetup.mail.docs')}</p>
            <Button asChild className="w-full">
                <Link to={serverSetupPaths.options}>{t('serverSetup.backToList')}</Link>
            </Button>
        </div>
    )
}
