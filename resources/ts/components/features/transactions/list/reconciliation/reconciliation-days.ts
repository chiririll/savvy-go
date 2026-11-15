import { fromUnits, numberToAmount, toUnits } from '@/lib/format/decimal-amount'
import type { ReconciliationDay } from '@/types'

/** A day of the list: its transactions (none for a day that only has reconciliation points) and its points. */
export interface DayGroup<T> {
    date: string | null
    items: T[]
    reconciliation?: ReconciliationDay
}

const dayKey = (date: string) => date.slice(0, 10)

/**
 * The days narrowed to the accounts. With no account chosen the server's days are kept as they are. Otherwise the
 * sum is made again from the kept points, exactly (in units of the base currency); a point without its base
 * value makes the day `unconverted`. Days left with no points go.
 */
export function filterDaysByAccounts(
    days: readonly ReconciliationDay[],
    accountIds: readonly number[],
    baseDecimals: number,
): ReconciliationDay[] {
    if (accountIds.length === 0) return [...days]
    const chosen = new Set(accountIds)
    const kept: ReconciliationDay[] = []
    for (const day of days) {
        const points = day.points.filter((point) => chosen.has(point.accountId))
        if (points.length === 0) continue
        let units = 0n
        let unconverted = false
        for (const point of points) {
            const value = point.differenceBase === null ? null : toUnits(numberToAmount(point.differenceBase, baseDecimals), baseDecimals)
            if (value === null) unconverted = true
            else units += value
        }
        kept.push({ ...day, points, unconverted, differenceBase: Number(fromUnits(units, baseDecimals)) })
    }
    return kept
}

/** The first and last date of the groups that have one, or null when none has. */
export function loadedRange(groups: readonly { date: string | null }[]): { start: string; end: string } | null {
    const dates = groups.flatMap((group) => (group.date ? [dayKey(group.date)] : []))
    if (dates.length === 0) return null
    return { start: dates.reduce((a, b) => (a < b ? a : b)), end: dates.reduce((a, b) => (a > b ? a : b)) }
}

const EARLIEST = '1900-01-01'
const LATEST = '2999-12-31'

/**
 * The span of days whose points the list shows. The loaded transactions bound it only on the side that more pages
 * still follow: the first page opens the newest end (or the oldest, when ascending), and the last page the other.
 * So a day with points and no transactions is shown also before the first or after the last transaction, and when
 * there are no transactions at all.
 */
export function reconciliationRange(
    loaded: { start: string; end: string } | null,
    newestFirst: boolean,
    hasNextPage: boolean,
): { start: string; end: string } {
    const near = newestFirst ? LATEST : EARLIEST
    const far = newestFirst ? EARLIEST : LATEST
    const bound = newestFirst ? loaded?.start : loaded?.end
    const farEnd = hasNextPage && bound ? bound : far
    return newestFirst ? { start: farEnd, end: near } : { start: near, end: farEnd }
}

/** Whether the groups run from the newest day to the oldest (the default when they are too few to tell). */
function isNewestFirst(dates: readonly string[]): boolean {
    return dates.length < 2 || dates[0] >= dates[dates.length - 1]
}

/**
 * Puts the points on the day groups, in the order of the list. A day with points and no transactions gets a group
 * of its own; only days inside the range of the loaded groups count. Undated groups stay last. `newestFirst` is the
 * direction of the list when the caller knows it; the groups' own order tells it otherwise.
 */
export function mergeReconciliationDays<T>(
    groups: readonly { date: string | null; items: T[] }[],
    days: readonly ReconciliationDay[],
    newestFirst?: boolean,
    allowed?: { start: string; end: string },
): DayGroup<T>[] {
    const range = allowed ?? loadedRange(groups)
    const byDate = new Map(days.map((day) => [dayKey(day.date), day]))
    const dated: DayGroup<T>[] = groups
        .filter((group) => group.date !== null)
        .map((group) => ({ ...group, reconciliation: byDate.get(dayKey(group.date as string)) }))
    const undated: DayGroup<T>[] = groups.filter((group) => group.date === null)
    if (!range) return [...dated, ...undated]

    const known = new Set(dated.map((group) => dayKey(group.date as string)))
    const extra = days
        .filter((day) => !known.has(dayKey(day.date)) && dayKey(day.date) >= range.start && dayKey(day.date) <= range.end)
        .map((day) => ({ date: day.date, items: [] as T[], reconciliation: day }))
    const sign = (newestFirst ?? isNewestFirst(dated.map((group) => dayKey(group.date as string)))) ? -1 : 1
    const all = [...dated, ...extra].sort((a, b) => sign * dayKey(a.date as string).localeCompare(dayKey(b.date as string)))
    return [...all, ...undated]
}

export type ReconciledTone = 'calm' | 'positive' | 'negative'

export interface ReconciledLabel {
    /** The signed, shortened sum in the base currency; null when the day is exact. */
    amount: string | null
    tone: ReconciledTone
}

/** What the line of a day says. `format` shortens a non-negative amount (formatCurrencyCompact). */
export function reconciledLabel(day: ReconciliationDay, format: (absolute: number) => string): ReconciledLabel {
    if (day.differenceBase === 0) return { amount: null, tone: 'calm' }
    const positive = day.differenceBase > 0
    return {
        amount: `${positive ? '+' : '-'}${format(Math.abs(day.differenceBase))}`,
        tone: positive ? 'positive' : 'negative',
    }
}

/** The exact difference of a point as text with a sign, in the currency of its account. */
export function signedAmount(value: number, format: (value: number) => string): string {
    return `${value > 0 ? '+' : ''}${format(value)}`
}
