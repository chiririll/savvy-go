import i18n from '@/lib/i18n'
import { localizeDefaultName } from '@/lib/localized-name'
import type { Transaction, TransactionStatus, TransactionType } from '@/types/transactions'
import { transactionTypeLabel } from '@/lib/labels'

const INCOMING_TYPES: TransactionType[] = ['income', 'debt_collection', 'debt_borrow', 'transfer_in']
const OUTGOING_TYPES: TransactionType[] = ['expense', 'debt_payment', 'debt_lend', 'transfer_out']

const STORED_MESSAGE_KEY = /^messages\.[a-z0-9_.]+$/i

export function displayTransactionDescription(
    transaction: Pick<Transaction, 'description' | 'type' | 'category' | 'account' | 'toAccount'>
): string {
    const stored = transaction.description?.trim()
    if (stored && !STORED_MESSAGE_KEY.test(stored)) {
        return stored
    }

    const name = transaction.toAccount?.name ?? transaction.account.name

    switch (transaction.type) {
        case 'transfer':
            return transaction.toAccount
                ? `${transaction.account.name} → ${transaction.toAccount.name}`
                : transaction.account.name
        case 'debt_payment':
            return i18n.t('pages:transactions.fallback.payment', { name })
        case 'debt_collection':
            return i18n.t('pages:transactions.fallback.collection', { name })
        case 'debt_lend':
            return i18n.t('pages:transactions.fallback.lend', { name })
        case 'debt_borrow':
            return i18n.t('pages:transactions.fallback.borrow', { name })
        default:
            return localizeDefaultName(transaction.category?.name) || transactionTypeLabel(i18n.t, transaction.type)
    }
}

export function transactionSubtitle(
    transaction: Pick<Transaction, 'account' | 'category'>,
): string {
    const accountName = transaction.account.name
    const categoryName = transaction.category
        ? localizeDefaultName(transaction.category.name)
        : null

    return categoryName ? `${accountName} · ${categoryName}` : accountName
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
        return { sign: '+', className: 'text-green-600' }
    }
    if (type === 'transfer') {
        return { sign: '', className: 'text-blue-600' }
    }
    if (OUTGOING_TYPES.includes(type)) {
        return { sign: '-', className: 'text-red-600' }
    }
    return { sign: '', className: 'text-foreground' }
}
