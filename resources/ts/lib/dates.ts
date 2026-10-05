/** Parse an ISO date key (YYYY-MM-DD) without UTC timezone shift. Ignores any trailing time
 *  component (legacy-imported rows store dates as "YYYY-MM-DD HH:MM:SS"). */
export function parseDateKey(dateKey: string): Date {
    const [year, month, day] = dateKey.slice(0, 10).split('-').map(Number)
    return new Date(year, (month || 1) - 1, day || 1)
}

export function formatYearMonth(date: Date = new Date()): string {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    return `${year}-${month}`
}

export function formatDateLocal(date: Date = new Date()): string {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
}

export function addDaysLocal(date: Date, days: number): string {
    const next = new Date(date)
    next.setDate(next.getDate() + days)
    return formatDateLocal(next)
}

export function isDateInFuture(date?: string | null): boolean {
    if (!date) {
        return false
    }

    return date > formatDateLocal()
}

export function isDateOverdue(date?: string | null): boolean {
    return Boolean(date && date < formatDateLocal())
}

export function formatTransactionGroupHeading(
    dateKey: string | null,
    locale: string,
    labels: { today: string; yesterday: string; tomorrow?: string; noDate: string },
): string {
    if (!dateKey) {
        return labels.noDate
    }

    const today = formatDateLocal()
    if (dateKey === today) {
        return labels.today
    }
    if (dateKey === addDaysLocal(new Date(), -1)) {
        return labels.yesterday
    }
    if (labels.tomorrow && dateKey === addDaysLocal(new Date(), 1)) {
        return labels.tomorrow
    }

    const date = parseDateKey(dateKey)
    const options: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'long' }
    if (date.getFullYear() !== new Date().getFullYear()) {
        options.year = 'numeric'
    }

    return date.toLocaleDateString(locale, options)
}

export function groupByDateKey<T extends { date: string | null }>(items: T[]): { date: string | null; items: T[] }[] {
    const groups: { date: string | null; items: T[] }[] = []
    const index = new Map<string, number>()

    for (const item of items) {
        const key = item.date ?? ''
        const existing = index.get(key)
        if (existing !== undefined) {
            groups[existing].items.push(item)
        } else {
            index.set(key, groups.length)
            groups.push({ date: item.date, items: [item] })
        }
    }

    return groups
}

/** Overdue → red, within the next 3 days → yellow, later or unset → muted. */
export function pendingDateClassName(dateKey?: string | null): string {
    if (!dateKey) {
        return 'text-muted-foreground'
    }

    const today = formatDateLocal()
    if (dateKey < today) {
        return 'text-red-600'
    }
    if (dateKey <= addDaysLocal(new Date(), 3)) {
        return 'text-yellow-600'
    }
    return 'text-muted-foreground'
}
