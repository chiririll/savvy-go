/**
 * i18next format for plurals written inline, so the rest of a sentence is
 * written once instead of once per plural form:
 *
 *   "{{count, plural(one: # code; few: # codes; other: # codes)}} left"
 *
 * The form is picked by the language's own plural rules; `#` stands for the
 * count, formatted for the language. `other` is required and also covers any
 * form not listed (Russian `many`, for one). Form texts cannot contain `:`,
 * `;` or `)`, which end the options.
 */
export function formatPlural(value: unknown, lng: string | undefined, options: Record<string, unknown>): string {
    const count = Number(value)
    const form = new Intl.PluralRules(lng).select(count)
    const text = String(options[form] ?? options.other ?? '')
    return text.replace('#', new Intl.NumberFormat(lng).format(count))
}
