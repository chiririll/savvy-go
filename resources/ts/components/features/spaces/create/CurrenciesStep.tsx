import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Plus, X } from 'lucide-react'
import { ACCOUNT_TYPE_CONFIG } from '@/constants/accounts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { accountTypeLabel } from '@/lib/labels'
import { testId } from '@/lib/test-id'
import type { CurrencyCatalogItem } from '@/types'
import type { AccountType } from '@/types/accounts'
import { CurrencyCombobox } from './CurrencyCombobox'
import { canRemoveCurrency, newAccountKey, type DraftContent } from './draft'
import { StepSection } from './StepSection'

const ACCOUNT_TYPES: AccountType[] = ['bank', 'cash', 'crypto']

interface Props {
    content: DraftContent
    /** Undefined while the server is still loading it. */
    catalog: CurrencyCatalogItem[] | undefined
    catalogFailed?: boolean
    /** The catalog is the short built-in list: a currency may be missing, and any code can be typed. */
    catalogBuiltIn?: boolean
    onRetryCatalog?: () => void
    /** The catalog is being fetched right now (a retry included). */
    catalogFetching?: boolean
    onChange: (patch: Partial<DraftContent>) => void
}

/** The currencies listed first when nothing is typed: the big ones, and the one the interface language implies. */
function popularCurrencies(language: string): string[] {
    const own = language.startsWith('ru') ? ['RUB'] : []
    return [...new Set(['USD', 'EUR', 'GBP', ...own])]
}

/**
 * The notice about a failed or built-in currency list, with a Retry that shows what happened:
 * a spinner while the refetch runs, then whether the list was recovered.
 */
function CatalogNotice({
    failed,
    builtIn,
    fetching,
    onRetry,
}: {
    failed?: boolean
    builtIn?: boolean
    fetching?: boolean
    onRetry?: () => void
}) {
    const { t } = useTranslation('settings')
    // idle: Retry not used; waiting: pressed, the fetch not yet seen; running: seen; done: finished.
    const [phase, setPhase] = useState<'idle' | 'waiting' | 'running' | 'done'>('idle')
    useEffect(() => {
        if (phase === 'waiting' && fetching) setPhase('running')
        else if (phase === 'running' && !fetching) setPhase('done')
    }, [phase, fetching])

    const retry = (
        <button
            type="button"
            className="underline disabled:no-underline disabled:opacity-50"
            disabled={phase === 'waiting' || phase === 'running'}
            onClick={() => {
                setPhase('waiting')
                onRetry?.()
            }}
        >
            {t('actions.retry', { ns: 'common' })}
        </button>
    )

    if (phase === 'waiting' || phase === 'running') {
        return (
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground" role="status">
                <Loader2 className="size-3 animate-spin" aria-hidden data-testid="create-space-catalog-spinner" />
                {t('spaces.create.currencies.refreshing')} {retry}
            </p>
        )
    }
    if (builtIn) {
        return (
            <p className="text-xs text-muted-foreground" role="status">
                {phase === 'done' ? t('spaces.create.currencies.stillBuiltIn') : t('spaces.create.currencies.builtIn')}{' '}
                {retry}
            </p>
        )
    }
    if (failed) {
        return (
            <p className="text-xs text-destructive">
                {t('spaces.create.currencies.failed')} {retry}
            </p>
        )
    }
    if (phase === 'done') {
        return (
            <p className="text-xs text-muted-foreground" role="status">
                {t('spaces.create.currencies.recovered')}
            </p>
        )
    }
    return null
}

/** Base currency and the other currencies, from the server-wide catalog. */
function Currencies({
    content,
    catalog,
    catalogFailed,
    catalogBuiltIn,
    catalogFetching,
    onRetryCatalog,
    onChange,
}: Props) {
    const { t, i18n } = useTranslation('settings')
    const taken = useMemo(
        () => new Set([content.baseCurrency, ...content.currencies].filter((c): c is string => !!c)),
        [content],
    )
    const popular = useMemo(() => popularCurrencies(i18n.language), [i18n.language])
    const loading = catalog === undefined
    const list = catalog ?? []

    return (
        <StepSection title={t('spaces.create.currencies.title')} hint={t('spaces.create.currencies.hint')}>
            <div className="space-y-1.5">
                <Label htmlFor="create-space-base">{t('spaces.create.currencies.base')}</Label>
                <CurrencyCombobox
                    id="create-space-base"
                    className="w-full"
                    catalog={list}
                    value={content.baseCurrency}
                    popular={popular}
                    allowManual={catalogBuiltIn}
                    disabled={loading}
                    placeholder={t('spaces.create.currencies.basePlaceholder')}
                    data-testid="create-space-base"
                    onChange={(code) =>
                        onChange({
                            baseCurrency: code,
                            // A currency cannot be both the base and one of the others.
                            currencies: content.currencies.filter((c) => c !== code),
                        })
                    }
                />
                {loading && !catalogFailed && (
                    <p className="text-xs text-muted-foreground">{t('spaces.create.currencies.loading')}</p>
                )}
                <CatalogNotice
                    failed={catalogFailed}
                    builtIn={catalogBuiltIn}
                    fetching={catalogFetching}
                    onRetry={onRetryCatalog}
                />            </div>
            <div className="flex flex-wrap items-center gap-2">
                {content.currencies.map((code) => (
                    <span key={code} className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-sm">
                        <span className="font-mono">{code}</span>
                        <button
                            type="button"
                            aria-label={t('spaces.create.currencies.remove', { code })}
                            disabled={!canRemoveCurrency(content, code)}
                            className="text-muted-foreground hover:text-foreground disabled:opacity-30"
                            onClick={() => onChange({ currencies: content.currencies.filter((c) => c !== code) })}
                        >
                            <X className="size-3.5" />
                        </button>
                    </span>
                ))}
                {/* Full width on a phone, so the search list is not squeezed into a small box. */}
                <CurrencyCombobox
                    className="w-full sm:w-44"
                    catalog={list}
                    exclude={taken}
                    popular={popular}
                    allowManual={catalogBuiltIn}
                    disabled={loading || !content.baseCurrency}
                    placeholder={t('spaces.create.currencies.add')}
                    data-testid="create-space-add-currency"
                    onChange={(code) => onChange({ currencies: [...content.currencies, code] })}
                />
            </div>
        </StepSection>
    )
}

