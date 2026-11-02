import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'
import { transactionsApi } from '@/api'
import { CommandDialog } from '@/components/shared/dialogs/CommandDialog'
import { transactionSearchHref, useTransactionListFilters } from '@/hooks/use-transaction-list-filters'
import { cn, formatCurrency } from '@/lib/utils'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from '@/lib/transaction-description'
import {
    parseTransactionSearch,
    serializeTransactionSearch,
    suggestSearchCompletions,
    type ParsedSearch,
} from '@/lib/transaction-search'
import type { Transaction, TransactionFilters } from '@/types'
import { useSearchRefs } from './use-search-refs'

const PREVIEW_SIZE = 8
const DEBOUNCE_MS = 200

interface TransactionSearchPaletteProps {
    open: boolean
    onOpenChange: (open: boolean) => void
}

type Option =
    | { kind: 'completion'; id: string; label: string; insert: string }
    | { kind: 'result'; id: string; transaction: Transaction }
    | { kind: 'all'; id: string; total: number }

function hasFilters(search: ParsedSearch): boolean {
    return Boolean(
        search.q ||
            search.type ||
            search.accountIds.length ||
            search.currency.length ||
            search.categoryIds.length ||
            search.tagIds.length,
    )
}

function previewFilters(search: ParsedSearch): TransactionFilters & { with_summary: boolean } {
    return {
        status: 'confirmed',
        per_page: PREVIEW_SIZE,
        sort_by: 'date',
        sort_direction: 'desc',
        with_summary: true,
        q: search.q || undefined,
        type: search.type ?? undefined,
        account_ids: search.accountIds.length ? search.accountIds : undefined,
        currency: search.currency.length ? search.currency : undefined,
        category_ids: search.categoryIds.length ? search.categoryIds : undefined,
        tag_ids: search.tagIds.length ? search.tagIds : undefined,
    }
}

function useDebounced<T>(value: T, delay: number): T {
    const [debounced, setDebounced] = useState(value)
    useEffect(() => {
        const timer = setTimeout(() => setDebounced(value), delay)
        return () => clearTimeout(timer)
    }, [value, delay])
    return debounced
}

