import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { catalog, catalogItem } from '@/test/fixtures'

import type { DraftContent } from './draft'
import { EntitiesStep } from './EntitiesStep'

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
    categories: [
        { name: '#OTHER', type: 'expense', icon: '📦', color: '#6b7280', isDefault: true },
        { name: 'Groceries', type: 'expense', icon: '🛒', color: '#16a34a', isDefault: false },
        { name: '#OTHER_INCOME', type: 'income', icon: '💰', color: '#2563eb', isDefault: true },
    ],
    tags: ['holiday'],
}

const empty: DraftContent = { baseCurrency: null, currencies: [], accounts: [], categories: [], tags: [] }

const changed = fn()

const Harness = ({ initial, ...rest }: { initial: DraftContent } & Partial<Parameters<typeof EntitiesStep>[0]>) => {
    const [content, setContent] = useState(initial)
    return (
        <EntitiesStep
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
    title: 'features/spaces/create/EntitiesStep',
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
        // The default category of each type stays.
        await expect(canvas.getByRole('button', { name: /remove other$/i })).toBeDisabled()
    },
}

export const InRussian: Story = { render: () => <Harness initial={seeded} />, globals: { locale: 'ru' } }
export const InDarkTheme: Story = { render: () => <Harness initial={seeded} />, globals: { theme: 'dark' } }

export const Empty: Story = {
    render: () => <Harness initial={empty} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/add at least one account/i)).toBeVisible()
        // Accounts need a currency, and other currencies need a base first.
        await expect(canvas.getByRole('button', { name: /add an account/i })).toBeDisabled()
        await expect(canvas.getByRole('combobox', { name: /add a currency/i })).toBeDisabled()
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
        // The account's type comes first; the other "Type" is the new category's.
        await userEvent.click(within(canvasElement).getAllByRole('combobox', { name: 'Type' })[0])
        await userEvent.click(await screen.findByRole('option', { name: /bank/i }))
        await expect(changed).toHaveBeenLastCalledWith({ accounts: [{ ...seeded.accounts[0], type: 'bank' }] })
    },
}

export const AccountRemoved: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /remove the account/i }))
        await expect(canvas.getByText(/add at least one account/i)).toBeVisible()
        // With no account on it, JPY can go now.
        await expect(canvas.getByRole('button', { name: 'Remove JPY' })).toBeEnabled()
    },
}

export const CategoryAddedAndRemoved: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getAllByRole('combobox', { name: 'Type' })[1])
        await userEvent.click(await screen.findByRole('option', { name: /income/i }))
        await userEvent.type(canvas.getByRole('textbox', { name: /new category/i }), ' Bonus {Enter}')
        await expect(changed).toHaveBeenLastCalledWith({
            categories: [
                ...seeded.categories,
                { name: 'Bonus', type: 'income', icon: '🏷️', color: '#6b7280', isDefault: false },
            ],
        })
        await userEvent.click(canvas.getByRole('button', { name: 'Remove Groceries' }))
        await expect(canvas.queryByDisplayValue('Groceries')).toBeNull()
    },
}

export const BlankCategoryIsNotAdded: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox', { name: /new category/i }), '   {Enter}')
        await expect(changed).not.toHaveBeenCalled()
    },
}

// A tag is added once, whatever its case.
export const TagAdded: Story = {
    render: () => <Harness initial={seeded} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const field = canvas.getByRole('textbox', { name: /new tag/i })
        await userEvent.type(field, 'HOLIDAY{Enter}')
        await expect(changed).not.toHaveBeenCalled()
        await expect(field).toHaveValue('')
        await userEvent.type(field, 'work{Enter}')
        await expect(changed).toHaveBeenLastCalledWith({ tags: ['holiday', 'work'] })
        await userEvent.click(canvas.getByRole('button', { name: /remove holiday/i }))
        await expect(changed).toHaveBeenLastCalledWith({ tags: ['work'] })
    },
}
