import i18n from '@/lib/i18n'

const DEFAULT_NAME_RE = /^#[A-Z][A-Z0-9_]*$/

export function defaultNameKey(name: string | null | undefined): string | null {
    const trimmed = name?.trim() ?? ''
    return DEFAULT_NAME_RE.test(trimmed) ? trimmed.slice(1) : null
}

export function localizeDefaultName(
    name: string | null | undefined,
    group: 'categories' = 'categories',
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

export function toStoredDefaultName(
    input: string,
    original?: string | null,
    group: 'categories' = 'categories',
): string {
    const trimmed = input.trim()

    if (original && defaultNameKey(original) && trimmed === localizeDefaultName(original, group)) {
        return original.trim()
    }

    return trimmed
}
