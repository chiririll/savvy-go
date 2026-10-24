import { ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
import { Confirm } from '@/components/shared/dialogs/Confirm'

interface SkipTransactionAlertProps {
    onConfirm: () => void
    trigger: ReactElement
}

export function SkipTransactionAlert({ onConfirm, trigger }: SkipTransactionAlertProps) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
        <Confirm
            trigger={trigger}
            title={t('transactions.skipTitle')}
            description={t('transactions.skipDescription')}
            action={tCommon('actions.skip')}
            onConfirm={onConfirm}
        />
    )
}
