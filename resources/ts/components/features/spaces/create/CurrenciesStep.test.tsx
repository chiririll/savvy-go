// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { catalogItem } from '@/test/fixtures'
import { CurrenciesStep } from './CurrenciesStep'
import type { DraftContent } from './draft'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'en' } }),
}))

const REQUIRED = 'spaces.create.accounts.required'
const items = [catalogItem({ code: 'USD', name: 'US Dollar', symbol: '$', rate: 1 })]

const empty: DraftContent = { baseCurrency: 'USD', currencies: [], accounts: [], categories: [], tags: [] }
const withAccount: DraftContent = {
    ...empty,
    accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'USD', initialBalance: '' }],
}

function Harness({ initial }: { initial: DraftContent }) {
    const [content, setContent] = useState(initial)
    return (
        <CurrenciesStep
            content={content}
            catalog={items}
            onChange={(patch) => setContent((c) => ({ ...c, ...patch }))}
        />
    )
}

afterEach(() => cleanup())

describe('CurrenciesStep without an account', () => {
    it('says nothing about a missing account: the Continue button is what stays shut', () => {
        render(<Harness initial={empty} />)
        expect(screen.queryByText(REQUIRED)).toBeNull()
        expect(screen.queryByRole('alert')).toBeNull()
        expect(screen.queryByTestId('create-space-no-account')).toBeNull()
        expect(screen.queryByTestId('create-space-account-hint')).toBeNull()
    })

    it('says nothing after the last account is removed either', () => {
        render(<Harness initial={withAccount} />)
        fireEvent.click(screen.getByRole('button', { name: 'spaces.create.accounts.remove' }))
        expect(screen.queryByText(REQUIRED)).toBeNull()
        expect(screen.queryByRole('alert')).toBeNull()
    })
})

describe('CurrenciesStep catalog notice', () => {
    const RETRY = 'actions.retry'

    function Retrying({ builtIn, failed }: { builtIn?: boolean; failed?: boolean }) {
        const [state, setState] = useState({ fetching: false, builtIn, failed })
        return (
            <>
                <CurrenciesStep
                    content={empty}
                    catalog={items}
                    catalogBuiltIn={state.builtIn}
                    catalogFailed={state.failed}
                    catalogFetching={state.fetching}
                    onRetryCatalog={() => setState((s) => ({ ...s, fetching: true }))}
                    onChange={() => {}}
                />
                {/* Stands for the refetch ending: with the list recovered, or still the built-in one. */}
                <button onClick={() => setState({ fetching: false, builtIn: false, failed: false })}>recover</button>
                <button onClick={() => setState((s) => ({ ...s, fetching: false }))}>stay</button>
            </>
        )
    }

    it('shows the built-in notice with a working Retry and no spinner', () => {
        render(<Retrying builtIn />)
        expect(screen.getByText(/spaces.create.currencies.builtIn/)).toBeTruthy()
        expect((screen.getByRole('button', { name: RETRY }) as HTMLButtonElement).disabled).toBe(false)
        expect(screen.queryByTestId('create-space-catalog-spinner')).toBeNull()
    })

    it('shows a spinner and disables Retry while the refetch runs', () => {
        render(<Retrying builtIn />)
        fireEvent.click(screen.getByRole('button', { name: RETRY }))
        expect(screen.getByTestId('create-space-catalog-spinner')).toBeTruthy()
        expect((screen.getByRole('button', { name: RETRY }) as HTMLButtonElement).disabled).toBe(true)
    })

    it('says the currencies loaded when the refetch brings the real list', () => {
        render(<Retrying builtIn />)
        fireEvent.click(screen.getByRole('button', { name: RETRY }))
        fireEvent.click(screen.getByText('recover'))
        expect(screen.getByText('spaces.create.currencies.recovered')).toBeTruthy()
        expect(screen.queryByTestId('create-space-catalog-spinner')).toBeNull()
    })

    it('says it is still the built-in list after a refetch, and Retry works again', () => {
        render(<Retrying builtIn />)
        fireEvent.click(screen.getByRole('button', { name: RETRY }))
        fireEvent.click(screen.getByText('stay'))
        expect(screen.getByText(/spaces.create.currencies.stillBuiltIn/)).toBeTruthy()
        expect((screen.getByRole('button', { name: RETRY }) as HTMLButtonElement).disabled).toBe(false)
    })

    it('shows the failure again when the list could not be loaded', () => {
        render(<Retrying failed />)
        fireEvent.click(screen.getByRole('button', { name: RETRY }))
        fireEvent.click(screen.getByText('stay'))
        expect(screen.getByText(/spaces.create.currencies.failed/)).toBeTruthy()
    })
})

describe('CurrenciesStep on a phone', () => {
    it('gives the add-currency picker the full width, a fixed one from sm up', () => {
        render(<Harness initial={withAccount} />)
        const trigger = screen.getByRole('combobox', { name: 'spaces.create.currencies.add' })
        expect(trigger.className).toContain('w-full')
        expect(trigger.className).toContain('sm:w-44')
    })
})
