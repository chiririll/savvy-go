import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, TransactionGroup } from '@/types'
import { paymentLegs, type PaymentAccount, type PaymentPart, type SplitError, type SplitResult } from './split-payment'

export type PaymentError = SplitError | 'unknownAccount'

export type PaymentResult = Extract<SplitResult, { ok: true }> | { ok: false; error: PaymentError }

function paymentAccount(account: Account): PaymentAccount {
    return { id: account.id, currency: account.currency?.code ?? '', decimals: account.currency?.decimals ?? 2 }
}

/** Whether the form describes a payment in several parts: more rows than the first one. */
export function isSplitPayment(values: { extra_payments?: unknown[] | null; changes?: unknown[] | null }): boolean {
    return Boolean(values.extra_payments?.length || values.changes?.length)
}

/**
 * The legs of the payment the form describes: the first row (amount and
 * account), the extra rows, and the change rows; the accounts give each amount
 * its currency.
 */
export function buildPaymentLegs(values: TransactionFormValues, accounts: Account[]): PaymentResult {
    const part = (accountId: number | null | undefined, amount: number | null | undefined): PaymentPart | null => {
        const account = accounts.find((a) => a.id === Number(accountId))
        return account ? { account: paymentAccount(account), amount: Number(amount) } : null
    }
    const parts = (rows: { account_id: number; amount: number }[]) =>
        rows.map((row) => part(row.account_id, row.amount))
    const known = (list: (PaymentPart | null)[]): list is PaymentPart[] => list.every(Boolean)

    const paid = [part(values.account_id, values.amount), ...parts(values.extra_payments ?? [])]
    const change = parts(values.changes ?? [])
    if (!known(paid) || !known(change)) {
        return { ok: false, error: 'unknownAccount' }
    }

    return paymentLegs(
        {
            date: values.date ?? '',
            description: values.description,
            categoryId: Number(values.category_id),
            tagIds: values.tag_ids,
        },
        paid,
        change,
    )
}

/**
 * The form values that describe a saved payment, or null when its legs have a
 * shape the form cannot show (one made through the API or MCP, such as an
 * exchange between accounts): expenses first, then the incomes of the change.
 */
export function paymentFormValues(group: TransactionGroup): Partial<TransactionFormValues> | null {
    const [first] = group.legs
    if (!first || group.legs.length < 2 || first.type !== 'expense') {
        return null
    }
    const paid = group.legs.filter((leg) => leg.type === 'expense')
    const change = group.legs.slice(paid.length)
    const ordered = group.legs.slice(0, paid.length).every((leg) => leg.type === 'expense')
    if (!ordered || !change.every((leg) => leg.type === 'income')) {
        return null
    }

    return {
        type: 'expense',
        account_id: first.account.id,
        category_id: first.category?.id ?? null,
        amount: first.amount,
        is_estimated: false,
        description: first.description ?? '',
        date: first.date ?? '',
        tag_ids: first.tags.map((tag) => tag.id),
        items: [],
        extra_payments: paid.slice(1).map((leg) => ({ account_id: leg.account.id, amount: leg.amount })),
        changes: change.map((leg) => ({ account_id: leg.account.id, amount: leg.amount })),
    }
}
