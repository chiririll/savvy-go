/** A point that pins an account's balance to the real one from `date` on. Money is in the account's currency. */
export interface ReconciliationPoint {
    id: number
    accountId: number
    date: string
    actualBalance: number
    /** What the ledger expected at the date, before the point. */
    expectedBalance: number
    /** `actualBalance - expectedBalance`. */
    difference: number
    note: string | null
    /** The difference in the base currency; null when the currency has no rate. */
    differenceBase: number | null
}

/** One account of a reconciliation request. */
export interface ReconciliationItemInput {
    accountId: number
    actualBalance: number
    note?: string | null
}

/** Reconciles accounts on a date; all or none is stored. */
export interface ReconciliationInput {
    date: string
    items: ReconciliationItemInput[]
}

/** The points of a day, for the transaction list. */
export interface ReconciliationDay {
    date: string
    /** Sum of the day's differences in the base currency, without the unconverted points. */
    differenceBase: number
    /** A point had no rate and is left out of `differenceBase`. */
    unconverted: boolean
    points: ReconciliationPoint[]
}

export interface ReconciliationFilters {
    account_ids?: number[]
    start_date?: string
    end_date?: string
}
