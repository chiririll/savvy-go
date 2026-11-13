import { Scale } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'

/** Opens the reconcile dialog; the header of the Accounts section shows it on both tabs. */
export function ReconcileButton({ onClick }: { onClick: () => void }) {
    const { t } = useTranslation('pages')
    return (
        <Button variant="outline" onClick={onClick} aria-label={t('accounts.reconcile.button')} {...testId('reconcile')}>
            <Scale className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">{t('accounts.reconcile.button')}</span>
        </Button>
    )
}
