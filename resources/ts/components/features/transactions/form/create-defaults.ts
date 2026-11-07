import type { TransactionFormDefaults } from './use-transaction-form'

/** What a button or a link can ask the form to open with. */
export interface CreateTransactionDefaults {
    type?: 'income' | 'expense' | 'transfer' | 'transfer_out'
    account_id?: number
    /** A transfer: the account that receives. */
    to_account_id?: number
    amount?: number
    category_id?: number
    description?: string
}

/** The form values behind what an opener asked for: the account and the amount are the first payment. */
export function fromCreateDefaults(next?: CreateTransactionDefaults): TransactionFormDefaults | undefined {
    if (!next) {
        return undefined
    }
    const { type, account_id, to_account_id, amount, category_id, description } = next
    return {
        ...(type ? { type } : {}),
        ...(description !== undefined ? { description } : {}),
        ...(to_account_id !== undefined ? { to_account_id } : {}),
        ...(category_id !== undefined ? { default_category_id: category_id } : {}),
        ...(account_id !== undefined || amount !== undefined
            ? { payments: [{ account_id: account_id ?? 0, amount: (amount ?? '') as number }] }
            : {}),
    }
}
