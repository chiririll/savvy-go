import { useRef } from 'react'
import type { FieldPath, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useNegativeBalanceConfirm } from '@/components/shared'
import type { NegativeBalanceWarning } from '@/lib/negative-balance'
import { payloadErrorPath } from '@/lib/transaction-payments'
import { buildTransactionInput, type PayloadError } from '@/lib/transaction-payload'
import { isTransferType } from '@/lib/transaction-type'
import {
    buildSpaceTransferInput,
    buildTransferInput,
    transferErrorPath,
    type TransferError,
} from '@/lib/transfer-payload'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, SpaceTransferInput, TransactionInput } from '@/types'

function payloadErrorMessage(t: TFunction, error: PayloadError): string {
    switch (error) {
        case 'unknownAccount':
            return t('validation.selectAccount')
        case 'duplicateAccount':
            return t('validation.accountUsedTwice')
        case 'amountZero':
            return t('validation.amountPositive')
        case 'netSign':
            return t('validation.changeTooLarge')
        case 'lineCurrency':
            return t('validation.lineCurrency')
    }
}

function transferErrorMessage(t: TFunction, error: TransferError): string {
    switch (error) {
        case 'unknownSpace':
            return t('validation.selectSpace')
        case 'unknownFrom':
            return t('validation.selectAccount')
        case 'unknownTo':
            return t('validation.transferDestination')
        case 'sameAccount':
            return t('validation.accountUsedTwice')
        case 'amountZero':
        case 'receivedZero':
        case 'feeInvalid':
            return t('validation.amountPositive')
    }
}

interface TransactionHandlersOptions {
    form: UseFormReturn<TransactionFormValues>
    warnings: NegativeBalanceWarning[]
    dateRequired: boolean
    accounts?: Account[]
    onSubmit: (data: TransactionInput) => void
    /** A transfer to a linked space goes through the transfers API, not as a transaction. */
    onSubmitSpaceTransfer?: (data: SpaceTransferInput) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
}

/** A transfer moves money between two accounts: it has no lines or change, and one payment that sends. */
function toTransfer(form: UseFormReturn<TransactionFormValues>, from: TransactionFormValues['type'], to: 'transfer' | 'transfer_out') {
    form.setValue('lines', [])
    form.setValue('change', [])
    form.setValue('payments', form.getValues('payments').slice(0, 1))
    // The other space has its own account ids, and takes no fee.
    if (from !== to && (from === 'transfer_out' || to === 'transfer_out')) {
        form.setValue('to_account_id', null)
        form.setValue('to_space_id', null)
        form.setValue('to_amount', null)
        form.setValue('fee', null)
    }
}

export function useTransactionHandlers({
    form,
    warnings,
    dateRequired,
    accounts,
    onSubmit,
    onSubmitSpaceTransfer,
    onTypeChange,
}: TransactionHandlersOptions) {
    const { t } = useTranslation('common')
    const { confirmIfNeeded, dialog } = useNegativeBalanceConfirm<TransactionInput | SpaceTransferInput>()

    const handleSubmit = form.handleSubmit((data) => {
        // The transfers API stores a transfer to another space under a date.
        if ((dateRequired || data.type === 'transfer_out') && !data.date) {
            form.setError('date', { message: t('validation.dateRequired') })
            return
        }
        if (isTransferType(data.type)) {
            const values = { ...data, type: data.type as 'transfer' | 'transfer_out' }
            const built =
                data.type === 'transfer_out'
                    ? buildSpaceTransferInput(values, accounts ?? [])
                    : buildTransferInput(values, accounts ?? [])
            if (!built.ok) {
                form.setError(transferErrorPath(built.error) as FieldPath<TransactionFormValues>, {
                    message: transferErrorMessage(t, built.error),
                })
                return
            }
            const send = (data.type === 'transfer_out' ? onSubmitSpaceTransfer : onSubmit) as (input: unknown) => void
            confirmIfNeeded(built.input, warnings, send ?? (() => {}))
            return
        }
        if (data.type !== 'income' && data.type !== 'expense') {
            return
        }
        const built = buildTransactionInput({ ...data, type: data.type }, accounts ?? [])
        if (!built.ok) {
            form.setError(payloadErrorPath(built.error, { ...data, type: data.type }, accounts ?? []) as FieldPath<TransactionFormValues>, {
                message: payloadErrorMessage(t, built.error),
            })
            return
        }
        confirmIfNeeded(built.input, warnings, onSubmit as (input: TransactionInput | SpaceTransferInput) => void)
    })

    // Categories are per income/expense type: keep each, so a slip onto another tab loses nothing.
    const categoryByType = useRef<Partial<Record<TransactionFormValues['type'], number>>>({})

    const handleTypeChange = (value: string) => {
        if (value !== 'income' && value !== 'expense' && value !== 'transfer' && value !== 'transfer_out') {
            return
        }
        const current = form.getValues('type')
        const categoryId = form.getValues('default_category_id')
        if (categoryId && (current === 'income' || current === 'expense')) {
            categoryByType.current[current] = categoryId
        }
        form.setValue('type', value)
        if (value === 'transfer' || value === 'transfer_out') {
            toTransfer(form, current, value)
            onTypeChange?.(value)
            return
        }
        // The categories of the lines are of the old type too.
        const lines = form.getValues('lines') ?? []
        if (lines.some((line) => line.category_id)) {
            form.setValue('lines', lines.map((line) => ({ ...line, category_id: null })))
        }
        // Null lets the model pick the default category of the type.
        form.setValue('default_category_id', categoryByType.current[value] ?? null)
        onTypeChange?.(value)
    }

    return { handleSubmit, handleTypeChange, negativeBalanceDialog: dialog }
}
