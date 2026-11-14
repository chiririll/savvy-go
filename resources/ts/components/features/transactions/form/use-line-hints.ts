import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { lineCoverage, linesToAdd, paymentCurrencies } from '@/lib/transaction/transaction-lines'
import { formatCurrency } from '@/lib/utils'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'

interface LineHintsOptions {
    form: UseFormReturn<TransactionFormValues>
    accounts?: Account[]
    payments?: TransactionFormValues['payments']
    change?: TransactionFormValues['change']
    lines?: TransactionFormValues['lines']
    defaultCategoryId?: number | null
}

/** How far the lines cover the payments (the default category takes the rest), and what to tell the user about it. */
export function useLineHints({ form, accounts, payments, change, lines, defaultCategoryId }: LineHintsOptions) {
    const { t } = useTranslation('forms')
    const values = { payments: payments ?? [], change: change ?? [], lines: lines ?? [] }
    const coverage = lineCoverage(values, accounts ?? [])
    const hasLines = Boolean(lines?.length)
    // What the lines leave uncovered goes to the default category: said where that category is chosen.
    let uncovered: string | null = null
    if (hasLines && coverage.remainder > 0 && coverage.currency) {
        const amount = formatCurrency(coverage.remainder, coverage.currency)
        uncovered = defaultCategoryId
            ? t('transactions.uncovered', { amount })
            : t('transactions.uncoveredNoCategory', { amount })
    }

    return {
        coverage,
        lineCurrencies: paymentCurrencies(values, accounts ?? []),
        hasLines,
        uncovered,
        nextLine: () => linesToAdd(form.getValues(), accounts ?? []),
    }
}
