import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { Currency, Transaction } from '@/types'
import { linesFollowPayments, type AmountDraft } from './apply-deferred'

interface AmountFieldProps {
    id: string
    label: string
    /** Says in words what the sign does, for those who can't see it. */
    hint?: string
    sign?: '+' | '−'
    currency?: Currency | null
    value: string
    onChange: (value: string) => void
}

function AmountField({ id, label, hint, sign, currency, value, onChange }: AmountFieldProps) {
    const step = currency && currency.decimals > 0 ? 10 ** -currency.decimals : 'any'

    return (
        <div className="grid min-w-0 gap-1.5">
            <Label htmlFor={id} className="min-w-0">
                <span className="truncate">{label}</span>
            </Label>
            <div className="flex min-w-0 items-center gap-2">
                {sign && (
                    <span aria-hidden className="w-3 shrink-0 text-center text-muted-foreground">
                        {sign}
                    </span>
                )}
                <div className="relative min-w-0 flex-1">
                    <Input
                        id={id}
                        type="number"
                        inputMode="decimal"
                        step={step}
                        min="0"
                        value={value}
                        aria-describedby={hint ? `${id}-hint` : undefined}
                        className={currency ? 'pr-14' : undefined}
                        onChange={(event) => onChange(event.target.value)}
                        required
                    />
                    {currency && (
                        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground">
                            {currency.code}
                        </span>
                    )}
                </div>
            </div>
            {hint && (
                <span id={`${id}-hint`} className="sr-only">
                    {hint}
                </span>
            )}
        </div>
    )
}

interface OccurrenceAmountFieldsProps {
    idPrefix: string
    transaction: Transaction
    draft: AmountDraft
    onChange: (draft: AmountDraft) => void
}

const replaceAt = (list: string[], index: number, value: string) => list.map((v, i) => (i === index ? value : v))

/**
 * The real amounts of an occurrence of a recurring rule, asked for when it is confirmed: one field per payment, and
 * one per line unless the only line follows the payments.
 */
export function OccurrenceAmountFields({ idPrefix, transaction, draft, onChange }: OccurrenceAmountFieldsProps) {
    const { t } = useTranslation('pages')
    const lines = [...transaction.lines].sort((a, b) => a.seq - b.seq)
    const showLines = lines.length > 0 && !linesFollowPayments(transaction)

    return (
        <div className="grid gap-3 rounded-md border border-dashed p-3">
            <p className="text-xs text-muted-foreground">{t('transactions.applyOccurrenceDescription')}</p>
            <div className="grid gap-3" role="group" aria-label={t('transactions.applyPaymentsTitle')}>
                {transaction.payments.map((p, index) => (
                    <AmountField
                        key={`${p.accountId}-${index}`}
                        id={`${idPrefix}-payment-${index}`}
                        label={p.account?.name ?? String(p.accountId)}
                        sign={p.amount < 0 ? '−' : '+'}
                        hint={p.amount < 0 ? t('transactions.applyTakes') : t('transactions.applyBrings')}
                        currency={p.account?.currency}
                        value={draft.payments[index] ?? ''}
                        onChange={(value) => onChange({ ...draft, payments: replaceAt(draft.payments, index, value) })}
                    />
                ))}
            </div>
            {showLines && (
                <div className="grid gap-3" role="group" aria-label={t('transactions.applyLinesTitle')}>
                    {lines.map((l, index) => (
                        <AmountField
                            key={l.seq}
                            id={`${idPrefix}-line-${index}`}
                            label={l.category?.name ?? t('transactions.applyLinesTitle')}
                            currency={
                                transaction.payments.find((p) => p.account?.currency?.id === l.currencyId)?.account
                                    ?.currency
                            }
                            value={draft.lines[index] ?? ''}
                            onChange={(value) => onChange({ ...draft, lines: replaceAt(draft.lines, index, value) })}
                        />
                    ))}
                </div>
            )}
        </div>
    )
}
