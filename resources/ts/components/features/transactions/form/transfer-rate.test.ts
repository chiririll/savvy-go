// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { useForm } from 'react-hook-form'
import { describe, expect, it } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { accounts } from '@/test/fixtures'
import type { Account } from '@/types'
import { useTransferRate } from './transfer-rate'

const [cash, euroCard] = accounts
const savings = { ...cash, id: 5, name: 'Savings' }

const base: TransactionFormValues = {
    type: 'transfer',
    description: '',
    date: '2020-05-04',
    tag_ids: [],
    payments: [{ account_id: cash.id, amount: 100 }],
    change: [],
    lines: [],
    default_category_id: null,
    default_note: null,
    to_account_id: euroCard.id,
    to_amount: null,
    exchange_rate: null,
}

function setup(
    values: Partial<TransactionFormValues>,
    from: Account | undefined,
    to: Account | undefined,
    sync?: boolean,
) {
    return renderHook(
        (props: { from?: Account; to?: Account }) => {
            const form = useForm<TransactionFormValues>({ defaultValues: { ...base, ...values } })
            const rate = useTransferRate({
                form,
                from: props.from,
                to: props.to,
                toRate: props.to?.currency?.rate,
                open: true,
                sync,
            })
            return { form, rate }
        },
        { initialProps: { from, to } },
    )
}

describe('useTransferRate without sync (the template of a recurring rule)', () => {
    it('leaves the received amount open between two currencies: it is worked out at each occurrence', () => {
        const { result } = setup({}, cash, euroCard, false)

        expect(result.current.form.getValues('to_amount')).toBeNull()
        expect(result.current.form.getValues('exchange_rate')).toBeNull()
    })

    it('has no rate to edit, but still knows the currencies differ', () => {
        const { result } = setup({}, cash, euroCard, false)

        expect(result.current.rate.sameCurrency).toBe(false)
        expect(result.current.rate.editable).toBe(false)
    })

    it('keeps an amount received that was written when the amount sent changes', () => {
        const { result } = setup({ to_amount: 80 }, cash, euroCard, false)

        act(() => result.current.form.setValue('payments.0.amount', 200))

        expect(result.current.form.getValues('to_amount')).toBe(80)
    })
})

describe('useTransferRate', () => {
    it('is not editable, and the received amount is the amount sent, in one currency', () => {
        const { result } = setup({ to_account_id: savings.id }, cash, savings)

        expect(result.current.rate.sameCurrency).toBe(true)
        expect(result.current.rate.editable).toBe(false)
        expect(result.current.form.getValues('to_amount')).toBe(100)
        expect(result.current.form.getValues('exchange_rate')).toBe(1)
    })

    it('starts from the catalog rate and works out the received amount between two currencies', () => {
        const { result } = setup({}, cash, euroCard)

        expect(result.current.rate.editable).toBe(true)
        // 1 EUR is worth 1.25 USD, so 1 USD buys 0.8 EUR.
        expect(result.current.form.getValues('exchange_rate')).toBe(0.8)
        expect(result.current.form.getValues('to_amount')).toBe(80)
    })

    it('keeps the rate of a saved transfer instead of the catalog one', () => {
        const { result } = setup({ to_amount: 90 }, cash, euroCard)

        expect(result.current.form.getValues('exchange_rate')).toBe(0.9)
        expect(result.current.form.getValues('to_amount')).toBe(90)
    })

    it('applies a rate that was typed to the received amount once it is committed', () => {
        const { result } = setup({}, cash, euroCard)

        act(() => {
            result.current.rate.startEditing()
            result.current.form.setValue('exchange_rate', 0.5)
        })
        act(() => result.current.rate.commit())

        expect(result.current.form.getValues('to_amount')).toBe(50)
    })

    it('works out the rate again from the amounts when the received amount is typed', () => {
        const { result } = setup({}, cash, euroCard)

        act(() => result.current.form.setValue('to_amount', 75))

        expect(result.current.form.getValues('exchange_rate')).toBe(0.75)
    })

    it('reads the amount sent from the first payment, and turns the rate with it', () => {
        const { result } = setup({}, cash, euroCard)

        act(() => result.current.form.setValue('payments.0.amount', 200))

        expect(result.current.form.getValues('to_amount')).toBe(80)
        expect(result.current.form.getValues('exchange_rate')).toBe(0.4)
    })

    it('leaves an expense alone', () => {
        const { result } = setup({ type: 'expense' }, cash, euroCard)

        expect(result.current.rate.editable).toBe(false)
        expect(result.current.form.getValues('to_amount')).toBeNull()
    })
})
