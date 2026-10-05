import { Account } from './accounts'

export const DEBT_TYPE_VALUES = ['i_owe', 'owed_to_me'] as const
export type DebtType = (typeof DEBT_TYPE_VALUES)[number]

/** A debt account; currentBalance is the amount still owed. */
export interface Debt extends Account {
    type: 'debt'
    debtType: DebtType
    targetAmount: number
    paymentProgress: number
    dueDate: string | null
    counterparty: string | null
    description: string | null
    isPaidOff: boolean
}

export interface DebtSummary {
    totalIOwe: number
    totalOwedToMe: number
    netDebt: number
    debtsCount: number
    currency: string | null
    decimals: number
}

export interface DebtsResponse {
    data: Debt[]
    summary?: DebtSummary
}
