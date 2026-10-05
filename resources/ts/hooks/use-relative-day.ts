import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { formatTransactionGroupHeading } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'

/** Formats a YYYY-MM-DD date as today / yesterday / tomorrow or a short date. */
export function useRelativeDay() {
    const { t, i18n } = useTranslation('pages')

    return useCallback((dateKey: string | null) => formatTransactionGroupHeading(
        dateKey?.slice(0, 10) ?? null,
        intlLocale(i18n.language),
        {
            today: t('transactions.today'),
            yesterday: t('transactions.yesterday'),
            tomorrow: t('transactions.tomorrow'),
            noDate: t('transactions.noDate'),
        },
    ), [t, i18n.language])
}
