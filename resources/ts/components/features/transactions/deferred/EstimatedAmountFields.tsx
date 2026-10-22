import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { Transaction } from '@/types'

interface AmountFieldProps {
    id: string
    label: string
    symbol?: string | null
    value: string
    onChange: (value: string) => void
}

function AmountField({ id, label, symbol, value, onChange }: AmountFieldProps) {
    return (
        <div className="grid gap-1.5">
            <Label htmlFor={id}>
                {label}
                {symbol ? ` (${symbol})` : ''}
            </Label>
            <Input
                id={id}
                type="number"
                inputMode="decimal"
                step="any"
                min="0"
                value={value}
                onChange={(event) => onChange(event.target.value)}
                required
            />
        </div>
    )
}

interface EstimatedAmountFieldsProps {
    idPrefix: string
    transaction: Transaction | null
    crossCurrency: boolean
    amount: string
    toAmount: string
    onAmountChange: (value: string) => void
    onToAmountChange: (value: string) => void
}

/** The real amounts of an estimated transaction, asked for when it is applied. */
export function EstimatedAmountFields({
    idPrefix,
    transaction,
    crossCurrency,
    amount,
    toAmount,
    onAmountChange,
    onToAmountChange,
}: EstimatedAmountFieldsProps) {
    const { t } = useTranslation('pages')

    return (
        <div className="grid gap-3 rounded-md border border-dashed p-3">
            <p className="text-xs text-muted-foreground">{t('transactions.applyEstimatedDescription')}</p>
            <AmountField
                id={`${idPrefix}-amount`}
                label={t('transactions.applyAmount')}
                symbol={transaction?.account.currency?.symbol}
                value={amount}
                onChange={onAmountChange}
            />
            {crossCurrency && (
                <AmountField
                    id={`${idPrefix}-to-amount`}
                    label={t('transactions.applyToAmount')}
                    symbol={transaction?.toAccount?.currency?.symbol}
                    value={toAmount}
                    onChange={onToAmountChange}
                />
            )}
        </div>
    )
}
