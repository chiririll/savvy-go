import type { SpaceTransfer, TransferSide } from '@/types/spaces'

/** A transfer someone has to decide on: in review, missing an account, unverifiable or frozen. */
export function transferNeedsDecision(tr: SpaceTransfer): boolean {
    return tr.status === 'in_review' || tr.needsAttention || !tr.verified || tr.frozen
}

/** `−200.00 EUR`: the amount of one side with its own currency, never summed with the other side. */
export function sideAmount(side: TransferSide, sign: '−' | '+'): string {
    return `${sign}${side.amount} ${side.currency}`
}

interface Sides {
    from: TransferSide
    to: TransferSide
}

/** The side that belongs to this space first, then the other space's. */
export function sidesForSpace(sides: Sides, outgoing: boolean): [string, string] {
    return outgoing
        ? [sideAmount(sides.from, '−'), sideAmount(sides.to, '+')]
        : [sideAmount(sides.to, '+'), sideAmount(sides.from, '−')]
}
