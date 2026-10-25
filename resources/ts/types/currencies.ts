import { BaseEntity } from './api'

export interface Currency extends BaseEntity {
    code: string
    name: string
    symbol: string
    decimals: number
    isBase: boolean
    rate: number
}

export interface CurrencyCatalogItem {
    code: string
    name: string
    symbol: string
    decimals: number
    rate: number | null
}

/** The server-wide catalog; `source` is "fallback" while it is the short built-in list. */
export interface CurrencyCatalog {
    items: CurrencyCatalogItem[]
    source: 'live' | 'fallback' | string
}
