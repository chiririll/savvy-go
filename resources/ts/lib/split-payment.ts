import { roundMoney } from './transaction-items'

/*
 * A payment made in several parts is saved as legs: ordinary transactions that
 * share a group (see the Go ledger.Groups). A payment with change is an
 * expense of the price plus a transfer that exchanges the money handed over
 * into the change, so the expense is the price whatever today's rate is.
 */

export interface PaymentAccount {
    id: number
    currency: string
    decimals: number
}

export interface PaymentCommon {
    date: string
    description?: string | null
    categoryId: number
    tagIds?: number[]
}

/** One leg as the API takes it. */
export interface PaymentLeg {
    type: 'expense' | 'transfer'
    account_id: number
    to_account_id?: number
    category_id?: number
    amount: number
    to_amount?: number
    description?: string
    date: string
    tag_ids?: number[]
}

export type SplitError = 'needTwoParts' | 'amountPositive' | 'sameCurrency' | 'handedOverTooLow'

export type SplitResult = { ok: true; legs: PaymentLeg[]; rate?: number } | { ok: false; error: SplitError }

const RATE_DECIMALS = 6

const positive = (...values: number[]) => values.every((value) => Number.isFinite(value) && value > 0)

function shared({ date, description, tagIds }: PaymentCommon) {
    return {
        date,
        ...(description ? { description } : {}),
        ...(tagIds?.length ? { tag_ids: tagIds } : {}),
    }
}

/** A payment settled with several accounts: one expense on each. */
export function splitAcrossAccounts(
    common: PaymentCommon,
    parts: { account: PaymentAccount; amount: number }[],
): SplitResult {
    if (new Set(parts.map((part) => part.account.id)).size < 2) {
        return { ok: false, error: 'needTwoParts' }
    }
    if (!positive(...parts.map((part) => part.amount))) {
        return { ok: false, error: 'amountPositive' }
    }

    return {
        ok: true,
        legs: parts.map(({ account, amount }) => ({
            type: 'expense',
            account_id: account.id,
            category_id: common.categoryId,
            amount: roundMoney(amount, account.decimals),
            ...shared(common),
        })),
    }
}

export interface PayWithChange {
    /**
     * Which currency the price is in: the one handed over ("paid") or the one
     * the change came back in ("change").
     */
    priceOn: 'paid' | 'change'
    price: number
    paidFrom: PaymentAccount
    handedOver: number
    changeTo: PaymentAccount
    change: number
}

/**
 * Handing over money in one currency and getting change in another.
 *
 * Price in the currency handed over (500 EUR for 450 EUR of rent, 5900 RSD
 * back): an expense of 450 EUR and an exchange of the 50 EUR left into 5900 RSD.
 *
 * Price in the currency of the change (450 EUR for 53000 RSD, 200 RSD back):
 * the 450 EUR are exchanged into the 53200 RSD that cover price and change,
 * then the price is paid from the account that received them.
 *
 * `rate` is what the exchange came to: units of the change currency per unit
 * of the currency handed over.
 */
export function payWithChange(common: PaymentCommon, input: PayWithChange): SplitResult {
    const { priceOn, price, paidFrom, handedOver, changeTo, change } = input
    if (!positive(price, handedOver, change)) {
        return { ok: false, error: 'amountPositive' }
    }
    if (paidFrom.currency === changeTo.currency) {
        return { ok: false, error: 'sameCurrency' }
    }

    const expense = (account: PaymentAccount, amount: number): PaymentLeg => ({
        type: 'expense',
        account_id: account.id,
        category_id: common.categoryId,
        amount,
        ...shared(common),
    })
    const exchange = (amount: number, toAmount: number): PaymentLeg => ({
        type: 'transfer',
        account_id: paidFrom.id,
        to_account_id: changeTo.id,
        amount,
        to_amount: toAmount,
        ...shared(common),
    })

    if (priceOn === 'paid') {
        const surplus = roundMoney(handedOver - price, paidFrom.decimals)
        if (surplus <= 0) {
            return { ok: false, error: 'handedOverTooLow' }
        }
        const toAmount = roundMoney(change, changeTo.decimals)
        return {
            ok: true,
            rate: roundMoney(toAmount / surplus, RATE_DECIMALS),
            legs: [expense(paidFrom, roundMoney(price, paidFrom.decimals)), exchange(surplus, toAmount)],
        }
    }

    const received = roundMoney(price + change, changeTo.decimals)
    const given = roundMoney(handedOver, paidFrom.decimals)
    return {
        ok: true,
        rate: roundMoney(received / given, RATE_DECIMALS),
        legs: [expense(changeTo, roundMoney(price, changeTo.decimals)), exchange(given, received)],
    }
}

/**
 * How far a rate is from the one expected, as a fraction of the expected one
 * (0.2 is 20%). Null when there is no rate to compare with.
 */
export function rateDeviation(rate: number, expected: number | null | undefined): number | null {
    if (!expected || !(expected > 0) || !Number.isFinite(rate)) {
        return null
    }
    return Math.abs(rate / expected - 1)
}
