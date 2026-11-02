import type { AccountType } from '@/types/accounts'
import type { CategoryType } from '@/types/categories'
import type { CreateSpaceInput, SpacePresets, SpaceRole } from '@/types/spaces'

export const DRAFT_STEPS = ['name', 'copy', 'currencies', 'categories', 'tags', 'confirm'] as const
export type DraftStep = (typeof DRAFT_STEPS)[number]

/**
 * A step name read from a stored draft. Drafts saved when currencies, categories
 * and tags were one `entities` step come back on the first of the three; an
 * unknown name gives null.
 */
export function migrateStep(step: string): DraftStep | null {
    if (step === 'entities') return 'currencies'
    return (DRAFT_STEPS as readonly string[]).includes(step) ? (step as DraftStep) : null
}

export const COPY_KINDS = ['currencies', 'accounts', 'categories', 'tags'] as const
export type CopyKind = (typeof COPY_KINDS)[number]
export type CopyKinds = Record<CopyKind, boolean>

export interface DraftAccount {
    /** Only a list key for the editor. */
    key: string
    name: string
    type: AccountType
    currency: string
    /** A decimal string; money never goes through a float. */
    initialBalance: string
}

export interface DraftCategory {
    name: string
    type: CategoryType
    icon: string
    color: string
    isDefault: boolean
}

/** What the entity step edits. */
export interface DraftContent {
    baseCurrency: string | null
    currencies: string[]
    accounts: DraftAccount[]
    categories: DraftCategory[]
    tags: string[]
}

/** Someone to invite right after the space is created. No address means a link to copy. */
export interface DraftInvitation {
    /** Only a list key. */
    key: string
    email: string | null
    /** Who a link is for; only kept when there is no address. */
    note: string | null
    role: SpaceRole
    /** Language of the email; unset in drafts saved before it was asked, and for a link. */
    locale?: string | null
}

export interface SpaceDraft extends DraftContent {
    step: DraftStep
    name: string
    /** '' means none. */
    emoji: string
    color: string
    /** The space to copy from; null starts from the defaults. */
    sourceId: number | null
    copy: CopyKinds
}

/** A space read through the existing endpoints, reduced to what can be copied. */
export interface CopySource {
    baseCurrency: string | null
    currencies: string[]
    accounts: { name: string; type: AccountType; currency: string }[]
    categories: DraftCategory[]
    tags: string[]
}

export const noCopy = (): CopyKinds => ({ currencies: false, accounts: false, categories: false, tags: false })

export const emptyDraft = (): SpaceDraft => ({
    step: 'name',
    name: '',
    emoji: '',
    color: '',
    sourceId: null,
    copy: noCopy(),
    baseCurrency: null,
    currencies: [],
    accounts: [],
    categories: [],
    tags: [],
})

let counter = 0
export const newAccountKey = () => `a${Date.now().toString(36)}${(counter++).toString(36)}`
const newInvitationKey = () => `i${Date.now().toString(36)}${(counter++).toString(36)}`

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/**
 * Adds someone to the invitation list. An address that is not one, or is
 * already listed (any case), leaves the list as it was; entries without an
 * address are links and can repeat.
 */
export function addInvitation(
    list: DraftInvitation[],
    input: { email: string | null; note?: string | null; role: SpaceRole; locale?: string | null },
): DraftInvitation[] {
    const email = input.email?.trim() || null
    if (email !== null && (!EMAIL.test(email) || list.some((i) => i.email?.toLowerCase() === email.toLowerCase())))
        return list
    const note = email === null ? input.note?.trim() || null : null
    return [...list, { key: newInvitationKey(), email, note, role: input.role, locale: email ? input.locale : null }]
}

export const removeInvitation = (list: DraftInvitation[], key: string): DraftInvitation[] =>
    list.filter((i) => i.key !== key)

/** The steps in order; the copy step only exists when there is another space to copy from. */
function stepsFor(otherSpaces: number): DraftStep[] {
    return DRAFT_STEPS.filter((step) => step !== 'copy' || otherSpaces > 0)
}

export function nextStep(step: DraftStep, otherSpaces: number): DraftStep {
    const order = stepsFor(otherSpaces)
    return order[Math.min(order.indexOf(step) + 1, order.length - 1)]
}

/** A draft saved on the copy step can come back when there is nothing to copy from. */
export function effectiveStep(step: DraftStep, otherSpaces: number): DraftStep {
    return step === 'copy' && otherSpaces === 0 ? 'name' : step
}

/** The space to copy from; a source that is gone (or none picked) means the defaults. */
export function effectiveSourceId(sourceId: number | null, spaceIds: number[]): number | null {
    return sourceId !== null && spaceIds.includes(sourceId) ? sourceId : null
}

export function previousStep(step: DraftStep, otherSpaces: number): DraftStep {
    const order = stepsFor(otherSpaces)
    return order[Math.max(order.indexOf(step) - 1, 0)]
}

/** The type defaults (#OTHER, #OTHER_INCOME) stay: every transaction needs one to fall back to. */
export const isProtectedCategory = (category: DraftCategory) => category.isDefault

export function removeCategory(categories: DraftCategory[], index: number): DraftCategory[] {
    const category = categories[index]
    if (category === undefined || isProtectedCategory(category)) return categories
    return categories.filter((_, i) => i !== index)
}

