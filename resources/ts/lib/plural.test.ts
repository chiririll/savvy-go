import { readFileSync } from 'node:fs'
import path from 'node:path'
import i18next from 'i18next'
import { beforeAll, describe, expect, it } from 'vitest'

import { formatPlural } from './plural'

const locale = (lng: string, ns: string) =>
    JSON.parse(readFileSync(path.resolve(__dirname, '../locales', lng, `${ns}.json`), 'utf8'))

/** A plain i18next instance wired up like lib/i18n.ts, on the real locale files. */
const i18n = i18next.createInstance()

beforeAll(async () => {
    await i18n.init({
        lng: 'ru',
        fallbackLng: 'en',
        interpolation: { escapeValue: false },
        defaultNS: 'pages',
        ns: ['pages', 'settings'],
        resources: {
            en: { pages: locale('en', 'pages'), settings: locale('en', 'settings') },
            ru: { pages: locale('ru', 'pages'), settings: locale('ru', 'settings') },
        },
    })
    i18n.services.formatter?.add('plural', formatPlural)
})

describe('formatPlural', () => {
    const ru = { one: '# позиция', few: '# позиции', other: '# позиций' }

    it.each([
        [0, '0 позиций'],
        [1, '1 позиция'],
        [2, '2 позиции'],
        [4, '4 позиции'],
        [5, '5 позиций'],
        [11, '11 позиций'],
        [12, '12 позиций'],
        [21, '21 позиция'],
        [22, '22 позиции'],
        [25, '25 позиций'],
        [101, '101 позиция'],
        [1.5, '1,5 позиций'],
    ])('picks the Russian form for %s', (count, expected) => {
        expect(formatPlural(count, 'ru', ru)).toBe(expected)
    })

    it('picks the English form', () => {
        const en = { one: '# item', other: '# items' }
        expect(formatPlural(0, 'en', en)).toBe('0 items')
        expect(formatPlural(1, 'en', en)).toBe('1 item')
        expect(formatPlural(2, 'en', en)).toBe('2 items')
    })

    it('formats the number for the language', () => {
        expect(formatPlural(1234, 'en', { other: '# items' })).toBe('1,234 items')
        expect(formatPlural(1234, 'ru', { other: '# позиций' })).toBe('1 234 позиций')
    })

    it('falls back to other for a form that is not listed', () => {
        expect(formatPlural(5, 'ru', { one: '# а', other: '# б' })).toBe('5 б')
    })

    it('copes with a missing other and with a text without #', () => {
        expect(formatPlural(5, 'en', { one: '# a' })).toBe('')
        expect(formatPlural(1, 'en', { one: 'one thing', other: 'things' })).toBe('one thing')
    })

    it('reads a numeric string like a number', () => {
        expect(formatPlural('1', 'en', { one: '# item', other: '# items' })).toBe('1 item')
    })
})

describe('plurals in the locale files', () => {
    it('is wired through i18next', () => {
        // One real message end to end, so a change in how i18next splits the
        // format options shows up here and not in the UI.
        expect(i18n.t('settings:security.recovery.remaining', { count: 1 })).toBe(
            'Остался 1 код. Новая генерация сделает старые недействительными.',
        )
    })

    it.each([
        [1, 'Остался 1 код.'],
        [2, 'Осталось 2 кода.'],
        [5, 'Осталось 5 кодов.'],
        [11, 'Осталось 11 кодов.'],
        [21, 'Остался 21 код.'],
    ])('keeps the Russian verb in agreement with the count (%s)', (count, start) => {
        expect(i18n.t('settings:security.recovery.remaining', { count })).toMatch(new RegExp(`^${start}`))
    })

    it('keeps the rest of the sentence once, around the plural', () => {
        expect(i18n.t('categories.reassignDescription', { count: 3, name: 'Еда' })).toBe(
            'В категории «Еда» есть 3 транзакции. Выберите категорию, куда перенести транзакции перед удалением.',
        )
        expect(i18n.t('categories.reassignDescription', { count: 12, name: 'Еда' })).toContain('есть 12 транзакций.')
    })

    it('renders English too', async () => {
        await i18n.changeLanguage('en')
        expect(i18n.t('settings:security.recovery.remaining', { count: 1 })).toBe('1 code remaining. Regenerating invalidates the old set.')
        expect(i18n.t('settings:security.recovery.remaining', { count: 4 })).toBe('4 codes remaining. Regenerating invalidates the old set.')
        expect(i18n.t('dashboard.pendingCount', { count: 1 })).toBe('1 transaction')
        expect(i18n.t('dashboard.pendingCount', { count: 0 })).toBe('0 transactions')
        await i18n.changeLanguage('ru')
    })
})
