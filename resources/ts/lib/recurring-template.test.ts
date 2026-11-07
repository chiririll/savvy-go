import { describe, expect, it } from 'vitest'

import { accounts, categories, recurring, recurringTemplate } from '@/test/fixtures'
import type { RecurringFormValues } from '@/schemas/recurring'
import {
    buildRecurringInput,
    canEditRecurringInForm,
    isApproximateOccurrence,
    isApproximateRule,
    recurringToFormValues,
    templateAsTransaction,
} from './recurring-template'

const [cash, euroCard, yenWallet] = accounts
const sameCurrencyAccount = { ...euroCard, id: 9, name: 'Euro savings' }
const all = [...accounts, sameCurrencyAccount]

const transfer = (overrides = {}) =>
    recurring({
        template: recurringTemplate({
            type: 'transfer',
            description: 'To the card',
            payments: [
                { account_id: cash.id, amount: -100 },
                { account_id: euroCard.id, amount: 80 },
            ],
            lines: [],
        }),
        ...overrides,
    })

describe('templateAsTransaction', () => {
    it('shows a template as the payments and lines a row draws, with accounts and categories', () => {
        const tx = templateAsTransaction(recurring(), accounts, categories)!

        expect(tx).toMatchObject({
            type: 'expense',
            description: 'Rent',
            status: 'pending',
            recurringTransactionId: 1,
            payments: [{ accountId: cash.id, account: cash, amount: -1100 }],
            lines: [{ categoryId: 1, category: categories[0], currencyId: cash.currency!.id, amount: 1100 }],
        })
    })

    it('reads amounts that another client saved as strings', () => {
        const rule = recurring({
            template: recurringTemplate({
                payments: [{ account_id: cash.id, amount: '-1100.00' as unknown as number }],
                lines: [{ category_id: 1, currency_id: 1, amount: '1100.00' as unknown as number }],
            }),
        })

        expect(templateAsTransaction(rule, accounts, categories)).toMatchObject({
            payments: [{ amount: -1100 }],
            lines: [{ amount: 1100 }],
        })
    })

    it('gives a line without a currency the one all the payment accounts share', () => {
        const rule = recurring({ template: recurringTemplate({ lines: [{ category_id: 1, amount: 1100 } as never] }) })

        expect(templateAsTransaction(rule, accounts, categories)!.lines[0].currencyId).toBe(cash.currency!.id)
    })

    it('works out an amount left open from the rates: 100 USD is 80 EUR', () => {
        const rule = transfer({
            template: recurringTemplate({
                type: 'transfer',
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: null, estimated: true },
                ],
                lines: [],
            }),
        })

        expect(templateAsTransaction(rule, accounts, categories)!.payments[1].amount).toBe(80)
    })

    it('leaves the fee out of what the open side receives', () => {
        const rule = transfer({
            template: recurringTemplate({
                type: 'transfer',
                payments: [
                    { account_id: cash.id, amount: -102 },
                    { account_id: euroCard.id, amount: null },
                ],
                lines: [{ category_id: 3, currency_id: cash.currency!.id, amount: 2 }],
            }),
        })

        expect(templateAsTransaction(rule, accounts, categories)!.payments[1].amount).toBe(80)
    })

    it('is null when an account is gone', () => {
        const rule = recurring({ template: recurringTemplate({ payments: [{ account_id: 99, amount: -5 }] }) })

        expect(templateAsTransaction(rule, accounts, categories)).toBeNull()
    })
})

describe('isApproximateRule', () => {
    it('is false for a rule with a figure', () => {
        expect(isApproximateRule(recurring().template)).toBe(false)
    })

    it('is true when a payment is marked as an estimate', () => {
        const template = recurringTemplate({ payments: [{ account_id: 1, amount: -1100, estimated: true }] })

        expect(isApproximateRule(template)).toBe(true)
    })

    it('is true when a payment has no amount: the rates of the day decide it', () => {
        const template = recurringTemplate({
            type: 'transfer',
            payments: [
                { account_id: 1, amount: -100 },
                { account_id: 2, amount: null },
            ],
        })

        expect(isApproximateRule(template)).toBe(true)
    })
})

