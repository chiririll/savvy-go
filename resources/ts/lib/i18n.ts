import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import type { BackendModule } from 'i18next'

import { formatPlural } from '@/lib/plural'

import enCommon from '@/locales/en/common.json'
import enNav from '@/locales/en/nav.json'
import enAuth from '@/locales/en/auth.json'
import enSettings from '@/locales/en/settings.json'
import enPages from '@/locales/en/pages.json'
import enForms from '@/locales/en/forms.json'
import enDefaults from '@/locales/en/defaults.json'

export const SUPPORTED_LOCALES = ['en', 'ru'] as const
export type AppLocale = (typeof SUPPORTED_LOCALES)[number]

export const LOCALE_LABELS: Record<AppLocale, string> = {
    en: 'English',
    ru: 'Русский',
}

export const LOCALE_STORAGE_KEY = 'savvy.locale'

export function isAppLocale(value: string): value is AppLocale {
    return (SUPPORTED_LOCALES as readonly string[]).includes(value)
}

const I18N_NAMESPACES = ['common', 'nav', 'auth', 'settings', 'pages', 'forms', 'defaults'] as const

/** BCP 47 tag for Intl formatters. */
export function intlLocale(locale: string = i18n.resolvedLanguage ?? i18n.language): string {
    return locale.startsWith('ru') ? 'ru-RU' : 'en-US'
}

function resolveDottedNamespaceKey(key: string): string {
    const parts = key.split('.')
    if (parts.length < 2) {
        return key
    }

    const ns = parts[0]
    const rest = parts.slice(1).join('.')
    if (!(I18N_NAMESPACES as readonly string[]).includes(ns)) {
        return key
    }

    if (i18n.exists(rest, { ns })) {
        // i18n-dynamic: this is the missing-key handler itself
        return i18n.t(rest, { ns })
    }

    return key
}

// English is bundled (fallback language); other locales are loaded on demand.
const localeLoaders = import.meta.glob<{ default: Record<string, unknown> }>('@/locales/*/*.json')

const lazyLocaleBackend: BackendModule = {
    type: 'backend',
    init() {},
    read(language, namespace, callback) {
        const loader = localeLoaders[`/resources/ts/locales/${language}/${namespace}.json`]
        if (!loader) {
            callback(null, {})
            return
        }
        loader().then((mod) => callback(null, mod.default), (err) => callback(err, false))
    },
}

export const i18nReady = i18n
    .use(lazyLocaleBackend)
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
        resources: {
            en: { common: enCommon, nav: enNav, auth: enAuth, settings: enSettings, pages: enPages, forms: enForms, defaults: enDefaults },
        },
        partialBundledLanguages: true,
        fallbackLng: 'en',
        supportedLngs: [...SUPPORTED_LOCALES],
        nonExplicitSupportedLngs: true,
        defaultNS: 'common',
        ns: [...I18N_NAMESPACES],
        interpolation: { escapeValue: false },
        parseMissingKeyHandler: resolveDottedNamespaceKey,
        detection: {
            order: ['localStorage', 'navigator'],
            caches: ['localStorage'],
            lookupLocalStorage: LOCALE_STORAGE_KEY,
        },
    })

function applyDocumentLang(lng: string) {
    if (typeof document === 'undefined') {
        return
    }

    document.documentElement.lang = lng.startsWith('ru') ? 'ru' : 'en'
}

// Created by init() above, before it resolves, so it is there for the first t().
i18n.services.formatter?.add('plural', formatPlural)

applyDocumentLang(i18n.resolvedLanguage ?? i18n.language)
i18n.on('languageChanged', applyDocumentLang)

export default i18n
