import type { Account, SpaceTransferInput, TransactionInput } from '@/types'
import { roundToCurrency } from './transaction-payload'

/** The fee of a transfer: money the source pays on top of the amount, spent on an expense category. */
export interface TransferFeeValues {
    amount: number
    category_id?: number | null
}

/** What the form holds that a transfer is made of. The first payment is the source and what it sends. */
export interface TransferValues {
    type: 'transfer' | 'transfer_out'
    description?: string | null
    date?: string | null
    tag_ids?: number[]
    payments: Array<{ account_id: number; amount: number }>
    to_account_id?: number | null
    /** What the destination receives; left out, it is what was sent. */
    to_amount?: number | null
    /** The linked space of a transfer_out; its account ids are not this space's. */
    to_space_id?: number | null
    fee?: TransferFeeValues | null
}

export type TransferError = 'unknownSpace' | 'unknownFrom' | 'unknownTo' | 'sameAccount' | 'amountZero' | 'receivedZero' | 'feeInvalid'

export type TransferResult = { ok: true; input: TransactionInput } | { ok: false; error: TransferError }
export type SpaceTransferResult = { ok: true; input: SpaceTransferInput } | { ok: false; error: TransferError }

const accountById = (accounts: Account[], id: unknown): Account | undefined => accounts.find((a) => a.id === Number(id))

const positive = (value: unknown): number => (Number(value) > 0 ? Number(value) : 0)

/** What the destination receives, rounded to its currency: what was sent when both accounts share one. */
export function transferReceived(values: TransferValues, accounts: Account[]): number {
    const from = accountById(accounts, values.payments[0]?.account_id)
    const to = accountById(accounts, values.to_account_id)
    const sent = positive(values.payments[0]?.amount)
    if (from?.currency && to?.currency && from.currency.id === to.currency.id) {
        return roundToCurrency(sent, from.currency)
    }
    const received = positive(values.to_amount)
    return to?.currency ? roundToCurrency(received, to.currency) : received
}

/** The field of the form a refused transfer belongs to. */
export function transferErrorPath(error: TransferError): string {
    switch (error) {
        case 'unknownSpace':
            return 'to_space_id'
        case 'unknownFrom':
            return 'payments.0.account_id'
        case 'unknownTo':
        case 'sameAccount':
            return 'to_account_id'
        case 'amountZero':
            return 'payments.0.amount'
        case 'receivedZero':
            return 'to_amount'
        case 'feeInvalid':
            return 'fee.amount'
    }
}

/** A fee that is empty or zero is no fee; a negative or broken one is a mistake. */
function feeOf(values: TransferValues): { amount: number } | TransferError | null {
    if (!values.fee) {
        return null
    }
    const amount = Number(values.fee.amount)
    if (Number.isNaN(amount) || amount < 0) {
        return 'feeInvalid'
    }
    return amount > 0 ? { amount } : null
}

export interface TransferBuildOptions {
    /**
     * The transfer is the template of a recurring rule: between two currencies the amount received may stay open
     * (`null`), and each occurrence takes what the rates of its day make of the amount sent.
     */
    template?: boolean
}

/**
 * The payload for a transfer between two accounts of this space: a payment that takes money from the source and
 * one that brings it to the destination. The fee is paid by the source on top of the amount, and saved as a line,
 * so reports count it as spending.
 */
export function buildTransferInput(
    values: TransferValues,
    accounts: Account[],
    options?: TransferBuildOptions,
): TransferResult {
    const from = accountById(accounts, values.payments[0]?.account_id)
    const to = accountById(accounts, values.to_account_id)
    if (!from?.currency) {
        return { ok: false, error: 'unknownFrom' }
    }
    if (!to?.currency) {
        return { ok: false, error: 'unknownTo' }
    }
    if (from.id === to.id) {
        return { ok: false, error: 'sameAccount' }
    }
    const sent = roundToCurrency(positive(values.payments[0].amount), from.currency)
    if (!(sent > 0)) {
        return { ok: false, error: 'amountZero' }
    }
    const received = transferReceived(values, accounts)
    const open = Boolean(options?.template) && !(received > 0)
    if (!(received > 0) && !open) {
        return { ok: false, error: 'receivedZero' }
    }
    const fee = feeOf(values)
    if (typeof fee === 'string') {
        return { ok: false, error: fee }
    }
    const feeAmount = fee ? roundToCurrency(fee.amount, from.currency) : 0

    return {
        ok: true,
        input: {
            type: 'transfer',
            description: values.description?.trim() ? values.description.trim() : null,
            date: values.date ? values.date : null,
            payments: [
                { account_id: from.id, amount: -roundToCurrency(sent + feeAmount, from.currency) },
                { account_id: to.id, amount: open ? null : received },
            ],
            lines:
                feeAmount > 0
                    ? [{ category_id: values.fee?.category_id ?? null, currency_id: from.currency.id, amount: feeAmount }]
                    : [],
            tag_ids: values.tag_ids ?? [],
        },
    }
}

/**
 * The request of the transfers API for a transfer to a linked space. The destination account is in the other
 * space, so it is not looked up here; the API checks it. A transfer to another space has no fee and no tags.
 */
export function buildSpaceTransferInput(values: TransferValues, accounts: Account[]): SpaceTransferResult {
    if (!(Number(values.to_space_id) > 0)) {
        return { ok: false, error: 'unknownSpace' }
    }
    const from = accountById(accounts, values.payments[0]?.account_id)
    if (!from?.currency) {
        return { ok: false, error: 'unknownFrom' }
    }
    if (!(Number(values.to_account_id) > 0)) {
        return { ok: false, error: 'unknownTo' }
    }
    const sent = roundToCurrency(positive(values.payments[0].amount), from.currency)
    if (!(sent > 0)) {
        return { ok: false, error: 'amountZero' }
    }
    const received = positive(values.to_amount) || sent

    return {
        ok: true,
        input: {
            to_space_id: Number(values.to_space_id),
            from_account_id: from.id,
            to_account_id: Number(values.to_account_id),
            from_amount: String(sent),
            to_amount: String(received),
            date: values.date ?? '',
            description: values.description?.trim() ? values.description.trim() : null,
        },
    }
}

export interface TransferSwap {
    account_id: number
    to_account_id: number
    amount: number | null
    to_amount: number | null
    exchange_rate: number | null
}

const RATE_DECIMALS = 6

/**
 * The values after the source and the destination change places, each with its amount, or null until both
 * accounts are chosen. The rate is inverted, so the amounts and the rate stay in agreement.
 */
export function swapTransfer(values: {
    payments: Array<{ account_id: number; amount: number | null }>
    to_account_id?: number | null
    to_amount?: number | null
    exchange_rate?: number | null
}): TransferSwap | null {
    const from = Number(values.payments[0]?.account_id)
    const to = Number(values.to_account_id)
    if (!(from > 0) || !(to > 0)) {
        return null
    }
    const sent = positive(values.payments[0].amount)
    const received = positive(values.to_amount)
    const rate = Number(values.exchange_rate)
    return {
        account_id: to,
        to_account_id: from,
        amount: received > 0 ? received : null,
        to_amount: sent > 0 ? sent : null,
        exchange_rate: rate > 0 ? Math.round((1 / rate) * 10 ** RATE_DECIMALS) / 10 ** RATE_DECIMALS : null,
    }
}
