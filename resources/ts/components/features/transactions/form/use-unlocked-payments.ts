import { useCallback, useEffect, useRef, useState } from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
import { followedAmount, lockBlocker, type LockValues } from '@/lib/transaction/payment-lock'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'

export interface PaymentLock {
    /** Whether the row follows the lines: its amount is theirs and cannot be typed. */
    follows: boolean
    /** Why the row cannot follow the lines, or null when it can. */
    blocker: 'several' | 'noLine' | 'foreignChange' | null
    toggle: () => void
}

/**
 * Which payment rows follow the lines instead of being fixed. A row follows from the start: it takes the sum of the
 * lines in its currency whenever they change. The exception is a transaction saved with lines that do not add up to
 * its payment (the rest went to the default category): that row opens fixed, or opening it would change what was
 * saved. The lock is not stored. When a row that follows can no longer because its lines are gone, it is fixed again
 * at the amount it had before it followed; when something else stops it (another payment in its currency, change in
 * another currency), it is fixed where it is. `keys` are the ids of the rows from the field array, so a lock stays with its row
 * when another is removed.
 */
export function useUnlockedPayments(accounts: Account[] | undefined, keys: string[]): Array<PaymentLock> | null {
    const { control, getValues, setValue } = useFormContext<TransactionFormValues>()
    const [payments, change, lines] = useWatch({ control, name: ['payments', 'change', 'lines'] })
    const [following, setFollowing] = useState<string[]>([])
    const before = useRef(new Map<string, unknown>())
    // The rows whose start was settled; forgotten when the lines are gone, so the next lines settle it again.
    const settled = useRef(new Set<string>())

    const values: LockValues = { payments: payments ?? [], change: change ?? [], lines: lines ?? [] }
    const list = accounts ?? []
    const hasLines = (lines?.length ?? 0) > 0
    const keyAt = keys.join('|')

    useEffect(() => {
        if (!hasLines) {
            settled.current.clear()
            if (following.length > 0) {
                keys.forEach((key, index) => {
                    if (following.includes(key) && before.current.has(key)) {
                        setValue(`payments.${index}.amount`, before.current.get(key) as number)
                    }
                })
                setFollowing([])
            }
            return
        }
        let next = following.filter((key) => keys.includes(key))
        keys.forEach((key, index) => {
            const amount = followedAmount(values, list, index)
            const current = getValues(`payments.${index}.amount`)
            if (!settled.current.has(key)) {
                if (amount === null) {
                    return
                }
                settled.current.add(key)
                const blank = String(current ?? '').trim() === ''
                if (blank || Number(current) === Number(amount)) {
                    before.current.set(key, current)
                    next = [...next, key]
                }
            }
            if (!next.includes(key)) {
                return
            }
            if (amount === null) {
                next = next.filter((k) => k !== key)
                if (lockBlocker(values, list, index) === 'noLine' && before.current.has(key)) {
                    setValue(`payments.${index}.amount`, before.current.get(key) as number)
                }
            } else if (String(current ?? '') !== amount) {
                setValue(`payments.${index}.amount`, amount as unknown as number, { shouldValidate: true })
            }
        })
        if (next.length !== following.length || next.some((key, i) => key !== following[i])) {
            setFollowing(next)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [following, payments, change, lines, keyAt, accounts])

    const toggle = useCallback(
        (key: string, index: number) => {
            if (following.includes(key)) {
                setFollowing(following.filter((k) => k !== key))
                return
            }
            before.current.set(key, getValues(`payments.${index}.amount`))
            setFollowing([...following, key])
        },
        [following, getValues],
    )

    if (!hasLines) {
        return null
    }
    return keys.map((key, index) => {
        const blocker = lockBlocker(values, list, index)
        return { follows: following.includes(key) && !blocker, blocker, toggle: () => toggle(key, index) }
    })
}