function Accounts({ content, catalog, onChange }: Pick<Props, 'content' | 'catalog' | 'onChange'>) {
    const { t } = useTranslation('settings')
    const { t: tc } = useTranslation()
    const codes = [content.baseCurrency, ...content.currencies].filter((c): c is string => !!c)
    const symbols = useMemo(() => new Map((catalog ?? []).map((c) => [c.code, c.symbol])), [catalog])
    // Nothing to add where the symbol is the code itself, or the catalog has no entry yet.
    const symbolOf = (code: string): string | null => {
        const symbol = symbols.get(code)?.trim()
        return symbol && symbol !== code ? symbol : null
    }
    const patch = (key: string, change: Partial<DraftContent['accounts'][number]>) =>
        onChange({ accounts: content.accounts.map((a) => (a.key === key ? { ...a, ...change } : a)) })
    const remove = (key: string) => onChange({ accounts: content.accounts.filter((a) => a.key !== key) })

    return (
        <StepSection title={t('spaces.create.accounts.title')} hint={t('spaces.create.accounts.hint')}>
            {content.accounts.map((account) => (
                // Phone: name on top, type and currency side by side, balance and remove last.
                <div
                    key={account.key}
                    className="grid grid-cols-2 gap-2 rounded-lg border p-2 sm:grid-cols-[2fr_1fr_1fr_1fr_auto]"
                >
                    <Input
                        className="col-span-2 sm:col-span-1"
                        {...testId('create-space-account-name')}
                        aria-label={t('spaces.create.accounts.name')}
                        placeholder={t('spaces.create.accounts.name')}
                        value={account.name}
                        maxLength={255}
                        onChange={(e) => patch(account.key, { name: e.target.value })}
                    />
                    <Select
                        value={account.type}
                        onValueChange={(type) => patch(account.key, { type: type as AccountType })}
                    >
                        <SelectTrigger className="w-full" aria-label={tc('fields.type')}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {ACCOUNT_TYPES.map((type) => {
                                const { icon: Icon, textColor } = ACCOUNT_TYPE_CONFIG[type]
                                return (
                                    <SelectItem key={type} value={type}>
                                        <Icon aria-hidden className={textColor} />
                                        {accountTypeLabel(tc, type)}
                                    </SelectItem>
                                )
                            })}
                        </SelectContent>
                    </Select>
                    <Select value={account.currency} onValueChange={(currency) => patch(account.key, { currency })}>
                        <SelectTrigger
                            className="w-full"
                            aria-label={t('spaces.create.accounts.currency')}
                            {...testId('create-space-account-currency')}
                        >
                            <SelectValue placeholder={t('spaces.create.accounts.currency')} />
                        </SelectTrigger>
                        <SelectContent>
                            {codes.map((code) => (
                                <SelectItem key={code} value={code}>
                                    <span className="font-mono">{code}</span>
                                    {symbolOf(code) && <span className="text-muted-foreground">{symbolOf(code)}</span>}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <div className="relative">
                        <Input
                            inputMode="decimal"
                            className={symbolOf(account.currency) ? 'pr-10' : undefined}
                            {...testId('create-space-account-balance')}
                            aria-label={t('spaces.create.accounts.balance')}
                            placeholder={t('spaces.create.accounts.balance')}
                            value={account.initialBalance}
                            onChange={(e) => patch(account.key, { initialBalance: e.target.value })}
                        />
                        {symbolOf(account.currency) && (
                            <span
                                aria-hidden
                                data-testid="create-space-account-balance-symbol"
                                className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-muted-foreground"
                            >
                                {symbolOf(account.currency)}
                            </span>
                        )}
                    </div>
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="justify-self-end"
                        aria-label={t('spaces.create.accounts.remove')}
                        onClick={() => remove(account.key)}
                    >
                        <X className="size-4" />
                    </Button>
                </div>
            ))}
            <Button
                type="button"
                variant="outline"
                size="sm"
                {...testId('create-space-add-account')}
                disabled={codes.length === 0}
                onClick={() =>
                    onChange({
                        accounts: [
                            ...content.accounts,
                            {
                                key: newAccountKey(),
                                name: '',
                                type: 'bank',
                                currency: content.baseCurrency ?? '',
                                initialBalance: '',
                            },
                        ],
                    })
                }
            >
                <Plus className="size-4" /> {t('spaces.create.accounts.add')}
            </Button>
        </StepSection>
    )
}

/** Step: the base currency, the other currencies, and the accounts that use them. */
export function CurrenciesStep(props: Props) {
    return (
        <div className="space-y-6">
            <Currencies {...props} />
            <Accounts content={props.content} catalog={props.catalog} onChange={props.onChange} />
        </div>
    )
}
