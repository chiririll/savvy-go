import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
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
    /** The user tried to go on without an account: the message about it turns into an error. */
    accountsAttempted?: boolean
    onChange: (patch: Partial<DraftContent>) => void
}

/** The currencies listed first when nothing is typed: the big ones, and the one the interface language implies. */
function popularCurrencies(language: string): string[] {
    const own = language.startsWith('ru') ? ['RUB'] : []
    return [...new Set(['USD', 'EUR', 'GBP', ...own])]
}

/** Base currency and the other currencies, from the server-wide catalog. */
function Currencies({ content, catalog, catalogFailed, catalogBuiltIn, onRetryCatalog, onChange }: Props) {
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
                {catalogFailed && (
                    <p className="text-xs text-destructive">
                        {t('spaces.create.currencies.failed')}{' '}
                        <button type="button" className="underline" onClick={onRetryCatalog}>
                            {t('actions.retry', { ns: 'common' })}
                        </button>
                    </p>
                )}
                {catalogBuiltIn && (
                    <p className="text-xs text-muted-foreground" role="status">
                        {t('spaces.create.currencies.builtIn')}{' '}
                        <button type="button" className="underline" onClick={onRetryCatalog}>
                            {t('actions.retry', { ns: 'common' })}
                        </button>
                    </p>
                )}
            </div>
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

function Accounts({
    content,
    accountsAttempted,
    onChange,
}: Pick<Props, 'content' | 'accountsAttempted' | 'onChange'>) {
    const { t } = useTranslation('settings')
    const { t: tc } = useTranslation()
    // Taking the last account away is an attempt too: the message was seen, and now it matters.
    const [removedLast, setRemovedLast] = useState(false)
    const codes = [content.baseCurrency, ...content.currencies].filter((c): c is string => !!c)
    const patch = (key: string, change: Partial<DraftContent['accounts'][number]>) =>
        onChange({ accounts: content.accounts.map((a) => (a.key === key ? { ...a, ...change } : a)) })
    const remove = (key: string) => {
        const accounts = content.accounts.filter((a) => a.key !== key)
        if (accounts.length === 0) setRemovedLast(true)
        onChange({ accounts })
    }

    return (
        <StepSection title={t('spaces.create.accounts.title')} hint={t('spaces.create.accounts.hint')}>
            {content.accounts.length === 0 &&
                (accountsAttempted || removedLast ? (
                    <p role="alert" className="text-xs text-destructive" data-testid="create-space-no-account">
                        {t('spaces.create.accounts.required')}
                    </p>
                ) : (
                    <p className="text-xs text-muted-foreground" data-testid="create-space-account-hint">
                        {t('spaces.create.accounts.required')}
                    </p>
                ))}
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
                            {ACCOUNT_TYPES.map((type) => (
                                <SelectItem key={type} value={type}>
                                    {accountTypeLabel(tc, type)}
                                </SelectItem>
                            ))}
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
                                    {code}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Input
                        inputMode="decimal"
                        {...testId('create-space-account-balance')}
                        aria-label={t('spaces.create.accounts.balance')}
                        placeholder={t('spaces.create.accounts.balance')}
                        value={account.initialBalance}
                        onChange={(e) => patch(account.key, { initialBalance: e.target.value })}
                    />
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
            <Accounts content={props.content} accountsAttempted={props.accountsAttempted} onChange={props.onChange} />
        </div>
    )
}
