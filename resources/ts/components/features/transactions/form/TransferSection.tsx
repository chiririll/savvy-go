import { ArrowUpDown } from 'lucide-react'
import { useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { swapTransfer } from '@/lib/transaction/transfer-payload'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { AmountAccountRow, RowLabels } from './AmountAccountRow'
import { useTemplateMode } from './template-mode'
import { BalanceShort, type BalanceMode } from './TransactionBalanceHint'
import type { useTransferRate } from './transfer-rate'

interface TransferSectionProps {
    from?: Account
    to?: Account
    rate: ReturnType<typeof useTransferRate>
    /** undefined: this space. null: a transfer to another space, none chosen yet. */
    destinationSpaceId?: number | null
    balances: BalancePreview[]
    balanceMode: BalanceMode
}

/**
 * The two accounts of a transfer, in two columns and two rows: the amount and the account that sends, then the amount
 * and the account that receives. The swap button stands at the end of both rows. Between two currencies the
 * destination has an amount of its own; in one currency it shows the amount sent and cannot be changed. The rate
 * stands in the row of the date: see `TransferRateField`.
 */
export function TransferSection({ from, to, rate, destinationSpaceId, balances, balanceMode }: TransferSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useFormContext<TransactionFormValues>()
    const toAccountId = useWatch({ control: form.control, name: 'to_account_id' })
    const sent = useWatch({ control: form.control, name: 'payments.0.amount' })
    const isSpaceTransfer = destinationSpaceId !== undefined
    const template = useTemplateMode()

    const swap = () => {
        const updates = swapTransfer({
            payments: form.getValues('payments') as Array<{ account_id: number; amount: number | null }>,
            to_account_id: form.getValues('to_account_id'),
            to_amount: form.getValues('to_amount'),
            exchange_rate: form.getValues('exchange_rate'),
        })
        if (!updates) {
            return
        }
        rate.keepNextPair()
        const options = { shouldDirty: true, shouldValidate: false }
        form.setValue('payments.0.account_id', updates.account_id, options)
        form.setValue('payments.0.amount', updates.amount as number, options)
        form.setValue('to_account_id', updates.to_account_id, options)
        form.setValue('to_amount', updates.to_amount, options)
        form.setValue('exchange_rate', updates.exchange_rate, options)
    }

    const balanceOf = (account?: Account) => balances.find((row) => row.account.id === account?.id)
    // The destination has an amount of its own only when its currency is not the source's.
    const ownAmount = Boolean(from && to && !rate.sameCurrency)

    return (
        <div className="space-y-1.5">
            <RowLabels amount={t('forms:transactions.sendAmount')} account={t('forms:fromAccount')} />
            <AmountAccountRow
                control={form.control}
                amountName="payments.0.amount"
                accountName="payments.0.account_id"
                amountLabel={t('forms:transactions.sendAmount')}
                accountLabel={t('forms:fromAccount')}
                account={from}
                excludeIds={Number(toAccountId) > 0 && !isSpaceTransfer ? [Number(toAccountId)] : undefined}
                balance={balanceMode !== 'none' ? <BalanceShort account={from} preview={balanceOf(from)} mode={balanceMode} /> : undefined}
            />

            <RowLabels
                amount={t('forms:transactions.receiveAmount')}
                account={t('forms:transactions.toAccount')}
                accountEnd={
                    !isSpaceTransfer && (
                        // Lifted into the gap between the two rows; it takes no height, so the labels stay level with the others.
                        <Button
                            type="button"
                            variant="outline"
                            size="icon-sm"
                            className="-my-1.5 shrink-0 -translate-y-3 rounded-full"
                            onClick={swap}
                            aria-label={t('forms:transactions.swapAccounts')}
                            title={t('forms:transactions.swapAccounts')}
                        >
                            <ArrowUpDown className="size-4" />
                        </Button>
                    )
                }
            />
            <AmountAccountRow
                control={form.control}
                amountName="to_amount"
                accountName="to_account_id"
                amountLabel={t('forms:transactions.receiveAmount')}
                accountLabel={t('forms:transactions.toAccount')}
                account={to}
                excludeId={isSpaceTransfer ? undefined : from?.id}
                spaceId={destinationSpaceId}
                fixedAmount={ownAmount ? undefined : (sent as string | number | null)}
                amountPlaceholder={template ? t('forms:recurring.toAmountPlaceholder') : undefined}
                balance={
                    balanceMode !== 'none' && !isSpaceTransfer ? (
                        <BalanceShort account={to} preview={balanceOf(to)} mode={balanceMode} />
                    ) : undefined
                }
            />
        </div>
    )
}
