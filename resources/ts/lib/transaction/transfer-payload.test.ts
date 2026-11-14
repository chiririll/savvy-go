import { describe, expect, it } from 'vitest'

import { accounts } from '@/test/fixtures'
import {
    buildSpaceTransferInput,
    buildTransferInput,
    swapTransfer,
    transferErrorPath,
    transferReceived,
    type TransferValues,
} from './transfer-payload'

const [cash, euroCard, yenWallet] = accounts
const sameCurrencyAccount = { ...euroCard, id: 9, name: 'Euro savings' }
const all = [...accounts, sameCurrencyAccount]

const values = (overrides: Partial<TransferValues> = {}): TransferValues => ({
    type: 'transfer',
    description: 'To the card',
    date: '2026-11-07',
    tag_ids: [],
    payments: [{ account_id: cash.id, amount: 100 }],
    to_account_id: euroCard.id,
    to_amount: 80,
    ...overrides,
})

describe('transferReceived', () => {
    it('is what was sent when both accounts share a currency', () => {
        expect(transferReceived(values({ payments: [{ account_id: euroCard.id, amount: 30 }], to_account_id: 9, to_amount: 5 }), all)).toBe(30)
    })

    it('is the amount the user wrote when the currencies differ', () => {
        expect(transferReceived(values(), all)).toBe(80)
    })

    it('is nothing while no amount was written for different currencies', () => {
        expect(transferReceived(values({ to_amount: null }), all)).toBe(0)
    })
})

describe('buildTransferInput', () => {
    it('makes a payment that takes money and one that brings it, each in its own account', () => {
        const built = buildTransferInput(values(), all)

        expect(built).toEqual({
            ok: true,
            input: {
                type: 'transfer',
                description: 'To the card',
                date: '2026-11-07',
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: 80 },
                ],
                lines: [],
                tag_ids: [],
            },
        })
    })

    it('sends the same amount both ways between accounts of one currency', () => {
        const built = buildTransferInput(
            values({ payments: [{ account_id: euroCard.id, amount: 30 }], to_account_id: 9, to_amount: null }),
            all,
        )

        expect(built.ok && built.input.payments).toEqual([
            { account_id: euroCard.id, amount: -30 },
            { account_id: 9, amount: 30 },
        ])
    })

    it('takes the fee from the source on top and saves it as a line', () => {
        const built = buildTransferInput(
            values({
                payments: [{ account_id: euroCard.id, amount: 200 }],
                to_account_id: 9,
                fee: { amount: 1.5, category_id: 7 },
            }),
            all,
        )

        expect(built.ok && built.input.payments).toEqual([
            { account_id: euroCard.id, amount: -201.5 },
            { account_id: 9, amount: 200 },
        ])
        expect(built.ok && built.input.lines).toEqual([
            { category_id: 7, currency_id: euroCard.currency!.id, amount: 1.5 },
        ])
    })

    it('rounds amounts to the decimals of each currency', () => {
        const built = buildTransferInput(
            values({ payments: [{ account_id: cash.id, amount: 0.1 + 0.2 }], to_account_id: yenWallet.id, to_amount: 60.4 }),
            all,
        )

        expect(built.ok && built.input.payments).toEqual([
            { account_id: cash.id, amount: -0.3 },
            { account_id: yenWallet.id, amount: 60 },
        ])
    })

    it('leaves a fee that is empty or zero out', () => {
        const built = buildTransferInput(values({ fee: { amount: 0, category_id: 7 } }), all)

        expect(built.ok && built.input.lines).toEqual([])
    })

    it('leaves a description that is blank empty and a missing date null', () => {
        const built = buildTransferInput(values({ description: '  ', date: '' }), all)

        expect(built.ok && built.input.description).toBeNull()
        expect(built.ok && built.input.date).toBeNull()
    })

    it.each([
        ['an unknown source', values({ payments: [{ account_id: 99, amount: 1 }] }), 'unknownFrom'],
        ['an unknown destination', values({ to_account_id: 99 }), 'unknownTo'],
        ['the same account twice', values({ to_account_id: cash.id }), 'sameAccount'],
        ['no amount sent', values({ payments: [{ account_id: cash.id, amount: 0 }] }), 'amountZero'],
        ['no amount received', values({ to_amount: 0 }), 'receivedZero'],
        ['a fee that is negative', values({ fee: { amount: -1, category_id: null } }), 'feeInvalid'],
    ] as const)('refuses %s', (_name, input, error) => {
        expect(buildTransferInput(input, all)).toEqual({ ok: false, error })
    })
})

