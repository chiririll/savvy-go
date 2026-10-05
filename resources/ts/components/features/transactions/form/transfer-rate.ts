import { useCallback, useEffect, useRef } from 'react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
import { isTransferType, type TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'

const RATE_DECIMALS = 6

export function roundTo(value: number, decimals: number): number {
    const factor = 10 ** decimals
    return Math.round(value * factor) / factor
}

/** The rate implied by both currencies' rates to the space's base currency. */
function catalogTransferRate(fromRate?: number | null, toRate?: number | null): number | null {
    const from = Number(fromRate)
    const to = Number(toRate)
    if (!(from > 0) || !(to > 0)) return null
    return roundTo(from / to, RATE_DECIMALS)
}

function derivedTransferRate(amount: number, toAmount: number): number | null {
    if (!(amount > 0) || !(toAmount > 0)) return null
    return roundTo(toAmount / amount, RATE_DECIMALS)
}

/** The rate a form opens with: the one its amounts imply, if any. */
export function resolveOpeningTransferRate(values?: Partial<TransactionFormValues>): number | null {
    if (values?.type && !isTransferType(values.type)) return null
    return derivedTransferRate(Number(values?.amount) || 0, Number(values?.to_amount) || 0)
}

interface TransferRateOptions {
    form: UseFormReturn<TransactionFormValues>
    from?: Account
    to?: Account
    /** The destination currency's rate in this space's catalog. */
    toRate?: number | null
    /** The form (re)opens: start over. */
    open: boolean
}

/**
 * Keeps a transfer's exchange_rate and to_amount consistent while the user
 * edits the amounts, the rate or the accounts.
 */
export function useTransferRate({ form, from, to, toRate, open }: TransferRateOptions) {
    const [type, amount, toAmount, accountId, toSpaceId, toAccountId] = useWatch({
        control: form.control,
        name: ['type', 'amount', 'to_amount', 'account_id', 'to_space_id', 'to_account_id'],
    })
    const isTransfer = isTransferType(type)
    const sameCurrency = Boolean(from && to && from.currency?.code === to.currency?.code)
    const editable = Boolean(isTransfer && from && to && !sameCurrency)
    const fromRate = from?.currency?.rate
    const destDecimals = to?.currency?.decimals ?? 2
    const pairKey = `${accountId ?? ''}:${toSpaceId ?? ''}:${toAccountId ?? ''}`

    const lastPairRef = useRef<string | null>(null)
    const editingRef = useRef(false)
    const skipRateFromAmountRef = useRef(false)
    const skipPairRecalcRef = useRef(false)

    useEffect(() => {
        if (!open) return
        lastPairRef.current = null
        editingRef.current = false
        skipRateFromAmountRef.current = false
        skipPairRecalcRef.current = false
    }, [open])

    /** The user is typing a rate: amounts don't override it until commit. */
    const startEditing = useCallback(() => {
        editingRef.current = true
    }, [])

    /** The accounts were swapped together with their amounts: keep those. */
    const keepNextPair = useCallback(() => {
        skipPairRecalcRef.current = true
    }, [])

    const applyReceiveFromRate = useCallback((rate: number) => {
        const sourceAmount = Number(form.getValues('amount')) || 0
        if (!(sourceAmount > 0) || !(rate > 0)) return
        skipRateFromAmountRef.current = true
        form.setValue('to_amount', roundTo(sourceAmount * rate, destDecimals), { shouldValidate: false })
    }, [form, destDecimals])

    /** Applies the typed rate to the received amount. */
    const commit = useCallback(() => {
        editingRef.current = false
        if (!editable) return

        let rate = Number(form.getValues('exchange_rate'))
        if (!(rate > 0)) {
            const sourceAmount = Number(form.getValues('amount')) || 0
            const destAmount = Number(form.getValues('to_amount')) || 0
            rate = derivedTransferRate(sourceAmount, destAmount)
                ?? catalogTransferRate(fromRate, toRate)
                ?? 1
            form.setValue('exchange_rate', rate, { shouldValidate: false })
        }

        applyReceiveFromRate(rate)
    }, [editable, form, fromRate, toRate, applyReceiveFromRate])

    useEffect(() => {
        if (!isTransfer) {
            lastPairRef.current = null
            return
        }

        if (!from || !to) {
            return
        }

        const prevPair = lastPairRef.current
        lastPairRef.current = pairKey
        const pairChanged = prevPair !== null && prevPair !== pairKey
        const isInit = prevPair === null

        if (skipPairRecalcRef.current) {
            skipPairRecalcRef.current = false
            return
        }

        if (sameCurrency) {
            form.setValue('exchange_rate', 1, { shouldValidate: false })
            form.setValue('to_amount', amount ? Number(amount) : null, { shouldValidate: false })
            return
        }

        if (pairChanged) {
            const rate = catalogTransferRate(fromRate, toRate) ?? 1
            form.setValue('exchange_rate', rate, { shouldValidate: false })
            applyReceiveFromRate(rate)
            return
        }

        const sourceAmount = Number(amount) || 0
        const destAmount = Number(toAmount) || 0

        if (isInit) {
            const rate = derivedTransferRate(sourceAmount, destAmount)
                ?? catalogTransferRate(fromRate, toRate)
                ?? 1
            form.setValue('exchange_rate', rate, { shouldValidate: false })
            if (sourceAmount > 0 && !(destAmount > 0)) {
                applyReceiveFromRate(rate)
            }
            return
        }

        if (editingRef.current || skipRateFromAmountRef.current) {
            skipRateFromAmountRef.current = false
            return
        }

        if (sourceAmount > 0 && destAmount > 0) {
            const nextRate = derivedTransferRate(sourceAmount, destAmount)
            const currentRate = Number(form.getValues('exchange_rate'))
            if (nextRate && Math.abs((currentRate || 0) - nextRate) >= 1 / 10 ** RATE_DECIMALS / 2) {
                form.setValue('exchange_rate', nextRate, { shouldValidate: false })
            }
            return
        }

        if (sourceAmount > 0 && !(destAmount > 0)) {
            const currentRate = Number(form.getValues('exchange_rate'))
            const rate = currentRate > 0
                ? currentRate
                : catalogTransferRate(fromRate, toRate) ?? 1
            if (!(currentRate > 0)) {
                form.setValue('exchange_rate', rate, { shouldValidate: false })
            }
            applyReceiveFromRate(rate)
        }
    }, [isTransfer, sameCurrency, amount, toAmount, pairKey, from, to, fromRate, toRate, form, applyReceiveFromRate])

    return { sameCurrency, editable, commit, startEditing, keepNextPair }
}
