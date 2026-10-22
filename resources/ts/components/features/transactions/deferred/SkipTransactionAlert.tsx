import { ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
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

interface SkipTransactionAlertProps {
    onConfirm: () => void
    trigger: ReactElement
}

export function SkipTransactionAlert({ onConfirm, trigger }: SkipTransactionAlertProps) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
        <AlertDialog>
            <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('transactions.skipTitle')}</AlertDialogTitle>
                    <AlertDialogDescription>
                        {t('transactions.skipDescription')}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={onConfirm}>
                        {tCommon('actions.skip')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
