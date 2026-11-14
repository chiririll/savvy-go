// @vitest-environment jsdom
import { expect, it } from 'vitest'

import i18n, { i18nReady } from './i18n'

// Guards the formatter registration in i18n.ts itself, which plural.test.ts does not use.
it('renders inline plurals in the app instance', async () => {
    await i18nReady
    expect(i18n.t('pages:dashboard.pendingCount', { count: 1 })).toBe('1 transaction')
    expect(i18n.t('pages:dashboard.pendingCount', { count: 5 })).toBe('5 transactions')
    await i18n.changeLanguage('ru')
    expect(i18n.t('pages:dashboard.pendingCount', { count: 2 })).toBe('2 операции')
    expect(i18n.t('pages:dashboard.pendingCount', { count: 5 })).toBe('5 операций')
})
