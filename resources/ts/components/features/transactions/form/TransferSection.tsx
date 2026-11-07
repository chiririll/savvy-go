import { ArrowUpDown } from 'lucide-react'
import { useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { AccountSelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { swapTransfer } from '@/lib/transfer-payload'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { PaymentRow } from './PaymentRow'
import { BalanceAfter } from './TransactionBalanceHint'
import { TransferRateField } from './TransferRateField'
import type { useTransferRate } from './transfer-rate'

interface TransferSectionProps {
    from?: Account
    to?: Account
    rate: ReturnType<typeof useTransferRate>
    /** undefined: this space. null: a transfer to another space, none chosen yet. */
    destinationSpaceId?: number | null
    balances: BalancePreview[]
    /** Whether saving moves balances now; a pending transfer shows a hint instead of the balances. */
    showBalances: boolean
}

/**
 * The two accounts of a transfer and what leaves one and arrives on the other. Between two currencies the
 * destination has an amount of its own and there is a rate; in one currency the amount is the same on both.
 */
export function TransferSection({ from, to, rate, destinationSpaceId, balances, showBalances }: TransferSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useFormContext<TransactionFormValues>()
    const toAccountId = useWatch({ control: form.control, name: 'to_account_id' })
    const isSpaceTransfer = destinationSpaceId !== undefined

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
    const fromBalance = balanceOf(from)
    const toBalance = isSpaceTransfer ? undefined : balanceOf(to)
    // The destination has an amount of its own only when its currency is not the source's.
    const ownAmount = Boolean(from && to && !rate.sameCurrency)

    return (
        <div className="space-y-3">
            <PaymentRow
                control={form.control}
                list="payments"
                index={0}
                currency={from?.currency}
                excludeIds={Number(toAccountId) > 0 && !isSpaceTransfer ? [Number(toAccountId)] : undefined}
                amountLabel={t('forms:transactions.sendAmount')}
                accountLabel={t('forms:fromAccount')}
                action={
                    isSpaceTransfer ? undefined : (
                        <Button
                            type="button"
                            variant="outline"
                            size="icon-sm"
                            className="shrink-0 rounded-full"
                            onClick={swap}
                            aria-label={t('forms:transactions.swapAccounts')}
                            title={t('forms:transactions.swapAccounts')}
                        >
                            <ArrowUpDown className="size-4" />
                        </Button>
                    )
                }
            >
                {showBalances && fromBalance && <BalanceAfter preview={fromBalance} />}
            </PaymentRow>

            <div className="space-y-1">
                <div className="grid min-w-0 grid-cols-[minmax(6.5rem,0.8fr)_minmax(0,1fr)] gap-2 sm:gap-4">
                    {ownAmount && (
                        <FormField
                            control={form.control}
                            name="to_amount"
                            render={({ field }) => (
                                <FormItem className="min-w-0">
                                    <FormLabel>{t('forms:transactions.receiveAmount')}</FormLabel>
                                    <div className="relative">
                                        <FormControl>
                                            <Input
                                                type="number"
                                                inputMode="decimal"
                                                step={to?.currency?.decimals ? 10 ** -to.currency.decimals : 1}
                                                min={0}
                                                placeholder="0.00"
                                                className="pr-12"
                                                {...field}
                                                value={field.value ?? ''}
                                                onChange={(event) =>
                                                    field.onChange(event.target.value ? Number(event.target.value) : null)
                                                }
                                            />
                                        </FormControl>
                                        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground">
                                            {to?.currency?.code}
                                        </span>
                                    </div>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    )}
                    <FormField
                        control={form.control}
                        name="to_account_id"
                        render={({ field }) => (
                            <FormItem className={ownAmount ? 'min-w-0' : 'col-span-2 min-w-0'}>
                                <FormLabel>{t('forms:transactions.toAccount')}</FormLabel>
                                <AccountSelect
                                    value={field.value}
                                    onChange={field.onChange}
                                    excludeId={isSpaceTransfer ? undefined : from?.id}
                                    spaceId={destinationSpaceId}
                                />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>
                {showBalances && toBalance && <BalanceAfter preview={toBalance} />}
            </div>

            {rate.editable && (
                <div className="grid grid-cols-2 gap-2 sm:gap-4">
                    <TransferRateField rate={rate} fromCode={from?.currency?.code} toCode={to?.currency?.code} />
                </div>
            )}
        </div>
    )
}
