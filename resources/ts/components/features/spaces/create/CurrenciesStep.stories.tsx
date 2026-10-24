import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { catalog, catalogItem } from '@/test/fixtures'

import { CurrenciesStep } from './CurrenciesStep'
import type { DraftContent } from './draft'

const items = [
    catalogItem({ code: 'USD', name: 'US Dollar', symbol: '$', rate: 1 }),
    catalogItem({ code: 'EUR', name: 'Euro', symbol: '€', rate: 1.25 }),
    catalogItem({ code: 'JPY', name: 'Japanese Yen', symbol: '¥', decimals: 0, rate: 0.005 }),
    catalogItem({ code: 'KWD', name: 'Kuwaiti Dinar', symbol: 'KD', decimals: 3, rate: 4 }),
    ...catalog,
]

const seeded: DraftContent = {
    baseCurrency: 'EUR',
    currencies: ['JPY'],
    accounts: [{ key: 'a1', name: 'Yen wallet', type: 'cash', currency: 'JPY', initialBalance: '15000' }],
    categories: [],
    tags: [],
}

const empty: DraftContent = { baseCurrency: null, currencies: [], accounts: [], categories: [], tags: [] }

const changed = fn()

const Harness = ({ initial, ...rest }: { initial: DraftContent } & Partial<Parameters<typeof CurrenciesStep>[0]>) => {
    const [content, setContent] = useState(initial)
    return (
        <CurrenciesStep
            content={content}
            catalog={items}
            {...rest}
            onChange={(patch) => {
                setContent((c) => ({ ...c, ...patch }))
                changed(patch)
            }}
        />
    )
}

const meta = {
    title: 'features/spaces/create/CurrenciesStep',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: () => changed.mockClear(),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Seeded: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('combobox', { name: 'Base currency' })).toHaveTextContent('EUR Euro')
        // JPY cannot go while an account uses it.
        await expect(canvas.getByRole('button', { name: 'Remove JPY' })).toBeDisabled()
    },
}

// First opened with no account: a hint, not yet an error.
export const Empty: Story = {
    render: () => <Harness initial={empty} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/add at least one account/i)).toBeVisible()
        await expect(canvas.queryByRole('alert')).toBeNull()
        // Accounts need a currency, and other currencies need a base first.
        await expect(canvas.getByRole('button', { name: /add an account/i })).toBeDisabled()
        await expect(canvas.getByRole('combobox', { name: /add a currency/i })).toBeDisabled()
    },
}

export const AccountRequiredAfterAttempt: Story = {
    render: () => <Harness initial={{ ...empty, baseCurrency: 'EUR' }} accountsAttempted />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('alert')).toHaveTextContent(/add at least one account/i)
    },
}

export const CatalogLoading: Story = {
    render: () => <Harness initial={empty} catalog={undefined} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/loading currencies/i)).toBeVisible()
        await expect(canvas.getByRole('combobox', { name: 'Base currency' })).toBeDisabled()
    },
}

export const CatalogFailed: Story = {
    render: () => {
        const retry = fn()
        return <Harness initial={empty} catalog={undefined} catalogFailed onRetryCatalog={retry} />
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/could not load the currency list/i)).toBeVisible()
        await expect(canvas.queryByText(/loading currencies/i)).toBeNull()
        await expect(canvas.getByRole('button', { name: /try again/i })).toBeVisible()
    },
}

export const BuiltInCatalog: Story = {
    render: () => <Harness initial={empty} catalogBuiltIn />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('status')).toHaveTextContent(/currency service is unavailable/i)
    },
}

// Picking a base that is already among the others moves it out of them.
export const BaseTakenFromTheOthers: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox', { name: 'Base currency' }))
        await userEvent.type(await screen.findByRole('textbox', { name: /search/i }), 'yen')
        await userEvent.click(screen.getByRole('option', { name: /JPY/ }))
        await expect(changed).toHaveBeenLastCalledWith({ baseCurrency: 'JPY', currencies: [] })
    },
}

export const CurrencyAdded: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('combobox', { name: /add a currency/i }))
        // The base and the ones listed are not offered again.
        await expect(screen.queryByRole('option', { name: /EUR/ })).toBeNull()
        await userEvent.click(await screen.findByRole('option', { name: /KWD/ }))
        await expect(changed).toHaveBeenLastCalledWith({ currencies: ['JPY', 'KWD'] })
        await expect(await canvas.findByRole('button', { name: 'Remove KWD' })).toBeEnabled()
    },
}

// On a phone the picker fills the row, so its list is as wide as the row.
export const AddCurrencyOnAPhone: Story = {
    tags: ['mobile-only'],
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const trigger = within(canvasElement).getByRole('combobox', { name: /add a currency/i })
        const row = trigger.parentElement!
        await expect(trigger.getBoundingClientRect().width).toBeGreaterThan(row.getBoundingClientRect().width * 0.9)
        await userEvent.click(trigger)
        // The list opens, and the whole of it fits the screen.
        const list = await screen.findByRole('listbox')
        await expect(list.getBoundingClientRect().right).toBeLessThanOrEqual(window.innerWidth)
    },
}

export const AccountAdded: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /add an account/i }))
        const names = canvas.getAllByRole('textbox', { name: 'Name' })
        await userEvent.type(names[1], 'Kuwait savings')
        // A new account starts in the base currency; only the space's currencies are offered.
        const currency = canvas.getAllByRole('combobox', { name: 'Currency' })[1]
        await expect(currency).toHaveTextContent('EUR')
        await userEvent.click(currency)
        await expect(await screen.findAllByRole('option')).toHaveLength(2)
        await userEvent.click(screen.getByRole('option', { name: 'JPY' }))
        // A balance is kept as typed: a decimal string, never a float.
        await userEvent.type(canvas.getAllByRole('textbox', { name: 'Opening balance' })[1], '0.1')
        await waitFor(() =>
            expect(changed).toHaveBeenLastCalledWith({
                accounts: [
                    seeded.accounts[0],
                    expect.objectContaining({ name: 'Kuwait savings', currency: 'JPY', initialBalance: '0.1' }),
                ],
            }),
        )
    },
}

export const AccountTypeChanged: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox', { name: 'Type' }))
        await userEvent.click(await screen.findByRole('option', { name: /bank/i }))
        await expect(changed).toHaveBeenLastCalledWith({ accounts: [{ ...seeded.accounts[0], type: 'bank' }] })
    },
}

// Removing the last account turns the hint into an error, and frees the currency.
export const AccountRemoved: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByText(/add at least one account/i)).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /remove the account/i }))
        await expect(canvas.getByRole('alert')).toHaveTextContent(/add at least one account/i)
        await expect(canvas.getByRole('button', { name: 'Remove JPY' })).toBeEnabled()
    },
}
