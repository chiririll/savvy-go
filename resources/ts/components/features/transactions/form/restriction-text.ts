import type { RestrictionReason } from '@/types'

/** The text of a restriction, in the user's language; `t` is any i18next translate function. */
export function restrictionText(t: (key: string) => string, reason: RestrictionReason): string {
    switch (reason) {
        case 'debt':
            return t('forms:transactions.restriction.debt')
        case 'space_transfer':
            return t('forms:transactions.restriction.space_transfer')
        case 'transfer_review':
            return t('forms:transactions.restriction.transfer_review')
        case 'recurring_pending':
            return t('forms:transactions.restriction.recurring_pending')
    }
}