describe('isApproximateOccurrence', () => {
    const rules = new Set([7])

    it('is true for a pending occurrence of a rule whose amount is a guess', () => {
        expect(isApproximateOccurrence({ status: 'pending', recurringTransactionId: 7 }, rules)).toBe(true)
    })

    it('is false once it is confirmed: the real amount is known', () => {
        expect(isApproximateOccurrence({ status: 'confirmed', recurringTransactionId: 7 }, rules)).toBe(false)
    })

    it('is false for a rule with a figure, and for a transaction no rule made', () => {
        expect(isApproximateOccurrence({ status: 'pending', recurringTransactionId: 8 }, rules)).toBe(false)
        expect(isApproximateOccurrence({ status: 'pending', recurringTransactionId: null }, rules)).toBe(false)
    })
})

describe('canEditRecurringInForm', () => {
    it('opens a plain expense', () => {
        expect(canEditRecurringInForm(recurring(), accounts)).toBe(true)
    })

    it('opens a transfer whose amount received is open', () => {
        const rule = transfer({
            template: recurringTemplate({
                type: 'transfer',
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: null, estimated: true },
                ],
                lines: [],
            }),
        })

        expect(canEditRecurringInForm(rule, accounts)).toBe(true)
    })

    it('does not open an expense with an amount left open: the form asks for each one', () => {
        const rule = recurring({
            template: recurringTemplate({ payments: [{ account_id: cash.id, amount: null }] }),
        })

        expect(canEditRecurringInForm(rule, accounts)).toBe(false)
    })

    it('does not open a rule whose account is gone', () => {
        const rule = recurring({ template: recurringTemplate({ payments: [{ account_id: 99, amount: -5 }] }) })

        expect(canEditRecurringInForm(rule, accounts)).toBe(false)
    })

    it('does not open a transfer with more than two accounts', () => {
        const rule = transfer({
            template: recurringTemplate({
                type: 'transfer',
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: 40 },
                    { account_id: yenWallet.id, amount: 8000 },
                ],
                lines: [],
            }),
        })

        expect(canEditRecurringInForm(rule, accounts)).toBe(false)
    })
})

