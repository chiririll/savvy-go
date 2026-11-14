import type { Account } from '@/types'

/** The account with the id, which may arrive as a numeric string from a select; undefined when there is none. */
export const accountOf = (accounts: Account[], id: unknown): Account | undefined => accounts.find((a) => a.id === Number(id))