/** Makes sure each type has a default: the preset one is added when the list lacks it. */
function withDefaults(categories: DraftCategory[], presets: SpacePresets): DraftCategory[] {
    const out = [...categories]
    for (const type of ['expense', 'income'] as const) {
        if (out.some((c) => c.type === type && c.isDefault)) continue
        const preset = presets.categories.find((c) => c.type === type && c.isDefault)
        if (preset) out.push({ ...preset })
    }
    return out
}

/** Adds the currency of every account to the list, once each; the base is not repeated. */
export function withAccountCurrencies<T extends DraftContent>(content: T): T {
    const seen = new Set<string>(content.baseCurrency ? [content.baseCurrency] : [])
    const currencies: string[] = []
    for (const code of [...content.currencies, ...content.accounts.map((a) => a.currency)]) {
        if (code && !seen.has(code)) {
            seen.add(code)
            currencies.push(code)
        }
    }
    return { ...content, currencies }
}

/**
 * The starting content of the entity step: each kind comes from the source if
 * it was ticked, otherwise from the defaults. A copied account keeps no balance
 * and pulls its currency in, so the draft stays consistent.
 */
export function buildContent(kinds: CopyKinds, source: CopySource | null, presets: SpacePresets): DraftContent {
    const from = (kind: CopyKind) => (source !== null && kinds[kind] ? source : null)
    const currencies = from('currencies')
    const accounts = from('accounts')
    const categories = from('categories')
    const tags = from('tags')
    return withAccountCurrencies({
        baseCurrency: currencies?.baseCurrency ?? null,
        currencies: currencies ? currencies.currencies.filter((code) => code !== currencies.baseCurrency) : [],
        accounts: accounts
            ? accounts.accounts.map((a) => ({
                  key: newAccountKey(),
                  name: a.name,
                  type: a.type,
                  currency: a.currency,
                  initialBalance: '',
              }))
            : [],
        categories: withDefaults(
            (categories ? categories.categories : presets.categories).map((c) => ({ ...c })),
            presets,
        ),
        tags: [...(tags ? tags.tags : presets.tags)],
    })
}

/** A currency may go unless it is the base or an account still uses it. */
export function canRemoveCurrency(content: DraftContent, code: string): boolean {
    return code !== content.baseCurrency && !content.accounts.some((a) => a.currency === code)
}

const BALANCE = /^-?\d+([.,]\d+)?$/

/** Which parts of the draft stop it from being sent. */
export function draftProblems(
    draft: SpaceDraft,
): ('name' | 'baseCurrency' | 'noAccounts' | 'accounts' | 'balances')[] {
    const problems: ReturnType<typeof draftProblems> = []
    if (!draft.name.trim()) problems.push('name')
    if (!draft.baseCurrency) problems.push('baseCurrency')
    // The draft's accounts are the final ones, copied accounts included: a space with none cannot hold a transaction.
    if (draft.accounts.length === 0) problems.push('noAccounts')
    if (draft.accounts.some((a) => !a.name.trim() || !a.currency)) problems.push('accounts')
    if (draft.accounts.some((a) => a.initialBalance.trim() !== '' && !BALANCE.test(a.initialBalance.trim())))
        problems.push('balances')
    return problems
}

/** What keeps the user on a step: the currencies step needs a base currency and valid accounts; the others block nothing. */
export function stepProblems(step: DraftStep, draft: SpaceDraft): ReturnType<typeof draftProblems> {
    if (step !== 'currencies') return []
    return draftProblems(draft).filter((p) => p !== 'name')
}

/** The body of POST /spaces. */
export function toRequest(draft: SpaceDraft): CreateSpaceInput {
    const content = withAccountCurrencies(draft)
    const seen = new Set<string>()
    const tags: string[] = []
    for (const raw of content.tags) {
        const tag = raw.trim()
        if (tag && !seen.has(tag.toLowerCase())) {
            seen.add(tag.toLowerCase())
            tags.push(tag)
        }
    }
    return {
        name: draft.name.trim(),
        ...(draft.emoji ? { emoji: draft.emoji } : {}),
        ...(draft.color ? { color: draft.color } : {}),
        baseCurrency: content.baseCurrency ?? undefined,
        currencies: content.currencies,
        accounts: content.accounts.map((a) => {
            const balance = a.initialBalance.trim().replace(',', '.')
            return {
                name: a.name.trim(),
                type: a.type,
                currency: a.currency,
                ...(balance ? { initialBalance: balance } : {}),
            }
        }),
        categories: content.categories.map((c) => ({ ...c, name: c.name.trim() })).filter((c) => c.name),
        tags,
    }
}

/** Reduces what the endpoints return for a space to what can be copied. */
export function toCopySource(content: {
    currencies: { code: string; isBase: boolean }[]
    accounts: { name: string; type: AccountType; currency: { code: string } | null }[]
    categories: { name: string; type: CategoryType; icon: string | null; color: string | null; isDefault: boolean }[]
    tags: { name: string }[]
}): CopySource {
    return {
        baseCurrency: content.currencies.find((c) => c.isBase)?.code ?? null,
        currencies: content.currencies.map((c) => c.code),
        accounts: content.accounts
            .filter((a) => a.currency !== null && a.type !== 'debt')
            .map((a) => ({ name: a.name, type: a.type, currency: a.currency!.code })),
        categories: content.categories.map((c) => ({
            name: c.name,
            type: c.type,
            icon: c.icon ?? '',
            color: c.color ?? '',
            isDefault: c.isDefault,
        })),
        tags: content.tags.map((t) => t.name),
    }
}
