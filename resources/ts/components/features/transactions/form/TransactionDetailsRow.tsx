import type { Control } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import { formatDateLocal } from '@/lib/utils'
import { TransactionCategoryField } from './TransactionCategoryField'
import { TransactionDateField } from './TransactionDateField'
import { TransferRateField } from './TransferRateField'

type RateProps = Parameters<typeof TransferRateField>[0]

interface TransactionDetailsRowProps {
    control: Control<TransactionFormValues>
    type: TransactionFormValues['type']
    rate: RateProps['rate']
    fromCode?: string
    toCode?: string
    dateRequired: boolean
    /** A transaction that already moved a balance cannot be dated in the future. */
    pastOnly?: boolean
}

/** The exchange rate (transfers) or the category, next to the date. */
export function TransactionDetailsRow({
    control,
    type,
    rate,
    fromCode,
    toCode,
    dateRequired,
    pastOnly,
}: TransactionDetailsRowProps) {
    return (
        <div className="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2">
            {type === 'transfer' ? (
                <TransferRateField rate={rate} fromCode={fromCode} toCode={toCode} />
            ) : (
                <TransactionCategoryField control={control} type={type as 'income' | 'expense'} />
            )}

            <TransactionDateField required={dateRequired} max={pastOnly ? formatDateLocal() : undefined} />
        </div>
    )
}
