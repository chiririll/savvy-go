import i18n from '@/lib/i18n'

const DEFAULT_NAME_RE = /^#[A-Z][A-Z0-9_]*$/

export type DefaultNameGroup = 'categories' | 'tags'

export function defaultNameKey(name: string | null | undefined): string | null {
    const trimmed = name?.trim() ?? ''
    return DEFAULT_NAME_RE.test(trimmed) ? trimmed.slice(1) : null
}

export function localizeDefaultName(
    name: string | null | undefined,
    group: DefaultNameGroup = 'categories',
): string {
    if (!name) {
        return ''
    }

    const key = defaultNameKey(name)
    if (!key) {
        return name
    }

    const i18nKey = `${group}.${key}`
    if (i18n.exists(i18nKey, { ns: 'defaults' })) {
        // i18n-dynamic: the key is a name stored in the database (e.g. #SALARY)
        return i18n.t(i18nKey, { ns: 'defaults' })
    }

    return name
}

/** The one-line hint of a preset category (#UTILITIES), or null when it has none. */
export function defaultNameHint(name: string | null | undefined): string | null {
    const key = defaultNameKey(name)
    const i18nKey = `categoryHints.${key}`
    // i18n-dynamic: the key is a name stored in the database (e.g. #UTILITIES)
    return key && i18n.exists(i18nKey, { ns: 'defaults' }) ? i18n.t(i18nKey, { ns: 'defaults' }) : null
}

export function toStoredDefaultName(
    input: string,
    original?: string | null,
    group: DefaultNameGroup = 'categories',
): string {
    const trimmed = input.trim()

    if (original && defaultNameKey(original) && trimmed === localizeDefaultName(original, group)) {
        return original.trim()
    }

    return trimmed
}
