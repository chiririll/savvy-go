import type { Control } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import { formatDateLocal } from '@/lib/utils'
import { FORM_ROW } from './form-grid'
import { TransactionCategoryField } from './TransactionCategoryField'
import { TransactionDateField } from './TransactionDateField'

interface TransactionDetailsRowProps {
    control: Control<TransactionFormValues>
    type: TransactionFormValues['type']
    dateRequired: boolean
    /** A template has no date: each occurrence is dated by its day. */
    withDate?: boolean
    /** A transaction that already moved a balance cannot be dated in the future. */
    pastOnly?: boolean
    /** The category field: see `TransactionCategoryField`. A transfer has none. */
    category?: Omit<React.ComponentProps<typeof TransactionCategoryField>, 'control' | 'type'>
    /** A field beside the date in place of the category: the rate of a transfer. */
    aside?: React.ReactNode
}

/** The date and the category (or the rate of a transfer), side by side; a template has the category alone. */
export function TransactionDetailsRow({ control, type, dateRequired, withDate = true, pastOnly, category, aside }: TransactionDetailsRowProps) {
    return (
        <div className={withDate || aside ? FORM_ROW : 'min-w-0'}>
            {withDate && <TransactionDateField required={dateRequired} max={pastOnly ? formatDateLocal() : undefined} />}
            {(type === 'income' || type === 'expense') && (
                <TransactionCategoryField control={control} type={type} {...category} />
            )}
            {aside}
        </div>
    )
}
