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

function Harness({ initial, attempted = false }: { initial: DraftContent; attempted?: boolean }) {
    const [content, setContent] = useState(initial)
    return (
        <CurrenciesStep
            content={content}
            catalog={items}
            accountsAttempted={attempted}
            onChange={(patch) => setContent((c) => ({ ...c, ...patch }))}
        />
    )
}

afterEach(() => cleanup())

describe('CurrenciesStep account message', () => {
    it('is a plain hint when the step is first opened', () => {
        render(<Harness initial={empty} />)
        expect(screen.getByText(REQUIRED)).toBeTruthy()
        expect(screen.queryByRole('alert')).toBeNull()
    })

    it('turns into an error once Next was tried', () => {
        render(<Harness initial={empty} attempted />)
        expect(screen.getByRole('alert').textContent).toBe(REQUIRED)
    })

    it('turns into an error when the last account is removed', () => {
        render(<Harness initial={withAccount} />)
        expect(screen.queryByText(REQUIRED)).toBeNull()
        fireEvent.click(screen.getByRole('button', { name: 'spaces.create.accounts.remove' }))
        expect(screen.getByRole('alert').textContent).toBe(REQUIRED)
    })

    it('shows nothing while an account exists, even after an attempt', () => {
        render(<Harness initial={withAccount} attempted />)
        expect(screen.queryByText(REQUIRED)).toBeNull()
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
