// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CurrencyCatalogItem } from '@/types'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))

import { CurrencyCombobox } from './CurrencyCombobox'

afterEach(() => cleanup())

const item = (code: string, name: string, symbol: string): CurrencyCatalogItem => ({
    code,
    name,
    symbol,
    decimals: 2,
    rate: null,
})
const catalog = [item('EUR', 'Euro', '€'), item('JPY', 'Japanese Yen', '¥'), item('USD', 'US Dollar', '$')]

const open = () => fireEvent.click(screen.getByRole('combobox'))
const search = (value: string) =>
    fireEvent.change(screen.getByLabelText('spaces.create.currencies.search'), { target: { value } })

describe('CurrencyCombobox', () => {
    it('finds a currency by its name and picks it', () => {
        const onChange = vi.fn()
        render(<CurrencyCombobox catalog={catalog} placeholder="pick" onChange={onChange} />)
        open()
        search('yen')
        const options = screen.getAllByRole('option')
        expect(options).toHaveLength(1)
        expect(options[0]!.textContent).toMatch(/^JPY/)
        fireEvent.click(options[0]!)
        expect(onChange).toHaveBeenCalledWith('JPY')
    })

    it('leaves out the excluded codes', () => {
        render(<CurrencyCombobox catalog={catalog} exclude={new Set(['USD'])} placeholder="pick" onChange={vi.fn()} />)
        open()
        expect(screen.getAllByRole('option').map((o) => o.textContent?.slice(0, 3))).toEqual(['EUR', 'JPY'])
    })

    it('lets a code that is not in the list be typed only when allowed', () => {
        const onChange = vi.fn()
        const { rerender } = render(<CurrencyCombobox catalog={catalog} placeholder="pick" onChange={onChange} />)
        open()
        search('sek')
        expect(screen.queryAllByRole('option')).toHaveLength(0)

        rerender(<CurrencyCombobox catalog={catalog} allowManual placeholder="pick" onChange={onChange} />)
        const manual = screen.getByRole('option')
        expect(manual.textContent).toMatch(/^SEK/)
        fireEvent.click(manual)
        expect(onChange).toHaveBeenCalledWith('SEK')
    })
})
