// @vitest-environment jsdom
import { act, cleanup, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { FormProvider, useFieldArray, useForm } from 'react-hook-form'
import { afterEach, describe, expect, it } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { account, currencies } from '@/test/fixtures'
import { useUnlockedPayments } from './use-unlocked-payments'

afterEach(cleanup)

const [usd, eur, , kwd] = currencies
const accounts = [
    account({ id: 1, currency: usd }),
    account({ id: 2, currency: usd }),
    account({ id: 3, currency: eur }),
    account({ id: 4, currency: kwd }),
]

const amount = (value: string) => value as unknown as number
const line = (value: string, currency_id: number | null = null) => ({ category_id: 1, currency_id, amount: amount(value), note: '' })

// The hook reads the form from context, so the wrapper has to provide the form the test holds.
function withForm(values: Partial<TransactionFormValues>) {
    let form!: ReturnType<typeof useForm<TransactionFormValues>>
    const view = renderHook(
        () => {
            const { fields } = useFieldArray({ control: form.control, name: 'payments' })
            return useUnlockedPayments(accounts, fields.map((f) => f.id))
        },
        {
            wrapper: function Wrapper({ children }: { children: ReactNode }) {
                form = useForm<TransactionFormValues>({ defaultValues: { change: [], lines: [], ...values } as TransactionFormValues })
                return <FormProvider {...form}>{children}</FormProvider>
            },
        },
    )
    return { view, form: () => form }
}

describe('useUnlockedPayments', () => {
    it('offers no lock before there are lines', () => {
        const { view } = withForm({ payments: [{ account_id: 1, amount: amount('10') }] })
        expect(view.result.current).toBeNull()
    })

    it('starts following when the lines add up to the payment', () => {
        const { view } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('6'), line('4')] })
        expect(view.result.current?.[0]).toMatchObject({ follows: true, blocker: null })
    })

    it('starts fixed when the lines of a saved transaction do not add up to the payment', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('6'), line('3')] })
        expect(view.result.current?.[0]).toMatchObject({ follows: false, blocker: null })
        act(() => form().setValue('lines.1.amount', amount('9')))
        expect(form().getValues('payments.0.amount')).toBe('10')
    })

    it('starts following when the payment is still empty', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('') }], lines: [line('6'), line('4')] })
        expect(view.result.current![0].follows).toBe(true)
        expect(form().getValues('payments.0.amount')).toBe('10.00')
    })

    it('follows the lines, also a single one', () => {
        const { form } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('10')] })
        act(() => form().setValue('lines.0.amount', amount('9.5')))
        expect(form().getValues('payments.0.amount')).toBe('9.50')
    })

    it('stays put when fixed, also with a single line', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('10')] })
        act(() => view.result.current![0].toggle())
        expect(view.result.current![0].follows).toBe(false)
        act(() => form().setValue('lines.0.amount', amount('9.5')))
        expect(form().getValues('payments.0.amount')).toBe('10.00')
    })

    it('follows again once the lock is opened', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('6'), line('4')] })
        act(() => view.result.current![0].toggle())
        act(() => form().setValue('lines.1.amount', amount('5')))
        expect(form().getValues('payments.0.amount')).toBe('10.00')
        act(() => view.result.current![0].toggle())
        expect(form().getValues('payments.0.amount')).toBe('11.00')
    })

    it('goes back to the amount it had when the lines are gone', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('10') }], lines: [line('6'), line('4')] })
        act(() => form().setValue('lines.0.amount', amount('1')))
        expect(form().getValues('payments.0.amount')).toBe('5.00')
        act(() => form().setValue('lines', []))
        expect(form().getValues('payments.0.amount')).toBe('10')
        expect(view.result.current).toBeNull()
    })

    it('only follows the lines in its own currency', () => {
        const { form } = withForm({
            payments: [{ account_id: 1, amount: amount('6') }, { account_id: 3, amount: amount('4') }],
            lines: [line('6'), line('4', 2)],
        })
        act(() => form().setValue('lines.1.amount', amount('7')))
        expect(form().getValues('payments.1.amount')).toBe('7.00')
        expect(form().getValues('payments.0.amount')).toBe('6.00')
    })

    it('adds the change in the same currency', () => {
        const { form } = withForm({
            payments: [{ account_id: 1, amount: amount('20') }],
            change: [{ account_id: 2, amount: amount('5') }],
            lines: [line('15')],
        })
        expect(form().getValues('payments.0.amount')).toBe('20.00')
        act(() => form().setValue('lines.0.amount', amount('12')))
        expect(form().getValues('payments.0.amount')).toBe('17.00')
    })

    it('stays fixed with several payments in one currency', () => {
        const { view, form } = withForm({
            payments: [{ account_id: 1, amount: amount('10') }, { account_id: 2, amount: amount('5') }],
            lines: [line('15')],
        })
        expect(view.result.current![0]).toMatchObject({ follows: false, blocker: 'several' })
        expect(view.result.current![1]).toMatchObject({ follows: false, blocker: 'several' })
        act(() => form().setValue('lines.0.amount', amount('9')))
        expect(form().getValues('payments.0.amount')).toBe('10')
    })

    it('follows with the decimals of the currency', () => {
        const { form } = withForm({ payments: [{ account_id: 4, amount: amount('0.102') }], lines: [line('0.1'), line('0.002')] })
        act(() => form().setValue('lines.1.amount', amount('0.005')))
        expect(form().getValues('payments.0.amount')).toBe('0.105')
    })

    it('keeps the total when a second payment in the same currency closes the lock', () => {
        const { view, form } = withForm({ payments: [{ account_id: 1, amount: amount('') }], lines: [line('30'), line('20')] })
        expect(form().getValues('payments.0.amount')).toBe('50.00')
        act(() => form().setValue('payments', [...form().getValues('payments'), { account_id: 2, amount: amount('') }]))
        expect(form().getValues('payments.0.amount')).toBe('50.00')
        expect(view.result.current![0]).toMatchObject({ follows: false, blocker: 'several' })
    })

    it('keeps the amount and closes the lock when change comes in another currency', () => {
        const { view, form } = withForm({ payments: [{ account_id: 3, amount: amount('') }], lines: [line('450', 2)] })
        expect(form().getValues('payments.0.amount')).toBe('450.00')
        act(() => form().setValue('change', [{ account_id: 1, amount: amount('5900') }]))
        expect(form().getValues('payments.0.amount')).toBe('450.00')
        expect(view.result.current![0]).toMatchObject({ follows: false, blocker: 'foreignChange' })
        act(() => form().setValue('payments.0.amount', amount('500')))
        expect(form().getValues('payments.0.amount')).toBe('500')
    })
})
