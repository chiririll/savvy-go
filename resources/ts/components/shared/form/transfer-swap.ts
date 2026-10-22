function asAccountId(value: unknown): number | null {
    const id = Number(value)
    return Number.isFinite(id) && id > 0 ? id : null
}

export interface SwapUpdates {
    to_account_id: number | null
    account_id: number | null
    amount?: unknown
    to_amount?: unknown
    exchange_rate?: number
}

/**
 * The field changes that turn a transfer around, or null when neither account is set.
 * The amounts and the rate follow only when both accounts are known: the received amount
 * becomes the sent one, and the rate is inverted.
 */
export function swapUpdates(values: Record<string, unknown>): SwapUpdates | null {
    const from = asAccountId(values.account_id)
    const to = asAccountId(values.to_account_id)
    if (from == null && to == null) return null

    const updates: SwapUpdates = { to_account_id: from, account_id: to }
    if (from == null || to == null) return updates

    const { amount, to_amount: toAmount, exchange_rate: rate } = values
    if (toAmount != null && toAmount !== '') {
        updates.amount = toAmount
        updates.to_amount = amount
    }
    if (typeof rate === 'number' && rate > 0) {
        updates.exchange_rate = 1 / rate
    }
    return updates
}
