import type { CardStatusLevel, CreditCardInfo } from '@/types'

/** How much of the limit is owed, 0 to 100. Money owed is `used`; a credit balance or no limit is 0. */
export function cardProgress(card: CreditCardInfo): number {
    if (card.limit <= 0 || card.used <= 0) {
        return 0
    }
    return Math.min(100, (card.used / card.limit) * 100)
}

/** The colour a card is drawn in: the level the backend computed. */
export const cardTone = (card: CreditCardInfo): CardStatusLevel => card.status.level
