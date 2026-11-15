import { decodeAccountCurrency } from '@/schemas'
import { type CurrencySelectValue } from '@/components/shared/pickers/CurrencySelect'

export function toSelectValue(currency: string): CurrencySelectValue | null {
    const decoded = decodeAccountCurrency(currency)
    if (decoded.currency_code) {
        return { source: 'catalog', code: decoded.currency_code }
    }
    if (decoded.currency_id) {
        return { source: 'existing', id: decoded.currency_id }
    }
    return null
}
