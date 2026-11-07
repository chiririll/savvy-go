import { Plus } from 'lucide-react'
import { useFieldArray, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { changeTooLarge, nextFreeAccount } from '@/lib/transaction-payments'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { BalanceAfter } from './TransactionBalanceHint'
import { PaymentRow } from './PaymentRow'
import { PaymentRowButtons } from './PaymentRowButtons'

const EMPTY_AMOUNT = '' as unknown as number

interface ChangeSectionProps {
    accounts?: Account[]
    usedAccountIds: number[]
    balances: BalancePreview[]
    showBalances: boolean
}

/**
 * Money that came back to other accounts, drawn apart from the payments although it is a payment too, of the
 * opposite sign. Any account and any currency; an account is in a transaction once.
 */
export function ChangeSection({ accounts, usedAccountIds, balances, showBalances }: ChangeSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { control } = useFormContext<TransactionFormValues>()
    const { fields, append, remove } = useFieldArray({ control, name: 'change' })
    const [payments, change] = useWatch({ control, name: ['payments', 'change'] })
    const free = nextFreeAccount(accounts, usedAccountIds)
    const tooLarge = changeTooLarge({ payments: payments ?? [], change: change ?? [] }, accounts ?? [])

    return (
        <section className="space-y-3" aria-label={t('forms:transactions.changeTitle')}>
            <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                    <div className="text-sm font-medium">{t('forms:transactions.changeTitle')}</div>
                    {fields.length === 0 && (
                        <p className="text-xs text-muted-foreground">{t('forms:transactions.changeHelp')}</p>
                    )}
                </div>
                {fields.length === 0 && (
                    <span title={free ? t('forms:transactions.addChange') : t('forms:transactions.noAccountLeft')}>
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-8 shrink-0"
                            aria-label={t('forms:transactions.addChange')}
                            disabled={!free}
                            onClick={() => append({ account_id: free!.id, amount: EMPTY_AMOUNT })}
                        >
                            <Plus />
                        </Button>
                    </span>
                )}
            </div>

            {fields.map((field, index) => {
                const account = accounts?.find((a) => a.id === Number(change?.[index]?.account_id))
                const balance = balances.find((row) => row.account.id === account?.id)
                const last = index === fields.length - 1
                return (
                    <PaymentRow
                        key={field.id}
                        control={control}
                        list="change"
                        index={index}
                        currency={account?.currency}
                        excludeIds={usedAccountIds.filter((id) => id !== account?.id)}
                        action={
                            <PaymentRowButtons
                                onRemove={() => remove(index)}
                                removeLabel={t('forms:transactions.removeChange')}
                                onAdd={last ? () => append({ account_id: free!.id, amount: EMPTY_AMOUNT }) : undefined}
                                addLabel={t('forms:transactions.addChange')}
                                addDisabled={!free}
                                addDisabledReason={t('forms:transactions.noAccountLeft')}
                            />
                        }
                    >
                        {showBalances && balance && <BalanceAfter preview={balance} />}
                    </PaymentRow>
                )
            })}

            {tooLarge && (
                <p role="alert" className="text-xs font-medium text-destructive">
                    {t('common:validation.changeTooLarge')}
                </p>
            )}
        </section>
    )
}
