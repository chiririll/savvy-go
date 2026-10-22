import { useTranslation } from 'react-i18next'
import { SPACE_TRANSFER_OPTION, TRANSACTION_TYPE_OPTIONS } from '@/constants'
import { SegmentedChoice } from './SegmentedChoice'
import { transactionTypeLabel } from '@/lib/labels'

export type TransactionFormType = (typeof TRANSACTION_TYPE_OPTIONS)[number]['value']
export type TransactionFormKind = TransactionFormType | typeof SPACE_TRANSFER_OPTION.value

interface TransactionTypeTabsProps {
    value: TransactionFormKind
    onChange: (value: TransactionFormKind) => void
    /** Adds a square tab for a transfer to a linked space. */
    withSpaceTransfer?: boolean
}

export function TransactionTypeTabs({ value, onChange, withSpaceTransfer }: TransactionTypeTabsProps) {
    const { t } = useTranslation('pages')
    const options = [
        ...TRANSACTION_TYPE_OPTIONS,
        ...(withSpaceTransfer ? [{ ...SPACE_TRANSFER_OPTION, iconOnly: true }] : []),
    ]

    return (
        <SegmentedChoice<TransactionFormKind>
            value={value}
            onChange={onChange}
            options={options.map((option) => ({
                ...option,
                label: transactionTypeLabel(t, option.value),
            }))}
        />
    )
}
