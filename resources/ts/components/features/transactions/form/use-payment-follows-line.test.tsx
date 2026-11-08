// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { Controller, FormProvider, useFieldArray, useForm, useFormContext, useWatch } from 'react-hook-form'
import { afterEach, describe, expect, it } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { accounts } from '@/test/fixtures'
import { followAmount, useStartingAmount } from './use-payment-follows-line'

const [cash] = accounts
const empty = '' as unknown as number

/** Reads both amounts through its own subscription, as the totals and the warnings of the form do. */
function Reader({ id, seen }: { id: string; seen: string[] }) {
    const { control } = useFormContext<TransactionFormValues>()
    const [payments, lines] = useWatch({ control, name: ['payments', 'lines'] })
    const text = `${lines?.[0]?.amount}|${payments?.[0]?.amount}`
    seen.push(text)
    return <output data-testid={id}>{text}</output>
}

function Fields({ seen }: { seen: string[] }) {
    const form = useFormContext<TransactionFormValues>()
    useFieldArray({ control: form.control, name: 'payments' })
    useFieldArray({ control: form.control, name: 'lines' })
    useStartingAmount(accounts)
    return (
        <>
            <Reader id="before" seen={seen} />
            <Controller
                control={form.control}
                name="payments.0.amount"
                render={({ field }) => (
                    <input
                        aria-label="pay"
                        value={String(field.value ?? '')}
                        onChange={(e) => {
                            followAmount(form, accounts, 'payment', e.target.value)
                            field.onChange(e.target.value)
                        }}
                    />
                )}
            />
            <Controller
                control={form.control}
                name="lines.0.amount"
                render={({ field }) => (
                    <input
                        aria-label="line"
                        value={String(field.value ?? '')}
                        onChange={(e) => {
                            followAmount(form, accounts, 'line', e.target.value)
                            field.onChange(e.target.value)
                        }}
                    />
                )}
            />
            <Reader id="after" seen={seen} />
        </>
    )
}

function Harness({ values, seen }: { values?: Partial<TransactionFormValues>; seen: string[] }) {
    const form = useForm<TransactionFormValues>({
        defaultValues: {
            type: 'expense',
            payments: [{ account_id: cash.id, amount: empty }],
            change: [],
            lines: [{ category_id: 1, currency_id: null, amount: empty, note: '' }],
            ...values,
        } as TransactionFormValues,
    })
    return (
        <FormProvider {...form}>
            <Fields seen={seen} />
        </FormProvider>
    )
}

const input = (label: string) => screen.getByLabelText(label) as HTMLInputElement
const read = (id: string) => screen.getByTestId(id).textContent

describe('followAmount', () => {
    afterEach(cleanup)

    it('moves the line to the payment and back, and every reader sees both at once', () => {
        const seen: string[] = []
        render(<Harness seen={seen} />)
        seen.length = 0

        fireEvent.change(input('line'), { target: { value: '12' } })
        expect(input('pay').value).toBe('12')
        expect([read('before'), read('after')]).toEqual(['12|12', '12|12'])

        fireEvent.change(input('pay'), { target: { value: '15.5' } })
        expect(input('line').value).toBe('15.5')
        expect([read('before'), read('after')]).toEqual(['15.5|15.5', '15.5|15.5'])
        // Never a render where they disagree.
        expect(seen.every((text) => text.split('|')[0] === text.split('|')[1])).toBe(true)
    })

    it('leaves a saved line that covers less alone', () => {
        render(
            <Harness
                seen={[]}
                values={{
                    payments: [{ account_id: cash.id, amount: 50 }],
                    lines: [{ category_id: 1, currency_id: null, amount: 40, note: '' }],
                }}
            />,
        )
        fireEvent.change(input('line'), { target: { value: '45' } })
        expect(input('pay').value).toBe('50')
    })

    it('takes an amount an opener gave the payment into the line', () => {
        render(<Harness seen={[]} values={{ payments: [{ account_id: cash.id, amount: 30 }] }} />)
        expect(input('line').value).toBe('30')
    })
})
