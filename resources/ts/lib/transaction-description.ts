import i18n from '@/lib/i18n'
import { localizeDefaultName } from '@/lib/localized-name'
import { primaryCategory } from '@/lib/transaction-money'
import { isServiceAccountType } from '@/types/accounts'
import type { Account } from '@/types/accounts'
import type { Transaction, TransactionStatus, TransactionType } from '@/types/transactions'
import { transactionTypeLabel } from '@/lib/labels'

const INCOMING_TYPES: TransactionType[] = ['income', 'debt_collection', 'debt_borrow', 'transfer_in']
const OUTGOING_TYPES: TransactionType[] = ['expense', 'debt_payment', 'debt_lend', 'transfer_out']

const STORED_MESSAGE_KEY = /^messages\.[a-z0-9_.]+$/i

type Described = Pick<Transaction, 'payments' | 'lines'>

/** Accounts of the payments, ordinary ones first and each once; a debt or space account only when nothing else paid. */
function paymentAccounts(transaction: Described): Account[] {
    const accounts = transaction.payments.flatMap((p) => (p.account ? [p.account] : []))
    const ordinary = accounts.filter((a) => !isServiceAccountType(a.type))
    return [...new Map((ordinary.length > 0 ? ordinary : accounts).map((a) => [a.id, a])).values()]
}

/** The account a debt transaction is about; the first account when none of the payments is on a debt. */
function debtName(transaction: Described): string {
    const accounts = transaction.payments.flatMap((p) => (p.account ? [p.account] : []))
    return (accounts.find((a) => a.type === 'debt') ?? accounts[0])?.name ?? ''
}

/** The sides of a transfer: the account that was paid from and the one that was paid to. */
function transferEnds(transaction: Described): { from: Account | null; to: Account | null } {
    const ordinary = transaction.payments.filter((p) => p.account && !isServiceAccountType(p.account.type))
    return {
        from: ordinary.find((p) => p.amount < 0)?.account ?? null,
        to: ordinary.find((p) => p.amount > 0)?.account ?? null,
    }
}

export function displayTransactionDescription(
    transaction: Pick<Transaction, 'description' | 'type' | 'payments' | 'lines'>,
): string {
    const stored = transaction.description?.trim()
    if (stored && !STORED_MESSAGE_KEY.test(stored)) {
        return stored
    }

    switch (transaction.type) {
        case 'transfer': {
            const { from, to } = transferEnds(transaction)
            if (from && to) {
                return `${from.name} → ${to.name}`
            }
            return (from ?? to ?? paymentAccounts(transaction)[0])?.name ?? ''
        }
        case 'debt_payment':
            return i18n.t('pages:transactions.fallback.payment', { name: debtName(transaction) })
        case 'debt_collection':
            return i18n.t('pages:transactions.fallback.collection', { name: debtName(transaction) })
        case 'debt_lend':
            return i18n.t('pages:transactions.fallback.lend', { name: debtName(transaction) })
        case 'debt_borrow':
            return i18n.t('pages:transactions.fallback.borrow', { name: debtName(transaction) })
        default:
            return (
                localizeDefaultName(primaryCategory(transaction)?.name) ||
                transactionTypeLabel(i18n.t, transaction.type)
            )
    }
}

/** The number of different categories among the lines. */
export function categoryCount(transaction: Pick<Transaction, 'lines'>): number {
    return new Set(transaction.lines.flatMap((l) => (l.categoryId == null ? [] : [l.categoryId]))).size
}

/** The first account, and "+N" for the other accounts that paid. */
export function accountLabel(transaction: Described): string {
    const [first, ...rest] = paymentAccounts(transaction)
    if (!first) {
        return ''
    }
    return rest.length > 0 ? `${first.name} +${rest.length}` : first.name
}

/** "Account · category", or "Account · N categories" for several. Notes never show in a row. */
export function transactionSubtitle(transaction: Described): string {
    const account = accountLabel(transaction)
    const count = categoryCount(transaction)
    const category =
        count > 1
            ? i18n.t('pages:transactions.categoriesCount', { count })
            : localizeDefaultName(primaryCategory(transaction)?.name)

    return [account, category].filter(Boolean).join(' · ')
}

export function transactionAmountAppearance(
    type: TransactionType,
    status?: TransactionStatus,
): {
    sign: '+' | '-' | ''
    className: string
} {
    if (status === 'skipped') {
        return { sign: '', className: 'text-muted-foreground' }
    }
    if (INCOMING_TYPES.includes(type)) {
        return { sign: '+', className: 'text-success' }
    }
    if (type === 'transfer') {
        return { sign: '', className: 'text-info' }
    }
    if (OUTGOING_TYPES.includes(type)) {
        return { sign: '-', className: 'text-destructive' }
    }
    return { sign: '', className: 'text-foreground' }
}
