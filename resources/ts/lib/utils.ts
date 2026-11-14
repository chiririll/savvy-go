import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

export { formatBalanceShort, formatCurrency, formatCurrencyCompact } from '@/lib/format/currency'
export { formatDateLocal, formatYearMonth, addDaysLocal, isDateInFuture } from '@/lib/format/dates'
export { toggleIdInArray } from '@/lib/ids'
export { schemaResolver } from '@/lib/ui/form-resolver'
