import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, TransactionGroup } from '@/types'
import {
    payWithChange,
    splitAcrossAccounts,
    type PaymentAccount,
    type PaymentCommon,
    type SplitError,
    type SplitResult,
} from './split-payment'
import { roundMoney } from './transaction-items'

export type PaymentError = SplitError | 'unknownAccount'

export type PaymentResult = Extract<SplitResult, { ok: true }> | { ok: false; error: PaymentError }

function paymentAccount(account: Account): PaymentAccount {
    return { id: account.id, currency: account.currency?.code ?? '', decimals: account.currency?.decimals ?? 2 }
}

/**
 * The legs of the payment the form describes, in the "several accounts" or
 * "with change" mode; the accounts give each amount its currency.
 */
export function buildPaymentLegs(values: TransactionFormValues, accounts: Account[]): PaymentResult {
    const find = (id: number | null | undefined) => {
        const account = accounts.find((a) => a.id === Number(id))
        return account ? paymentAccount(account) : null
    }
    const common: PaymentCommon = {
        date: values.date ?? '',
        description: values.description,
        categoryId: Number(values.category_id),
        tagIds: values.tag_ids,
    }
    const main = find(values.account_id)
    if (!main) {
        return { ok: false, error: 'unknownAccount' }
    }

    if (values.payment_mode === 'change') {
        const changeTo = find(values.change_account_id)
        if (!changeTo) {
            return { ok: false, error: 'unknownAccount' }
        }
        return payWithChange(common, {
            priceOn: 'paid',
            price: Number(values.amount),
            paidFrom: main,
            handedOver: Number(values.handed_over),
            changeTo,
            change: Number(values.change_amount),
        })
    }

    const parts = [{ account: main, amount: Number(values.amount) }]
    for (const extra of values.extra_payments ?? []) {
        const account = find(extra.account_id)
        if (!account) {
            return { ok: false, error: 'unknownAccount' }
        }
        parts.push({ account, amount: Number(extra.amount) })
    }
    return splitAcrossAccounts(common, parts)
}

/**
 * The form values that describe a saved payment, or null when its legs have a
 * shape the form cannot show (one made through the API or MCP, such as a
 * refund leg). The two shapes the form makes are read back.
 */
export function paymentFormValues(group: TransactionGroup): Partial<TransactionFormValues> | null {
    const [first, second] = group.legs
    if (!first || first.type !== 'expense') {
        return null
    }
    const common: Partial<TransactionFormValues> = {
        type: 'expense',
        account_id: first.account.id,
        category_id: first.category?.id ?? null,
        amount: first.amount,
        is_estimated: false,
        description: first.description ?? '',
        date: first.date ?? '',
        tag_ids: first.tags.map((tag) => tag.id),
        items: [],
    }

    if (group.legs.length >= 2 && group.legs.every((leg) => leg.type === 'expense')) {
        return {
            ...common,
            payment_mode: 'several',
            extra_payments: group.legs.slice(1).map((leg) => ({ account_id: leg.account.id, amount: leg.amount })),
        }
    }

    if (group.legs.length === 2 && second.type === 'transfer' && second.account.id === first.account.id) {
        if (!second.toAccount || second.toAmount == null) {
            return null
        }
        return {
            ...common,
            payment_mode: 'change',
            handed_over: roundMoney(first.amount + second.amount, first.account.currency?.decimals ?? 2),
            change_account_id: second.toAccount.id,
            change_amount: second.toAmount,
        }
    }

    return null
}