describe('recurringToFormValues', () => {
    it('shows an expense in one line as the simple mode with its category and the schedule', () => {
        const values = recurringToFormValues(recurring(), accounts)!

        expect(values).toMatchObject({
            type: 'expense',
            description: 'Rent',
            date: null,
            tag_ids: [],
            payments: [{ account_id: cash.id, amount: 1100 }],
            change: [],
            lines: [],
            default_category_id: 1,
            estimated: false,
            is_active: true,
            frequency: 'monthly',
            interval: 1,
            day_of_month: 1,
            start_date: '2026-01-01',
            end_date: null,
        })
    })

    it('shows several lines as lines', () => {
        const rule = recurring({
            template: recurringTemplate({
                payments: [{ account_id: cash.id, amount: -100 }],
                lines: [
                    { category_id: 1, currency_id: 1, amount: 60, note: '- milk' },
                    { category_id: 3, currency_id: 1, amount: 40 },
                ],
            }),
        })

        expect(recurringToFormValues(rule, accounts)).toMatchObject({
            default_category_id: null,
            lines: [
                { category_id: 1, amount: 60, note: '- milk' },
                { category_id: 3, amount: 40 },
            ],
        })
    })

    it('shows change apart from the payments', () => {
        const rule = recurring({
            template: recurringTemplate({
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: 4 },
                ],
                lines: [{ category_id: 1, currency_id: 1, amount: 95 }],
            }),
        })

        expect(recurringToFormValues(rule, accounts)).toMatchObject({
            payments: [{ account_id: cash.id, amount: 100 }],
            change: [{ account_id: euroCard.id, amount: 4 }],
        })
    })

    it('keeps a transfer open on the side that has no amount', () => {
        const rule = transfer({
            template: recurringTemplate({
                type: 'transfer',
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: null, estimated: true },
                ],
                lines: [],
            }),
        })

        expect(recurringToFormValues(rule, accounts)).toMatchObject({
            type: 'transfer',
            payments: [{ account_id: cash.id, amount: 100 }],
            to_account_id: euroCard.id,
            to_amount: null,
            exchange_rate: null,
            estimated: true,
        })
    })

    it('shows an estimate flag on any payment as the flag of the form', () => {
        const rule = recurring({
            template: recurringTemplate({ payments: [{ account_id: cash.id, amount: -1100, estimated: true }] }),
        })

        expect(recurringToFormValues(rule, accounts)!.estimated).toBe(true)
    })

    it('takes the weekday and day of the month from the start when it has none', () => {
        const rule = recurring({
            schedule: { frequency: 'weekly', interval: 2, start: '2026-12-09', end: '2027-06-30' },
        })

        expect(recurringToFormValues(rule, accounts)).toMatchObject({
            frequency: 'weekly',
            interval: 2,
            day_of_week: 3, // a Wednesday
            day_of_month: 9,
            end_date: '2027-06-30',
        })
    })

    it('keeps a paused rule paused and the tags', () => {
        const rule = recurring({ isActive: false, template: recurringTemplate({ tag_ids: [2, 3] }) })

        expect(recurringToFormValues(rule, accounts)).toMatchObject({ is_active: false, tag_ids: [2, 3] })
    })

    it('is null for a rule the form cannot show', () => {
        const rule = recurring({ template: recurringTemplate({ payments: [{ account_id: 99, amount: -5 }] }) })

        expect(recurringToFormValues(rule, accounts)).toBeNull()
    })
})

const values = (overrides: Partial<RecurringFormValues> = {}): RecurringFormValues => ({
    type: 'expense',
    description: 'Rent',
    date: null,
    tag_ids: [],
    payments: [{ account_id: cash.id, amount: 1100 }],
    change: [],
    lines: [],
    default_category_id: 1,
    default_note: null,
    estimated: false,
    frequency: 'monthly',
    interval: 1,
    day_of_week: 1,
    day_of_month: 5,
    start_date: '2026-12-05',
    end_date: null,
    is_active: true,
    ...overrides,
})

