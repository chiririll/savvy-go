// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it } from 'vitest'
import { currencies } from '@/test/fixtures'
import type { Currency } from '@/types'
import { AmountInput } from './AmountInput'

afterEach(cleanup)

const [usd, , jpy, kwd] = currencies
const btc: Currency = { id: 9, code: 'BTC', name: 'Bitcoin', symbol: '₿', decimals: 8, isBase: false, rate: 60000 }

/** Holds the value as the form does, and keeps what the form was told. */
function Field({ currency, start = '', readOnly, sent = [] }: { currency: Currency; start?: string; readOnly?: boolean; sent?: string[] }) {
    const [value, setValue] = useState(start)
    return (
        <AmountInput
            aria-label="amount"
            currency={currency}
            value={value}
            onChange={(next) => {
                sent.push(next)
                setValue(next)
            }}
            readOnly={readOnly}
        />
    )
}

const box = () => screen.getByLabelText('amount') as HTMLInputElement
const type = (text: string) => fireEvent.change(box(), { target: { value: text } })

describe('AmountInput formulas', () => {
    it('keeps the formula while typing and evaluates it when the focus is lost', () => {
        render(<Field currency={usd} />)
        type('12,5*3 + (40-5)/2')
        expect(box().value).toBe('12,5*3 + (40-5)/2')
        fireEvent.blur(box())
        expect(box().value).toBe('55.00')
    })

    it('keeps a typed formula from the form until it is evaluated', () => {
        const sent: string[] = []
        render(<Field currency={usd} start="10" sent={sent} />)
        type('10+')
        type('10+5')
        expect(sent).toEqual([])
        fireEvent.blur(box())
        expect(sent).toEqual(['15.00'])
    })

    it('evaluates a formula on Enter, and does not submit the form with it', () => {
        render(<Field currency={usd} />)
        type('2*3')
        const notPrevented = fireEvent.keyDown(box(), { key: 'Enter' })
        expect(box().value).toBe('6.00')
        expect(notPrevented).toBe(false)
    })

    it('lets Enter through on a plain amount, so it submits as before', () => {
        render(<Field currency={usd} />)
        type('12')
        expect(fireEvent.keyDown(box(), { key: 'Enter' })).toBe(true)
    })

    it('tells the form a formula that does not work, so saving is refused', () => {
        const sent: string[] = []
        render(<Field currency={usd} start="10" sent={sent} />)
        type('1/0')
        fireEvent.blur(box())
        expect(sent).toEqual(['1/0'])
    })

    it('shows what the form has again when it changes the amount', () => {
        function Outer() {
            const [value, setValue] = useState('10')
            return (
                <>
                    <AmountInput aria-label="amount" currency={usd} value={value} onChange={setValue} />
                    <button onClick={() => setValue('42.00')}>set</button>
                </>
            )
        }
        render(<Outer />)
        type('1+1')
        fireEvent.click(screen.getByText('set'))
        expect(box().value).toBe('42.00')
    })

    it.each([
        [usd, '10/3', '3.33'],
        [jpy, '10/4', '3'],
        [kwd, '2/3', '0.667'],
        [btc, '1/3', '0.33333333'],
    ])('rounds to the decimals of %j', (currency, formula, expected) => {
        render(<Field currency={currency} />)
        type(formula)
        fireEvent.blur(box())
        expect(box().value).toBe(expected)
    })

    it('leaves a formula that does not work as typed, and marks the field', () => {
        render(<Field currency={usd} />)
        type('1/0')
        fireEvent.blur(box())
        expect(box().value).toBe('1/0')
        expect(box().getAttribute('aria-invalid')).toBe('true')
        type('1/2')
        expect(box().getAttribute('aria-invalid')).not.toBe('true')
    })

    it('leaves a plain amount alone and still cuts the decimals of the currency', () => {
        render(<Field currency={jpy} />)
        type('12.5')
        expect(box().value).toBe('12')
        fireEvent.blur(box())
        expect(box().value).toBe('12')
    })

    it('does not take or evaluate anything when read-only', () => {
        render(<Field currency={usd} start="1+1" readOnly />)
        fireEvent.blur(box())
        expect(box().value).toBe('1+1')
        expect(box().readOnly).toBe(true)
    })
})
