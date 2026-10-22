import { useState } from 'react'
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
} from '@/components/ui/alert-dialog'
import { formatCurrency } from '@/lib/currency'
import type { NegativeBalanceWarning } from '@/lib/negative-balance'

interface NegativeBalanceConfirmDialogProps {
    open: boolean
    warnings: NegativeBalanceWarning[]
    onOpenChange: (open: boolean) => void
    onConfirm: () => void
}

export function NegativeBalanceConfirmDialog({
    open,
    warnings,
    onOpenChange,
    onConfirm,
}: NegativeBalanceConfirmDialogProps) {
    const { t } = useTranslation('common')

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('negativeBalance.title')}</AlertDialogTitle>
                    <AlertDialogDescription asChild>
                        <div className="space-y-3">
                            <p>
                                {warnings.length === 1
                                    ? t('negativeBalance.descriptionOne', { name: warnings[0].accountName })
                                    : t('negativeBalance.description')}
                            </p>
                            <ul className="space-y-2">
                                {warnings.map((warning) => (
                                    <li key={warning.accountName} className="text-foreground text-sm">
                                        <div className="font-medium">{warning.accountName}</div>
                                        <div className="text-muted-foreground font-mono text-xs">
                                            {t('negativeBalance.current')}:{' '}
                                            {formatCurrency(warning.currentBalance, warning.currency)}
                                            {' → '}
                                            {t('negativeBalance.resulting')}:{' '}
                                            <span className="text-destructive">
                                                {formatCurrency(warning.resultingBalance, warning.currency)}
                                            </span>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>{t('actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={onConfirm}>
                        {t('actions.confirm')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}

export function useNegativeBalanceConfirm<T>() {
    const [pending, setPending] = useState<{
        payload: T
        warnings: NegativeBalanceWarning[]
        proceed: (payload: T) => void
    } | null>(null)

    const confirmIfNeeded = (
        payload: T,
        warnings: NegativeBalanceWarning[],
        proceed: (payload: T) => void,
    ) => {
        if (warnings.length === 0) {
            proceed(payload)
            return
        }

        setPending({ payload, warnings, proceed })
    }

    const dialog = (
        <NegativeBalanceConfirmDialog
            open={pending !== null}
            warnings={pending?.warnings ?? []}
            onOpenChange={(open) => {
                if (!open) {
                    setPending(null)
                }
            }}
            onConfirm={() => {
                if (!pending) {
                    return
                }

                const { payload, proceed } = pending
                setPending(null)
                proceed(payload)
            }}
        />
    )

    return { confirmIfNeeded, dialog }
}
