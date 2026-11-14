import { useFieldArray, type UseFormReturn } from 'react-hook-form'
import { TransactionTypeTabs } from '@/components/shared'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { LinkedSpace } from '@/types/spaces'
import { PendingHint } from './TransactionBalanceHint'
import { ChangeSection } from './ChangeSection'
import { PaymentsSection } from './PaymentsSection'
import { TemplateModeContext } from './template-mode'
import { TransactionDescriptionField } from './TransactionDescriptionField'
import { TransactionDetailsRow } from './TransactionDetailsRow'
import { TransactionLines } from './TransactionLines'
import { TransactionTagsField } from './TransactionTagsField'
import { TransferAccounts } from './TransferAccounts'
import { TransferFeeSection } from './TransferFeeSection'
import { TransferRateField } from './TransferRateField'
import type { TransactionModel } from './use-transaction-model'

interface TransactionFormBodyProps {
    form: UseFormReturn<TransactionFormValues>
    /** What the form works out from what it holds: see `useTransactionModel`. */
    model: TransactionModel
    onTypeChange: (value: string) => void
    /** The linked spaces a transfer can go to; a template cannot move money between spaces. */
    transferTargets: LinkedSpace[]
    withSpaceTransfer: boolean
    originalAffectsBalance?: boolean
    /** The form draws the template of a recurring rule: no date, no balances, a transfer may leave its amount open. */
    template?: boolean
    /** The type of a saved transaction cannot change: the selector is not drawn. */
    hideTypeSelector?: boolean
}

/**
 * The fields of a transaction, from its type to the tags and the fee: the body of the transaction form, and of the
 * template of a recurring rule, which draws the same fields. The caller owns the `<form>`, the submit and the
 * confirmations.
 */
export function TransactionFormBody({
    form,
    model,
    onTypeChange,
    transferTargets,
    withSpaceTransfer,
    originalAffectsBalance,
    template = false,
    hideTypeSelector = false,
}: TransactionFormBodyProps) {
    const {
        accounts,
        coverage,
        lineCurrencies,
        hasLines,
        uncovered,
        nextLine,
        transactionType,
        isTransfer,
        isSpaceTransfer,
        destinationSpaceId,
        destinationSpaceAccounts,
        selectedAccount,
        selectedToAccount,
        rate,
        dateRequired,
        isPendingDate,
        isUndated,
        balances,
        usedAccountIds,
    } = model

    const lines = useFieldArray({ control: form.control, name: 'lines' })
    const addLine = () => lines.append(nextLine())

    // Change is for what is paid; an income shows it only when a saved one has some.
    const showChange = !isTransfer && (transactionType === 'expense' || (form.watch('change')?.length ?? 0) > 0)
    // A template moves no balance: there is nothing to preview, and no date to be pending by.
    const showBalances = !template && !isPendingDate
    // The balance now stands under every account, so nothing moves when an amount is typed; a pending date only
    // keeps it from moving.
    const balanceMode = template ? 'none' : showBalances ? 'after' : 'current'

    return (
        <TemplateModeContext.Provider value={template}>
            {!hideTypeSelector && (
                <TransactionTypeTabs value={transactionType} withSpaceTransfer={withSpaceTransfer} onChange={onTypeChange} />
            )}

            <TransactionDescriptionField control={form.control} />

            {isTransfer ? (
                <TransferAccounts
                    isSpaceTransfer={isSpaceTransfer}
                    links={transferTargets}
                    destinationAccounts={destinationSpaceAccounts}
                    destinationSpaceId={destinationSpaceId}
                    from={selectedAccount}
                    to={selectedToAccount}
                    rate={rate}
                    balances={balances.rows}
                    balanceMode={balanceMode}
                />
            ) : (
                <PaymentsSection
                    accounts={accounts}
                    usedAccountIds={usedAccountIds}
                    balances={balances.rows}
                    balanceMode={balanceMode}
                />
            )}
            {!template && isPendingDate && <PendingHint isUndated={isUndated} />}

            <TransactionDetailsRow
                control={form.control}
                type={transactionType}
                dateRequired={dateRequired}
                withDate={!template}
                pastOnly={originalAffectsBalance}
                category={{ hasLines, locked: coverage.remainder === 0, onAddLine: addLine }}
                aside={
                    isTransfer ? (
                        <TransferRateField rate={rate} fromCode={selectedAccount?.currency?.code} toCode={selectedToAccount?.currency?.code} />
                    ) : undefined
                }
            />

            {!isTransfer && (
                <TransactionLines
                    control={form.control}
                    fields={lines.fields}
                    remove={lines.remove}
                    type={transactionType === 'income' ? 'income' : 'expense'}
                    currencies={lineCurrencies}
                    coverage={coverage}
                    uncovered={uncovered}
                />
            )}

            {showChange && (
                <ChangeSection
                    accounts={accounts}
                    usedAccountIds={usedAccountIds}
                    balances={balances.rows}
                    balanceMode={balanceMode}
                />
            )}

            {isTransfer && !isSpaceTransfer && <TransferFeeSection currency={selectedAccount?.currency} />}

            {/* The transfers API keeps no tags. */}
            {!isSpaceTransfer && <TransactionTagsField control={form.control} />}

        </TemplateModeContext.Provider>
    )
}
