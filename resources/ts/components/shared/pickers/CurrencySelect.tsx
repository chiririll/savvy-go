import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronDown, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { FormControl } from '@/components/ui/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useCurrencies, useCurrencyCatalog } from '@/hooks'
import { cn } from '@/lib/utils'
import type { Currency, CurrencyCatalogItem } from '@/types'

export type CurrencySelectValue = { source: 'existing'; id: number } | { source: 'catalog'; code: string }

interface CurrencySelectProps {
    value?: CurrencySelectValue | null
    onChange: (value: CurrencySelectValue) => void
    allowCatalog?: boolean
    disabled?: boolean
    placeholder?: string
}

function matchesQuery(query: string, code: string, name: string): boolean {
    if (!query) {
        return true
    }

    const q = query.toLowerCase()

    return code.toLowerCase().includes(q) || name.toLowerCase().includes(q)
}

function CurrencyRow({
    code,
    name,
    symbol,
    selected,
    isNew,
    onSelect,
}: {
    code: string
    name: string
    symbol: string
    selected: boolean
    isNew?: boolean
    onSelect: () => void
}) {
    return (
        <button
            type="button"
            className={cn(
                'flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm',
                'hover:bg-accent hover:text-accent-foreground',
                selected && 'bg-accent',
            )}
            onClick={onSelect}
        >
            <Check className={cn('size-4 shrink-0', selected ? 'opacity-100' : 'opacity-0')} />
            <span className="min-w-0 flex-1 truncate">
                <span className="font-mono font-medium">{code}</span>
                <span className="text-muted-foreground">
                    {' '}
                    · {symbol} · {name}
                </span>
            </span>
            {isNew && <Plus className="size-3.5 shrink-0 text-muted-foreground" />}
        </button>
    )
}

export function CurrencySelect({ value, onChange, allowCatalog = true, disabled, placeholder }: CurrencySelectProps) {
    const { t } = useTranslation('forms')
    const { data: currencies, isLoading } = useCurrencies()
    const { data: catalog = [] } = useCurrencyCatalog(allowCatalog)
    const [open, setOpen] = useState(false)
    const [query, setQuery] = useState('')

    const selectedExisting =
        value?.source === 'existing' ? currencies?.find((currency) => currency.id === value.id) : undefined
    const selectedCatalog = value?.source === 'catalog' ? catalog.find((item) => item.code === value.code) : undefined

    const existing = useMemo(
        () => (currencies ?? []).filter((currency) => matchesQuery(query, currency.code, currency.name)),
        [currencies, query],
    )

    const catalogItems = useMemo(() => {
        if (!allowCatalog) {
            return []
        }

        return catalog.filter((item) => matchesQuery(query, item.code, item.name))
    }, [allowCatalog, catalog, query])

    const label = selectedExisting
        ? formatCurrencyLabel(selectedExisting)
        : selectedCatalog
          ? formatCurrencyLabel(selectedCatalog)
          : value?.source === 'catalog'
            ? value.code
            : null

    const pick = (next: CurrencySelectValue) => {
        onChange(next)
        setOpen(false)
        setQuery('')
    }

    return (
        <Popover
            modal
            open={open}
            onOpenChange={(next) => {
                setOpen(next)
                if (!next) {
                    setQuery('')
                }
            }}
        >
            <PopoverTrigger asChild>
                <FormControl>
                    <Button
                        type="button"
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        disabled={disabled || isLoading}
                        className={cn('w-full justify-between font-normal', !label && 'text-muted-foreground')}
                    >
                        <span className="truncate">{label ?? placeholder ?? t('selectCurrency')}</span>
                        <ChevronDown className="size-4 shrink-0 opacity-50" />
                    </Button>
                </FormControl>
            </PopoverTrigger>
            <PopoverContent
                align="start"
                aria-label={t('selectCurrency')}
                className="w-[var(--radix-popover-trigger-width)] min-w-[var(--radix-popover-trigger-width)] p-0"
            >
                <div className="border-b p-2">
                    <Input
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder={t('searchCurrency')}
                        autoComplete="off"
                    />
                </div>
                <div className="max-h-64 overflow-auto p-1">
                    {existing.length > 0 && (
                        <div>
                            {allowCatalog && (
                                <p className="px-2 py-1.5 text-xs text-muted-foreground">{t('currencyYour')}</p>
                            )}
                            {existing.map((currency) => (
                                <CurrencyRow
                                    key={currency.id}
                                    code={currency.code}
                                    name={currency.name}
                                    symbol={currency.symbol}
                                    selected={value?.source === 'existing' && value.id === currency.id}
                                    onSelect={() => pick({ source: 'existing', id: currency.id })}
                                />
                            ))}
                        </div>
                    )}
                    {catalogItems.length > 0 && (
                        <div>
                            <p className="px-2 py-1.5 text-xs text-muted-foreground">{t('currencyCatalog')}</p>
                            {catalogItems.map((item) => (
                                <CurrencyRow
                                    key={item.code}
                                    code={item.code}
                                    name={item.name}
                                    symbol={item.symbol}
                                    selected={value?.source === 'catalog' && value.code === item.code}
                                    isNew
                                    onSelect={() => pick({ source: 'catalog', code: item.code })}
                                />
                            ))}
                        </div>
                    )}
                    {existing.length === 0 && catalogItems.length === 0 && (
                        <p className="px-2 py-6 text-center text-sm text-muted-foreground">{t('currencyEmpty')}</p>
                    )}
                </div>
            </PopoverContent>
        </Popover>
    )
}

function formatCurrencyLabel(item: Currency | CurrencyCatalogItem): string {
    return `${item.code} · ${item.symbol} · ${item.name}`
}
