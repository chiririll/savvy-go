import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { accountTypeLabel, categoryTypeLabel } from '@/lib/labels'
import { testId } from '@/lib/test-id'
import { localizeDefaultName, toStoredDefaultName } from '@/lib/localized-name'
import type { CurrencyCatalogItem } from '@/types'
import type { AccountType } from '@/types/accounts'
import { CATEGORY_TYPES, type CategoryType } from '@/types/categories'
import {
    canRemoveCurrency,
    isProtectedCategory,
    newAccountKey,
    removeCategory,
    type DraftContent,
} from './draft'

const ACCOUNT_TYPES: AccountType[] = ['bank', 'cash', 'crypto']

interface Props {
    content: DraftContent
    /** Undefined while the server is still loading it. */
    catalog: CurrencyCatalogItem[] | undefined
    catalogFailed?: boolean
    onRetryCatalog?: () => void
    onChange: (patch: Partial<DraftContent>) => void
}

function Section({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
    return (
        <section className="space-y-2">
            <div>
                <h3 className="text-sm font-semibold">{title}</h3>
                {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
            </div>
            {children}
        </section>
    )
}

function CurrencyOption({ item }: { item: CurrencyCatalogItem }) {
    return (
        <>
            <span className="font-mono">{item.code}</span> <span className="text-muted-foreground">{item.name}</span>
        </>
    )
}

/** Base currency and the other currencies, from the server-wide catalog. */
function Currencies({ content, catalog, catalogFailed, onRetryCatalog, onChange }: Props) {
    const { t } = useTranslation('settings')
    const taken = new Set([content.baseCurrency, ...content.currencies])
    const loading = catalog === undefined
    // While loading, the chosen base (e.g. from a draft) still has an item to show.
    const options: CurrencyCatalogItem[] =
        catalog ?? (content.baseCurrency ? [{ code: content.baseCurrency, name: '', symbol: '', decimals: 2, rate: null }] : [])
    const free = (catalog ?? []).filter((c) => !taken.has(c.code))

    return (
        <Section title={t('spaces.create.currencies.title')} hint={t('spaces.create.currencies.hint')}>
            <div className="space-y-1.5">
                <Label>{t('spaces.create.currencies.base')}</Label>
                <Select
                    value={content.baseCurrency ?? ''}
                    onValueChange={(code) =>
                        onChange({
                            baseCurrency: code,
                            // A currency cannot be both the base and one of the others.
                            currencies: content.currencies.filter((c) => c !== code),
                        })
                    }
                >
                    <SelectTrigger className="w-full" disabled={loading} data-testid="create-space-base">
                        <SelectValue placeholder={t('spaces.create.currencies.basePlaceholder')} />
                    </SelectTrigger>
                    <SelectContent>
                        {options.map((c) => (
                            <SelectItem key={c.code} value={c.code}>
                                <CurrencyOption item={c} />
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                {loading && !catalogFailed && <p className="text-xs text-muted-foreground">{t('spaces.create.currencies.loading')}</p>}
                {catalogFailed && (
                    <p className="text-xs text-destructive">
                        {t('spaces.create.currencies.failed')}{' '}
                        <button type="button" className="underline" onClick={onRetryCatalog}>{t('actions.retry', { ns: 'common' })}</button>
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
                <Select value="" onValueChange={(code) => onChange({ currencies: [...content.currencies, code] })}>
                    <SelectTrigger className="w-44" disabled={loading || !content.baseCurrency} {...testId('create-space-add-currency')}>
                        <SelectValue placeholder={t('spaces.create.currencies.add')} />
                    </SelectTrigger>
                    <SelectContent>
                        {free.map((c) => (
                            <SelectItem key={c.code} value={c.code}>
                                <CurrencyOption item={c} />
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>
        </Section>
    )
}

function Accounts({ content, onChange }: Pick<Props, 'content' | 'onChange'>) {
    const { t } = useTranslation('settings')
    const { t: tc } = useTranslation()
    const codes = [content.baseCurrency, ...content.currencies].filter((c): c is string => !!c)
    const patch = (key: string, change: Partial<DraftContent['accounts'][number]>) =>
        onChange({ accounts: content.accounts.map((a) => (a.key === key ? { ...a, ...change } : a)) })

    return (
        <Section title={t('spaces.create.accounts.title')} hint={t('spaces.create.accounts.hint')}>
            {content.accounts.map((account) => (
                // Phone: name on top, type and currency side by side, balance and remove last.
                <div key={account.key} className="grid grid-cols-2 gap-2 rounded-lg border p-2 sm:grid-cols-[2fr_1fr_1fr_1fr_auto]">
                    <Input
                        className="col-span-2 sm:col-span-1"
                        {...testId('create-space-account-name')}
                        aria-label={t('spaces.create.accounts.name')}
                        placeholder={t('spaces.create.accounts.name')}
                        value={account.name}
                        maxLength={255}
                        onChange={(e) => patch(account.key, { name: e.target.value })}
                    />
                    <Select value={account.type} onValueChange={(type) => patch(account.key, { type: type as AccountType })}>
                        <SelectTrigger className="w-full">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {ACCOUNT_TYPES.map((type) => (
                                <SelectItem key={type} value={type}>{accountTypeLabel(tc, type)}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Select value={account.currency} onValueChange={(currency) => patch(account.key, { currency })}>
                        <SelectTrigger className="w-full" {...testId('create-space-account-currency')}>
                            <SelectValue placeholder={t('spaces.create.accounts.currency')} />
                        </SelectTrigger>
                        <SelectContent>
                            {codes.map((code) => (
                                <SelectItem key={code} value={code}>{code}</SelectItem>
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
                        onClick={() => onChange({ accounts: content.accounts.filter((a) => a.key !== account.key) })}
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
                            { key: newAccountKey(), name: '', type: 'bank', currency: content.baseCurrency ?? '', initialBalance: '' },
                        ],
                    })
                }
            >
                <Plus className="size-4" /> {t('spaces.create.accounts.add')}
            </Button>
        </Section>
    )
}

function Categories({ content, onChange }: Pick<Props, 'content' | 'onChange'>) {
    const { t } = useTranslation('settings')
    const { t: tc } = useTranslation()
    const [name, setName] = useState('')
    const [type, setType] = useState<CategoryType>('expense')
    const rename = (index: number, input: string) =>
        onChange({
            categories: content.categories.map((c, i) => (i === index ? { ...c, name: toStoredDefaultName(input, c.name) } : c)),
        })

    return (
        <Section title={t('spaces.create.categories.title')} hint={t('spaces.create.categories.hint')}>
            <ul className="space-y-1.5">
                {content.categories.map((category, index) => (
                    <li key={index} className="flex items-center gap-2">
                        <span aria-hidden className="w-6 text-center">{category.icon}</span>
                        <Input
                            aria-label={t('spaces.create.categories.name')}
                            value={localizeDefaultName(category.name)}
                            maxLength={255}
                            onChange={(e) => rename(index, e.target.value)}
                        />
                        <span className="w-20 shrink-0 text-xs text-muted-foreground">{categoryTypeLabel(tc, category.type)}</span>
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            disabled={isProtectedCategory(category)}
                            title={isProtectedCategory(category) ? t('spaces.create.categories.protected') : undefined}
                            aria-label={t('spaces.create.categories.remove', { name: localizeDefaultName(category.name) })}
                            onClick={() => onChange({ categories: removeCategory(content.categories, index) })}
                        >
                            <X className="size-4" />
                        </Button>
                    </li>
                ))}
            </ul>
            <form
                className="flex gap-2"
                onSubmit={(e) => {
                    e.preventDefault()
                    if (!name.trim()) return
                    onChange({
                        categories: [...content.categories, { name: name.trim(), type, icon: '🏷️', color: '#6b7280', isDefault: false }],
                    })
                    setName('')
                }}
            >
                <Input
                    aria-label={t('spaces.create.categories.new')}
                    placeholder={t('spaces.create.categories.new')}
                    value={name}
                    maxLength={255}
                    onChange={(e) => setName(e.target.value)}
                />
                <Select value={type} onValueChange={(v) => setType(v as CategoryType)}>
                    <SelectTrigger className="w-32">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {CATEGORY_TYPES.map((v) => (
                            <SelectItem key={v} value={v}>{categoryTypeLabel(tc, v)}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <Button type="submit" variant="outline" size="icon" aria-label={t('spaces.create.categories.add')}>
                    <Plus className="size-4" />
                </Button>
            </form>
        </Section>
    )
}

function Tags({ content, onChange }: Pick<Props, 'content' | 'onChange'>) {
    const { t } = useTranslation('settings')
    const [name, setName] = useState('')

    return (
        <Section title={t('spaces.create.tags.title')} hint={t('spaces.create.tags.hint')}>
            <div className="flex flex-wrap gap-2">
                {content.tags.map((tag) => (
                    <span key={tag} className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-sm">
                        {tag}
                        <button
                            type="button"
                            aria-label={t('spaces.create.tags.remove', { name: tag })}
                            className="text-muted-foreground hover:text-foreground"
                            onClick={() => onChange({ tags: content.tags.filter((x) => x !== tag) })}
                        >
                            <X className="size-3.5" />
                        </button>
                    </span>
                ))}
            </div>
            <form
                className="flex gap-2"
                onSubmit={(e) => {
                    e.preventDefault()
                    const tag = name.trim()
                    if (tag && !content.tags.some((x) => x.toLowerCase() === tag.toLowerCase())) onChange({ tags: [...content.tags, tag] })
                    setName('')
                }}
            >
                <Input
                    aria-label={t('spaces.create.tags.new')}
                    placeholder={t('spaces.create.tags.new')}
                    value={name}
                    maxLength={50}
                    onChange={(e) => setName(e.target.value)}
                />
                <Button type="submit" variant="outline" size="icon" aria-label={t('spaces.create.tags.add')}>
                    <Plus className="size-4" />
                </Button>
            </form>
        </Section>
    )
}

/** Step 4: every base entity, all editable. */
export function EntitiesStep(props: Props) {
    return (
        <div className="space-y-6">
            <Currencies {...props} />
            <Accounts content={props.content} onChange={props.onChange} />
            <Categories content={props.content} onChange={props.onChange} />
            <Tags content={props.content} onChange={props.onChange} />
        </div>
    )
}
