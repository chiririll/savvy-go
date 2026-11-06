import type { ReactNode } from 'react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { AccountSelect } from '@/components/shared/pickers/AccountSelect'
import { useAccounts } from '@/hooks'
import { formatCurrency } from '@/lib/utils'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import { ACCOUNT_FILTER } from './use-transaction-model'

type Form = UseFormReturn<TransactionFormValues>

/** The lists of rows below the first one: the accounts that paid too, and where the change went. */
export type PaymentRowsName = 'extra_payments' | 'changes'

/** Width of the column of a row's buttons: one icon button, or two with the gap. */
export const ONE_BUTTON = '2.25rem'
export const TWO_BUTTONS = '4.75rem'

const NEW_ROW = { account_id: 0, amount: 0 }

/** The rows of a list, and how to add or remove one. */
export function usePaymentRows(form: Form, name: PaymentRowsName) {
    const rows = useWatch({ control: form.control, name }) ?? []
    const set = (next: typeof rows) => {
        form.clearErrors(name)
        form.setValue(name, next, { shouldDirty: true })
    }

    return {
        rows,
        add: () => set([...(form.getValues(name) ?? []), { ...NEW_ROW }]),
        remove: (index: number) => set((form.getValues(name) ?? []).filter((_, i) => i !== index)),
    }
}

function RowButton({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
    return (
        <Button type="button" variant="outline" size="icon" aria-label={label} title={label} onClick={onClick}>
            {children}
        </Button>
    )
}

/** The props of the first row (amount and account) that make room for its "+" while rows can be added. */
export function useFirstRow(form: Form, enabled: boolean) {
    const { rows } = usePaymentRows(form, 'extra_payments')
    if (!enabled) {
        return {}
    }
    return { actionWidth: rows.length > 0 ? TWO_BUTTONS : ONE_BUTTON, rowAction: <FirstRowAction form={form} /> }
}

/** The "+" on the first row, shown while it is the last one. */
function FirstRowAction({ form }: { form: Form }) {
    const { t } = useTranslation('forms')
    const { rows, add } = usePaymentRows(form, 'extra_payments')

    if (rows.length > 0) {
        return null
    }
    return (
        <RowButton label={t('transactions.payment.addAccount')} onClick={add}>
            <Plus className="size-4" />
        </RowButton>
    )
}

interface PaymentRowProps {
    form: Form
    name: PaymentRowsName
    index: number
    actionWidth: string
    children: ReactNode
}

/** An amount and the account it is in, with the buttons of the row. */
function PaymentRow({ form, name, index, actionWidth, children }: PaymentRowProps) {
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const accountId = useWatch({ control: form.control, name: `${name}.${index}.account_id` })
    const symbol = accounts?.find((a) => a.id === Number(accountId))?.currency?.symbol

    return (
        <div className="grid items-start gap-4" style={{ gridTemplateColumns: `1fr 1fr ${actionWidth}` }}>
            <FormField
                control={form.control}
                name={`${name}.${index}.amount`}
                render={({ field }) => (
                    <FormItem className="min-w-0">
                        <FormControl>
                            <div className="relative">
                                <Input
                                    type="number"
                                    step="0.01"
                                    min={0}
                                    placeholder="0.00"
                                    className={symbol ? 'pr-10' : undefined}
                                    value={field.value || ''}
                                    onChange={(event) => field.onChange(event.target.value ? Number(event.target.value) : 0)}
                                />
                                {symbol && (
                                    <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground">
                                        {symbol}
                                    </span>
                                )}
                            </div>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />
            <FormField
                control={form.control}
                name={`${name}.${index}.account_id`}
                render={({ field }) => (
                    <FormItem className="min-w-0">
                        <AccountSelect value={field.value} onChange={field.onChange} />
                        <FormMessage />
                    </FormItem>
                )}
            />
            <div className="flex justify-end gap-1">{children}</div>
        </div>
    )
}

/** The accounts that paid besides the first row; the "+" is on the last of them. */
export function ExtraPaymentRows({ form }: { form: Form }) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { rows, add, remove } = usePaymentRows(form, 'extra_payments')
    const [accountId, amount] = useWatch({ control: form.control, name: ['account_id', 'amount'] })

    if (rows.length === 0) {
        return null
    }

    // One total per currency: amounts of different currencies are not added.
    const totals = new Map<string, { amount: number; account: Account }>()
    for (const part of [{ account_id: accountId, amount }, ...rows]) {
        const account = accounts?.find((a) => a.id === Number(part.account_id))
        if (account?.currency) {
            const key = account.currency.code
            totals.set(key, { account, amount: (totals.get(key)?.amount ?? 0) + (Number(part.amount) || 0) })
        }
    }

    return (
        <div className="min-w-0 space-y-2">
            {rows.map((_, index) => (
                <PaymentRow key={index} form={form} name="extra_payments" index={index} actionWidth={TWO_BUTTONS}>
                    <RowButton label={t('forms:transactions.payment.removeRow')} onClick={() => remove(index)}>
                        <X className="size-4" />
                    </RowButton>
                    {index === rows.length - 1 && (
                        <RowButton label={t('forms:transactions.payment.addAccount')} onClick={add}>
                            <Plus className="size-4" />
                        </RowButton>
                    )}
                </PaymentRow>
            ))}
            <FormField control={form.control} name="extra_payments" render={() => <FormMessage />} />
            <p className="text-sm font-medium">
                {t('forms:transactions.total')}:{' '}
                <span className="font-mono">
                    {[...totals.values()].map((total) => formatCurrency(total.amount, total.account.currency)).join(' + ')}
                </span>
            </p>
        </div>
    )
}

/** Where the change went, one row for each account, added with "Add change". */
export function ChangeSection({ form }: { form: Form }) {
    const { t } = useTranslation('forms')
    const { rows, add, remove } = usePaymentRows(form, 'changes')

    return (
        <div className="min-w-0 space-y-3">
            <div className="flex min-w-0 items-center justify-between gap-2">
                <div className="text-sm font-medium leading-none">
                    {t('transactions.payment.changeCount', { count: rows.length })}
                </div>
                <Button type="button" variant="outline" size="sm" onClick={add}>
                    <Plus className="mr-1 size-4" />
                    {t('transactions.payment.addChange')}
                </Button>
            </div>
            {rows.length === 0 ? (
                <p className="text-xs text-muted-foreground">{t('transactions.payment.changeHelp')}</p>
            ) : (
                <div className="min-w-0 space-y-2">
                    {rows.map((_, index) => (
                        <PaymentRow key={index} form={form} name="changes" index={index} actionWidth={ONE_BUTTON}>
                            <RowButton label={t('transactions.payment.removeRow')} onClick={() => remove(index)}>
                                <X className="size-4" />
                            </RowButton>
                        </PaymentRow>
                    ))}
                    <FormField control={form.control} name="changes" render={() => <FormMessage />} />
                </div>
            )}
        </div>
    )
}
