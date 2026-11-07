import type { RestrictionReason, Transaction, TransactionInput } from '@/types'
import { canEditInForm } from './transaction-form-values'

/** How the form opens a saved transaction: whole, or only its details. */
export type FormMode = 'full' | 'details'

/**
 * The form a saved transaction opens in. Whole when the server lets it be changed and the form can show all of it.
 * Details only when the server allows no more than that (a debt, a side of a transfer between spaces), or when the
 * form cannot show the transaction without dropping something: the description, tags and the categories and notes
 * of its lines are still safe to change. Null when nothing can change.
 */
export function formModeOf(transaction: Transaction): FormMode | null {
    if (transaction.actions.edit && canEditInForm(transaction)) {
        return 'full'
    }
    return transaction.actions.editDetails ? 'details' : null
}

/** Why a transaction that cannot be changed is refused: its first restriction, if the server gave one. */
export function refusalReason(transaction: Transaction): RestrictionReason | null {
    return transaction.restrictions[0]?.reason ?? null
}

/** A change to a saved transaction: what it leaves out stays as it is. */
export type TransactionPatch = Partial<TransactionInput>

/** What the restricted form edits. */
export interface DetailsFormValues {
    description: string
    tag_ids: number[]
    /** One per line of the transaction, in its order. */
    lines: Array<{ category_id: number | null; note: string | null }>
}

const inOrder = (transaction: Transaction) => [...transaction.lines].sort((a, b) => a.seq - b.seq)

export function toDetailsValues(transaction: Transaction): DetailsFormValues {
    return {
        description: transaction.description ?? '',
        tag_ids: transaction.tags.map((tag) => tag.id),
        lines: inOrder(transaction).map((l) => ({ category_id: l.categoryId, note: l.note })),
    }
}

/**
 * The change to send for the restricted form. It names no payments, type or date, so they stay as they are; the
 * lines go with the amount and currency they have, since the server refuses a change of either in this mode. The
 * tags always go, because a save without them clears them.
 */
export function buildDetailsInput(transaction: Transaction, values: DetailsFormValues): TransactionPatch {
    const input: TransactionPatch = {
        description: values.description.trim(),
        tag_ids: values.tag_ids,
    }
    const lines = inOrder(transaction)
    if (lines.length > 0) {
        input.lines = lines.map((l, index) => ({
            category_id: values.lines[index]?.category_id ?? null,
            currency_id: l.currencyId,
            amount: l.amount,
            note: values.lines[index]?.note?.trim() || null,
        }))
    }
    return input
}
