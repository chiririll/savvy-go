import type { Control } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import { formatDateLocal } from '@/lib/utils'
import { TransactionCategoryField } from './TransactionCategoryField'
import { TransactionDateField } from './TransactionDateField'

interface TransactionDetailsRowProps {
    control: Control<TransactionFormValues>
    type: TransactionFormValues['type']
    dateRequired: boolean
    /** A transaction that already moved a balance cannot be dated in the future. */
    pastOnly?: boolean
    /** The category field: see `TransactionCategoryField`. A transfer has none. */
    category?: Omit<React.ComponentProps<typeof TransactionCategoryField>, 'control' | 'type'>
}

/** The date and the category, side by side. */
export function TransactionDetailsRow({ control, type, dateRequired, pastOnly, category }: TransactionDetailsRowProps) {
    return (
        <div className="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2">
            <TransactionDateField required={dateRequired} max={pastOnly ? formatDateLocal() : undefined} />
            {(type === 'income' || type === 'expense') && (
                <TransactionCategoryField control={control} type={type} {...category} />
            )}
        </div>
    )
}