describe('buildTransferInput for a template', () => {
    it('leaves the amount received open between currencies: each occurrence works it out at its rates', () => {
        const built = buildTransferInput(values({ to_amount: null }), all, { template: true })

        expect(built).toMatchObject({
            ok: true,
            input: {
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: null },
                ],
            },
        })
    })

    it('keeps an amount received that was written', () => {
        const built = buildTransferInput(values(), all, { template: true })

        expect(built).toMatchObject({ ok: true, input: { payments: [{ amount: -100 }, { amount: 80 }] } })
    })

    it('still sends the same amount both ways in one currency', () => {
        const built = buildTransferInput(
            values({ payments: [{ account_id: euroCard.id, amount: 30 }], to_account_id: 9, to_amount: null }),
            all,
            { template: true },
        )

        expect(built).toMatchObject({ ok: true, input: { payments: [{ amount: -30 }, { amount: 30 }] } })
    })

    it('takes the fee on top of what is sent, as a transfer does', () => {
        const built = buildTransferInput(values({ to_amount: null, fee: { amount: 2, category_id: 4 } }), all, { template: true })

        expect(built).toMatchObject({
            ok: true,
            input: {
                payments: [{ amount: -102 }, { amount: null }],
                lines: [{ category_id: 4, amount: 2 }],
            },
        })
    })

    it('still refuses without it: a transfer made now has an amount received', () => {
        expect(buildTransferInput(values({ to_amount: null }), all)).toEqual({ ok: false, error: 'receivedZero' })
    })
})

describe('buildSpaceTransferInput', () => {
    const space = (overrides: Partial<TransferValues> = {}) =>
        values({ type: 'transfer_out', to_space_id: 5, to_account_id: 12, to_amount: 90, ...overrides })

    it('names the space, both accounts, both amounts and the date', () => {
        expect(buildSpaceTransferInput(space(), all)).toEqual({
            ok: true,
            input: {
                to_space_id: 5,
                from_account_id: cash.id,
                to_account_id: 12,
                from_amount: '100',
                to_amount: '90',
                date: '2026-11-07',
                description: 'To the card',
            },
        })
    })

    it('rounds what is sent and sends a blank description as null', () => {
        const built = buildSpaceTransferInput(space({ payments: [{ account_id: cash.id, amount: 0.1 + 0.2 }], description: ' ' }), all)

        expect(built.ok && built.input.from_amount).toBe('0.3')
        expect(built.ok && built.input.description).toBeNull()
    })

    it('receives what was sent when no other amount was written', () => {
        const built = buildSpaceTransferInput(space({ to_amount: null }), all)

        expect(built.ok && built.input.to_amount).toBe('100')
    })

    it.each([
        ['no space', space({ to_space_id: null }), 'unknownSpace'],
        ['an unknown source', space({ payments: [{ account_id: 99, amount: 1 }] }), 'unknownFrom'],
        ['no destination account', space({ to_account_id: null }), 'unknownTo'],
        ['no amount sent', space({ payments: [{ account_id: cash.id, amount: 0 }] }), 'amountZero'],
    ] as const)('refuses %s', (_name, input, error) => {
        expect(buildSpaceTransferInput(input, all)).toEqual({ ok: false, error })
    })

    it('does not mistake the destination in another space for an account of this one', () => {
        // Account 2 exists here, but the destination is in another space: it is never looked up.
        const built = buildSpaceTransferInput(space({ to_account_id: euroCard.id }), all)

        expect(built.ok && built.input.to_account_id).toBe(euroCard.id)
    })
})

describe('transferErrorPath', () => {
    it.each([
        ['unknownSpace', 'to_space_id'],
        ['unknownFrom', 'payments.0.account_id'],
        ['unknownTo', 'to_account_id'],
        ['sameAccount', 'to_account_id'],
        ['amountZero', 'payments.0.amount'],
        ['receivedZero', 'to_amount'],
        ['feeInvalid', 'fee.amount'],
    ] as const)('puts %s on %s', (error, path) => {
        expect(transferErrorPath(error)).toBe(path)
    })
})

describe('swapTransfer', () => {
    it('does nothing until both accounts are chosen', () => {
        expect(swapTransfer({ payments: [{ account_id: 1, amount: 10 }], to_account_id: null })).toBeNull()
        expect(swapTransfer({ payments: [{ account_id: 0, amount: 10 }], to_account_id: 2 })).toBeNull()
    })

    it('swaps the accounts and their amounts and inverts the rate', () => {
        expect(
            swapTransfer({
                payments: [{ account_id: 1, amount: 10 }],
                to_account_id: 2,
                to_amount: 25,
                exchange_rate: 2.5,
            }),
        ).toEqual({ account_id: 2, to_account_id: 1, amount: 25, to_amount: 10, exchange_rate: 0.4 })
    })

    it('keeps an empty amount empty and leaves the rate alone when there is none', () => {
        expect(swapTransfer({ payments: [{ account_id: 1, amount: 10 }], to_account_id: 2 })).toEqual({
            account_id: 2,
            to_account_id: 1,
            amount: null,
            to_amount: 10,
            exchange_rate: null,
        })
    })

    it('reads ids that arrive as numeric strings', () => {
        expect(swapTransfer({ payments: [{ account_id: '3' as unknown as number, amount: 1 }], to_account_id: '4' as unknown as number })).toMatchObject({
            account_id: 4,
            to_account_id: 3,
        })
    })
})