/** Search the transactions with keywords (`account:Card tags:food`) and jump to a result or to the filtered list. */
export function TransactionSearchPalette({ open, onOpenChange }: TransactionSearchPaletteProps) {
    const { t } = useTranslation('pages')
    const navigate = useNavigate()
    const { pathname } = useLocation()
    const { params } = useTransactionListFilters()
    const refs = useSearchRefs()
    const listId = useId()
    const [input, setInput] = useState('')
    const [active, setActive] = useState(0)
    const inputRef = useRef<HTMLInputElement>(null)

    // Opening on the list page shows the filters applied there, written as keywords.
    const wasOpen = useRef(false)
    useEffect(() => {
        if (open && !wasOpen.current) {
            setInput(
                pathname.endsWith('/transactions')
                    ? serializeTransactionSearch(
                          {
                              q: params.q ?? '',
                              type: params.type,
                              accountIds: params.accountIds,
                              currency: params.currency,
                              categoryIds: params.categoryIds,
                              tagIds: params.tagIds,
                          },
                          refs,
                      )
                    : '',
            )
            setActive(0)
        }
        wasOpen.current = open
    }, [open, pathname, params, refs])

    const parsed = useMemo(() => parseTransactionSearch(input, refs), [input, refs])
    const debounced = useDebounced(parsed, DEBOUNCE_MS)
    const filters = useMemo(() => previewFilters(debounced), [debounced])
    const searching = hasFilters(debounced)
    const { data, isFetching } = useQuery({
        queryKey: ['transactions', 'palette', filters],
        queryFn: ({ signal }) => transactionsApi.getAll(filters, signal),
        enabled: open && searching,
        placeholderData: keepPreviousData,
    })

    const completions = useMemo(() => suggestSearchCompletions(input, refs), [input, refs])
    const results = searching ? (data?.data ?? []) : []
    const total = data?.meta?.total ?? 0

    const options: Option[] = [
        ...completions.map<Option>((item) => ({ kind: 'completion', id: `c-${item.insert}`, ...item })),
        ...results.map<Option>((transaction) => ({ kind: 'result', id: `r-${transaction.id}`, transaction })),
        ...(searching && total > 0 ? [{ kind: 'all', id: 'all', total } as Option] : []),
    ]
    const activeIndex = Math.min(active, Math.max(options.length - 1, 0))

    const close = () => onOpenChange(false)
    const applyFilters = () => {
        if (hasFilters(parsed)) {
            navigate(transactionSearchHref(parsed))
        }
        close()
    }
    const accept = (option: Option) => {
        if (option.kind === 'completion') {
            setInput(option.insert)
            setActive(0)
            inputRef.current?.focus()
        } else if (option.kind === 'result') {
            navigate(`/transactions?edit=${option.transaction.id}`)
            close()
        } else {
            applyFilters()
        }
    }

    const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
        if (event.nativeEvent.isComposing) return
        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
            event.preventDefault()
            const step = event.key === 'ArrowDown' ? 1 : -1
            if (options.length) setActive((activeIndex + step + options.length) % options.length)
        } else if (event.key === 'Tab' && completions.length > 0) {
            event.preventDefault()
            const chosen = options[activeIndex]
            accept(chosen?.kind === 'completion' ? chosen : options[0])
        } else if (event.key === 'Enter') {
            event.preventDefault()
            const chosen = options[activeIndex]
            if (event.ctrlKey || event.metaKey || !chosen) applyFilters()
            else accept(chosen)
        }
    }

    const summary = data?.summary
    const activeId = options[activeIndex] ? `${listId}-${options[activeIndex].id}` : undefined

    return (
        <CommandDialog
            open={open}
            onOpenChange={onOpenChange}
            title={t('transactions.palette.title')}
            description={t('transactions.palette.description')}
            initialFocus={inputRef}
        >
                <div className="flex items-center gap-2 border-b px-3 pr-10">
                    <Search className="size-4 shrink-0 text-muted-foreground" />
                    <input
                        ref={inputRef}
                        role="combobox"
                        aria-expanded={options.length > 0}
                        aria-controls={listId}
                        aria-activedescendant={activeId}
                        aria-autocomplete="list"
                        aria-label={t('transactions.palette.title')}
                        value={input}
                        onChange={(event) => {
                            setInput(event.target.value)
                            setActive(0)
                        }}
                        onKeyDown={onKeyDown}
                        placeholder={t('transactions.palette.placeholder')}
                        autoComplete="off"
                        spellCheck={false}
                        className="h-12 w-full bg-transparent text-sm outline-hidden placeholder:text-muted-foreground"
                    />
                </div>

                <ul id={listId} role="listbox" aria-label={t('transactions.palette.title')} className="max-h-96 overflow-y-auto p-1">
                    {options.map((option, index) => (
                        <li
                            key={option.id}
                            id={`${listId}-${option.id}`}
                            role="option"
                            aria-selected={index === activeIndex}
                            onMouseMove={() => setActive(index)}
                            onMouseDown={(event) => event.preventDefault()}
                            onClick={() => accept(option)}
                            className={cn(
                                'flex cursor-pointer items-center justify-between gap-3 rounded-md px-3 py-2 text-sm',
                                index === activeIndex && 'bg-accent text-accent-foreground',
                            )}
                        >
                            <OptionContent option={option} />
                        </li>
                    ))}
                </ul>

                <div aria-live="polite" className="space-y-1 border-t px-3 py-2 text-xs text-muted-foreground">
                    {searching && !isFetching && results.length === 0 && completions.length === 0 && (
                        <p>{t('transactions.palette.empty')}</p>
                    )}
                    {summary && searching && total > 0 && (
                        <p>
                            {t('transactions.palette.totals', {
                                income: formatCurrency(summary.income, summary.currency),
                                expense: formatCurrency(summary.expense, summary.currency),
                            })}
                        </p>
                    )}
                    <p>{t('transactions.palette.keys')}</p>
                </div>
        </CommandDialog>
    )
}

function OptionContent({ option }: { option: Option }) {
    const { t } = useTranslation('pages')

    if (option.kind === 'completion') {
        return (
            <>
                <span>{option.label}</span>
                <kbd className="rounded border px-1.5 text-xs text-muted-foreground">Tab</kbd>
            </>
        )
    }
    if (option.kind === 'all') {
        return <span className="font-medium">{t('transactions.palette.showAll', { count: option.total })}</span>
    }

    const { transaction } = option
    const { sign, className } = transactionAmountAppearance(transaction.type, transaction.status)
    return (
        <>
            <span className="min-w-0">
                <span className="block truncate">{displayTransactionDescription(transaction)}</span>
                <span className="block truncate text-xs text-muted-foreground">
                    {transaction.date ? `${transaction.date} · ` : ''}
                    {transactionSubtitle(transaction)}
                </span>
            </span>
            <span className={cn('shrink-0 tabular-nums', className)}>
                {sign}
                {formatCurrency(transaction.amount, transaction.account.currency)}
            </span>
        </>
    )
}