describe('buildRecurringInput', () => {
    it('makes a template of signed payments and lines, with no date, and the schedule', () => {
        expect(buildRecurringInput(values(), all)).toEqual({
            ok: true,
            input: {
                template: {
                    type: 'expense',
                    description: 'Rent',
                    payments: [{ account_id: cash.id, amount: -1100 }],
                    lines: [{ category_id: 1, currency_id: cash.currency!.id, amount: 1100 }],
                    tag_ids: [],
                },
                schedule: { frequency: 'monthly', interval: 1, day_of_month: 5, start: '2026-12-05', end: null },
                is_active: true,
            },
        })
    })

    it('keeps only the day the frequency uses: the API refuses the other', () => {
        const weekly = buildRecurringInput(values({ frequency: 'weekly' }), all)
        const daily = buildRecurringInput(values({ frequency: 'daily' }), all)

        expect(weekly).toMatchObject({ ok: true, input: { schedule: { day_of_week: 1 } } })
        expect(weekly.ok && 'day_of_month' in weekly.input.schedule).toBe(false)
        expect(daily.ok && ('day_of_week' in daily.input.schedule || 'day_of_month' in daily.input.schedule)).toBe(false)
    })

    it('sends the end date and a paused rule', () => {
        const built = buildRecurringInput(values({ end_date: '2027-12-05', is_active: false }), all)

        expect(built).toMatchObject({ ok: true, input: { schedule: { end: '2027-12-05' }, is_active: false } })
    })

    it('marks every payment as an estimate when the form says the amount is one', () => {
        const built = buildRecurringInput(values({ estimated: true }), all)

        expect(built).toMatchObject({ ok: true, input: { template: { payments: [{ amount: -1100, estimated: true }] } } })
    })

    it('does not mark a payment that is not an estimate', () => {
        const built = buildRecurringInput(values(), all)

        expect(built.ok && 'estimated' in built.input.template.payments[0]).toBe(false)
    })

    it('makes an income of positive payments', () => {
        const built = buildRecurringInput(values({ type: 'income', default_category_id: 2 }), all)

        expect(built).toMatchObject({
            ok: true,
            input: { template: { type: 'income', payments: [{ amount: 1100 }], lines: [{ category_id: 2 }] } },
        })
    })

    it('leaves a transfer open on the side that has no amount, as an estimate', () => {
        const built = buildRecurringInput(
            values({
                type: 'transfer',
                default_category_id: null,
                payments: [{ account_id: cash.id, amount: 100 }],
                to_account_id: euroCard.id,
                to_amount: null,
            }),
            all,
        )

        expect(built).toMatchObject({
            ok: true,
            input: {
                template: {
                    type: 'transfer',
                    payments: [
                        { account_id: cash.id, amount: -100 },
                        { account_id: euroCard.id, amount: null, estimated: true },
                    ],
                    lines: [],
                },
            },
        })
    })

    it('saves the fee of a transfer as a line', () => {
        const built = buildRecurringInput(
            values({
                type: 'transfer',
                default_category_id: null,
                payments: [{ account_id: cash.id, amount: 100 }],
                to_account_id: euroCard.id,
                to_amount: 80,
                fee: { amount: 2, category_id: 3 },
            }),
            all,
        )

        expect(built).toMatchObject({
            ok: true,
            input: { template: { payments: [{ amount: -102 }, { amount: 80 }], lines: [{ category_id: 3, amount: 2 }] } },
        })
    })

    it('says which part of the payload the API would refuse', () => {
        const duplicate = buildRecurringInput(
            values({ payments: [{ account_id: cash.id, amount: 5 }], change: [{ account_id: cash.id, amount: 1 }] }),
            all,
        )
        const same = buildRecurringInput(
            values({ type: 'transfer', default_category_id: null, to_account_id: cash.id }),
            all,
        )

        expect(duplicate).toEqual({ ok: false, source: 'payload', error: 'duplicateAccount' })
        expect(same).toEqual({ ok: false, source: 'transfer', error: 'sameAccount' })
    })

    it('opens again as what was saved', () => {
        const several = values({
            payments: [
                { account_id: cash.id, amount: 70 },
                { account_id: yenWallet.id, amount: 5000 },
            ],
            change: [{ account_id: euroCard.id, amount: 2 }],
            lines: [
                { category_id: 1, currency_id: cash.currency!.id, amount: 60, note: '- milk' },
                { category_id: 3, currency_id: cash.currency!.id, amount: 32.5, note: null },
            ],
            default_category_id: null,
            tag_ids: [1],
            frequency: 'weekly',
            day_of_week: 3,
            start_date: '2026-12-09',
            end_date: '2027-06-30',
        })
        const built = buildRecurringInput(several, all)
        if (!built.ok) {
            throw new Error('not built')
        }
        const rule = recurring({ template: built.input.template, schedule: built.input.schedule })

        expect(recurringToFormValues(rule, all)).toMatchObject({
            payments: several.payments,
            change: several.change,
            lines: [
                { category_id: 1, amount: 60, note: '- milk' },
                { category_id: 3, amount: 32.5 },
            ],
            tag_ids: [1],
            frequency: 'weekly',
            day_of_week: 3,
            start_date: '2026-12-09',
            end_date: '2027-06-30',
        })
    })
})
