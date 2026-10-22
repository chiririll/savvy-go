import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { testId } from '@/lib/test-id'
import { cn } from '@/lib/utils'
import type { CurrencyCatalogItem } from '@/types'
import { manualCode, searchCurrencies } from './currency-search'

interface Props {
    catalog: readonly CurrencyCatalogItem[]
    /** The chosen code; null shows the placeholder. For adding, leave it null so the trigger stays a button. */
    value?: string | null
    onChange: (code: string) => void
    /** Codes that cannot be picked, such as the ones already chosen. */
    exclude?: ReadonlySet<string>
    popular?: readonly string[]
    /** Any currency code may be typed, because the list is the short built-in one. */
    allowManual?: boolean
    placeholder: string
    disabled?: boolean
    /** For a label that names the picker; without one, the placeholder names it. */
    id?: string
    className?: string
    'data-testid'?: string
}

/**
 * A currency picker that searches by code, name and symbol, with a capped list
 * that scrolls. An option reads "CODE name", so it can be found by its code in
 * any language.
 */
export function CurrencyCombobox({
    catalog,
    value = null,
    onChange,
    exclude,
    popular,
    allowManual,
    placeholder,
    disabled,
    id,
    className,
    ...rest
}: Props) {
    const { t } = useTranslation('settings')
    const [open, setOpen] = useState(false)
    const [query, setQuery] = useState('')
    const { shown, total } = useMemo(
        () => searchCurrencies(catalog, query, { exclude, popular }),
        [catalog, query, exclude, popular],
    )
    const manual = allowManual ? manualCode(catalog, query) : null
    const chosen = value ? catalog.find((c) => c.code === value) : undefined

    const pick = (code: string) => {
        onChange(code)
        setOpen(false)
        setQuery('')
    }

    return (
        <Popover
            modal
            open={open}
            onOpenChange={(next) => {
                setOpen(next)
                if (!next) setQuery('')
            }}
        >
            <PopoverTrigger asChild>
                <Button
                    type="button"
                    id={id}
                    variant="outline"
                    role="combobox"
                    aria-label={id ? undefined : placeholder}
                    aria-expanded={open}
                    disabled={disabled}
                    className={cn('justify-between font-normal', !value && 'text-muted-foreground', className)}
                    data-testid={rest['data-testid']}
                >
                    <span className="truncate">
                        {value ? (
                            <>
                                <span className="font-mono">{value}</span>
                                {chosen && <span className="text-muted-foreground"> {chosen.name}</span>}
                            </>
                        ) : (
                            placeholder
                        )}
                    </span>
                    <ChevronDown className="size-4 shrink-0 opacity-50" />
                </Button>
            </PopoverTrigger>
            <PopoverContent
                align="start"
                aria-label={placeholder}
                className="w-[var(--radix-popover-trigger-width)] min-w-64 p-0"
            >
                <div className="border-b p-2">
                    <Input
                        autoFocus
                        {...testId('currency-search')}
                        autoComplete="off"
                        value={query}
                        placeholder={t('spaces.create.currencies.search')}
                        aria-label={t('spaces.create.currencies.search')}
                        onChange={(e) => setQuery(e.target.value)}
                    />
                </div>
                <div className="max-h-64 overflow-auto p-1">
                    <div role="listbox" aria-label={placeholder}>
                        {manual && (
                            <button
                                type="button"
                                role="option"
                                aria-selected={false}
                                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
                                onClick={() => pick(manual)}
                            >
                                <span className="font-mono">{manual}</span>
                                <span className="text-muted-foreground">{t('spaces.create.currencies.useCode')}</span>
                            </button>
                        )}
                        {shown.map((c) => (
                            <button
                                key={c.code}
                                type="button"
                                role="option"
                                aria-selected={c.code === value}
                                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent hover:text-accent-foreground"
                                onClick={() => pick(c.code)}
                            >
                                <Check
                                    className={cn('size-4 shrink-0', c.code === value ? 'opacity-100' : 'opacity-0')}
                                />
                                <span className="min-w-0 flex-1 truncate">
                                    <span className="font-mono">{c.code}</span>{' '}
                                    <span className="text-muted-foreground">{c.name}</span>
                                </span>
                                <span className="shrink-0 text-xs text-muted-foreground">{c.symbol}</span>
                            </button>
                        ))}
                    </div>
                    {shown.length === 0 && !manual && (
                        <p className="px-2 py-6 text-center text-sm text-muted-foreground">
                            {t('spaces.create.currencies.none')}
                        </p>
                    )}
                    {total > shown.length && (
                        <p className="px-2 py-1.5 text-center text-xs text-muted-foreground">
                            {t('spaces.create.currencies.more', {
                                count: total - shown.length,
                            })}
                        </p>
                    )}
                </div>
            </PopoverContent>
        </Popover>
    )
}
