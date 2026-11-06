import { roundMoney } from './transaction-items'

/*
 * A payment made in several parts is saved as legs: ordinary transactions that
 * share a group (see the Go ledger.Groups). Each account that paid is an
 * expense for what it paid. Change given back is an income on the account it
 * went to, so a purchase paid in euros with change in dinars is an expense in
 * euros and an income in dinars; reports convert both to the base currency.
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
    type: 'expense' | 'income'
    account_id: number
    category_id?: number
    amount: number
    description?: string
    date: string
    tag_ids?: number[]
}

export interface PaymentPart {
    account: PaymentAccount
    amount: number
}

export type SplitError = 'needTwoParts' | 'amountPositive'

export type SplitResult = { ok: true; legs: PaymentLeg[] } | { ok: false; error: SplitError }

const positive = (...values: number[]) => values.every((value) => Number.isFinite(value) && value > 0)

function leg(common: PaymentCommon, type: PaymentLeg['type'], { account, amount }: PaymentPart): PaymentLeg {
    return {
        type,
        account_id: account.id,
        category_id: common.categoryId,
        amount: roundMoney(amount, account.decimals),
        date: common.date,
        ...(common.description ? { description: common.description } : {}),
        ...(common.tagIds?.length ? { tag_ids: common.tagIds } : {}),
    }
}

/**
 * The legs of an expense paid from several accounts, with change given back
 * on others: an expense on each account that paid, then an income on each
 * account that got change. Without change at least two accounts must pay,
 * otherwise it is an ordinary expense.
 */
export function paymentLegs(common: PaymentCommon, paid: PaymentPart[], change: PaymentPart[] = []): SplitResult {
    if (change.length === 0 && new Set(paid.map((part) => part.account.id)).size < 2) {
        return { ok: false, error: 'needTwoParts' }
    }
    if (!positive(...paid.map((part) => part.amount), ...change.map((part) => part.amount))) {
        return { ok: false, error: 'amountPositive' }
    }

    return {
        ok: true,
        legs: [...paid.map((part) => leg(common, 'expense', part)), ...change.map((part) => leg(common, 'income', part))],
    }
}
