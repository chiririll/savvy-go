import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { FormDialog, FormDialogCancel } from '@/components/shared/dialogs/FormDialog'
import { Button } from '@/components/ui/button'
import { AddAccountPopover } from './AddAccountPopover'
import { ReconcileRowItem } from './ReconcileRowItem'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useBalancesAt, useCreateReconciliations } from '@/hooks'
import { numberToAmount } from '@/lib/decimal-amount'
import { formatDateLocal } from '@/lib/dates'
import { formatCurrency } from '@/lib/currency'
import { testId } from '@/lib/test-id'
import {
    addRow,
    applyComputed,
    canSave,
    decimalsOf,
    editRow,
    initialRows,
    reconcilableAccounts,
    removeRow,
    removedAccounts,
    rowDifference,
    toPayload,
} from '@/lib/reconciliation-rows'
import type { Account } from '@/types'

const FORM_ID = 'reconcile-form'

interface ReconcileDialogProps {
    accounts: readonly Account[]
    open: boolean
    onOpenChange: (open: boolean) => void
}

/** Pins the balance of accounts to the real one on a date. The form is rebuilt each time the dialog opens. */
export function ReconcileDialog({ accounts, open, onOpenChange }: ReconcileDialogProps) {
    const { t } = useTranslation(['pages', 'common'])
    const today = formatDateLocal()
    const [date, setDate] = useState(today)
    const [rows, setRows] = useState(() => initialRows(accounts))
    const create = useCreateReconciliations()

    // Every opening starts over: today, all accounts, their current balances.
    useEffect(() => {
        if (!open) return
        setDate(formatDateLocal())
        setRows(initialRows(accounts))
        // Accounts refetched while the dialog is open must not discard what was typed.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open])

    const reconcilable = useMemo(() => reconcilableAccounts(accounts), [accounts])
    const byId = useMemo(() => new Map(reconcilable.map((account) => [account.id, account])), [reconcilable])
    const decimals = useMemo(
        () => new Map(reconcilable.map((account) => [account.id, decimalsOf(account)])),
        [reconcilable],
    )
    const isToday = date === today
    const history = useBalancesAt(date, decimals, open && !isToday && date !== '' && date < today)
    const current = useMemo(
        () =>
            new Map(
                reconcilable.map((account) => [account.id, numberToAmount(account.currentBalance, decimalsOf(account))]),
            ),
        [reconcilable],
    )
    const balances = isToday ? current : history.data
    // A date whose balances have not arrived: rows would show the previous date's difference.
    const loading = balances === undefined

    useEffect(() => {
        if (balances) setRows((previous) => applyComputed(previous, balances))
    }, [balances])

    const removed = removedAccounts(accounts, rows)
    const valid = !loading && canSave(date, today, rows, decimals)

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (!valid || create.isPending) return
        create.mutate(toPayload(date, rows), { onSuccess: () => onOpenChange(false) })
    }

    return (
        <FormDialog
            open={open}
            onOpenChange={onOpenChange}
            title={t('pages:accounts.reconcile.title')}
            description={t('pages:accounts.reconcile.description')}
            formId={FORM_ID}
            isSubmitting={create.isPending}
            className="sm:max-w-xl"
            footerEnd={
                <div className="ml-auto flex shrink-0 items-center gap-2">
                    <FormDialogCancel />
                    <Button type="submit" form={FORM_ID} disabled={!valid || create.isPending} {...testId('form-submit')}>
                        {create.isPending ? t('common:actions.saving') : t('common:actions.save')}
                    </Button>
                </div>
            }
        >
        <form id={FORM_ID} onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
                <Label htmlFor="reconcile-date">{t('common:fields.date')}</Label>
                <Input
                    id="reconcile-date"
                    type="date"
                    value={date}
                    max={today}
                    required
                    onChange={(event) => setDate(event.target.value)}
                    {...testId('reconcile-date')}
                />
            </div>

            {rows.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t('pages:accounts.reconcile.empty')}</p>
            ) : (
                <ul className="space-y-3">
                    {rows.map((row) => {
                        const account = byId.get(row.accountId)
                        if (!account) return null
                        const difference = loading ? null : rowDifference(row, decimalsOf(account))
                        return (
                            <ReconcileRowItem
                                key={row.accountId}
                                row={row}
                                account={account}
                                difference={difference}
                                disabled={loading}
                                onChange={(value) => setRows((previous) => editRow(previous, row.accountId, value))}
                                onRemove={() => setRows((previous) => removeRow(previous, row.accountId))}
                            />
                        )
                    })}
                </ul>
            )}

            {removed.length > 0 && (
                <div className="flex justify-end">
                    <AddAccountPopover
                        accounts={removed}
                        onPick={(id) =>
                            setRows((previous) => addRow(previous, accounts, id, balances?.get(id) ?? null))
                        }
                    />
                </div>
            )}
        </form>
        </FormDialog>
    )
}
