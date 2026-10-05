import i18n from '@/lib/i18n'
import type { ApiError } from '@/types'

const I18N_NAMESPACES = ['common', 'forms', 'pages', 'auth', 'settings', 'nav'] as const

function looksLikeI18nKey(message: string): boolean {
    return /^[a-zA-Z][\w.-]*$/.test(message)
}

function toCamelCase(value: string): string {
    return value.replace(/_([a-z])/g, (_, char: string) => char.toUpperCase())
}

function translateIfExists(key: string, ns: string): string | undefined {
    if (!i18n.exists(key, { ns })) {
        return undefined
    }

    // i18n-dynamic: the key is an error code sent by the server
    const translated = i18n.t(key, { ns })
    return translated && translated !== key ? translated : undefined
}

function resolveI18nKey(message: string): string {
    const parts = message.split('.')

    if (parts.length >= 2) {
        const maybeNs = parts[0]
        const rest = parts.slice(1).join('.')
        if ((I18N_NAMESPACES as readonly string[]).includes(maybeNs)) {
            const nested = translateIfExists(rest, maybeNs)
            if (nested) {
                return nested
            }
        }
    }

    for (const ns of I18N_NAMESPACES) {
        const direct = translateIfExists(message, ns)
        if (direct) {
            return direct
        }
    }

    const leaf = parts[parts.length - 1]
    const candidates = [leaf, toCamelCase(leaf)].filter((value, index, list) => list.indexOf(value) === index)

    for (const candidate of candidates) {
        const fromValidation = translateIfExists(`validation.${candidate}`, 'common')
        if (fromValidation) {
            return fromValidation
        }

        const fromForms = translateIfExists(`transactions.${candidate}`, 'forms')
            ?? translateIfExists(`debts.${candidate}`, 'forms')
        if (fromForms) {
            return fromForms
        }
    }

    return message
}

export function getApiErrorMessage(error: unknown, fallback: string): string {
    if (!error || typeof error !== 'object') {
        return fallback
    }

    const apiError = error as ApiError
    const firstDetail = apiError.details
        ? Object.values(apiError.details).flat().find((item) => typeof item === 'string' && item.length > 0)
        : undefined

    const raw = firstDetail || apiError.message
    if (!raw) {
        return fallback
    }

    return resolveUserFacingText(raw)
}

export function resolveUserFacingText(message: string): string {
    return looksLikeI18nKey(message) ? resolveI18nKey(message) : message
}
