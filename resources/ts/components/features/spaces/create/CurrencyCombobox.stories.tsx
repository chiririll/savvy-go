import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { catalog, catalogItem } from '@/test/fixtures'
import type { CurrencyCatalogItem } from '@/types'

import { CurrencyCombobox } from './CurrencyCombobox'

// Currencies of every precision: 0 decimals for JPY, 3 for KWD.
const items: CurrencyCatalogItem[] = [
    catalogItem({ code: 'USD', name: 'US Dollar', symbol: '$', rate: 1 }),
    catalogItem({ code: 'EUR', name: 'Euro', symbol: '€', rate: 1.25 }),
    catalogItem({ code: 'JPY', name: 'Japanese Yen', symbol: '¥', decimals: 0, rate: 0.005 }),
    catalogItem({ code: 'KWD', name: 'Kuwaiti Dinar', symbol: 'KD', decimals: 3, rate: 4 }),
    ...catalog,
]

const meta = {
    tags: ['desktop-only'],
    title: 'features/spaces/create/CurrencyCombobox',
    component: CurrencyCombobox,
    parameters: { layout: 'padded', a11y: overlayA11y },
    args: { catalog: items, placeholder: 'Choose the base currency', onChange: fn(), className: 'w-72' },
    render: (args) => {
        const [value, setValue] = useState(args.value ?? null)
        return (
            <CurrencyCombobox
                {...args}
                value={value}
                onChange={(code) => {
                    setValue(code)
                    args.onChange(code)
                }}
            />
        )
    },
} satisfies Meta<typeof CurrencyCombobox>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    await userEvent.click(within(canvasElement).getByRole('combobox'))
    return screen.findByRole('textbox', { name: /search by code/i })
}

export const Empty: Story = {}
export const Chosen: Story = { args: { value: 'KWD' } }
export const Disabled: Story = { args: { disabled: true } }

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(screen.getAllByRole('option').length).toBe(items.length)
    },
}

// The popular ones come first when nothing is typed.
export const PopularFirst: Story = {
    args: { popular: ['GBP', 'EUR'] },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        const options = screen.getAllByRole('option')
        await expect(options[0]).toHaveTextContent('GBP')
        await expect(options[1]).toHaveTextContent('EUR')
    },
}

export const SearchedByName: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.type(await open(canvasElement), 'dinar')
        const options = screen.getAllByRole('option')
        await expect(options).toHaveLength(1)
        await userEvent.click(options[0])
        await expect(args.onChange).toHaveBeenCalledWith('KWD')
        await expect(within(canvasElement).getByRole('combobox')).toHaveTextContent('KWD Kuwaiti Dinar')
    },
}

export const SearchedBySymbol: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.type(await open(canvasElement), '¥')
        await expect(screen.getByRole('option', { name: /JPY/ })).toBeVisible()
    },
}

export const NothingFound: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.type(await open(canvasElement), 'zzz')
        await expect(screen.getByText(/no currency found/i)).toBeVisible()
    },
}

// The ones already chosen are left out.
export const Excluding: Story = {
    args: { exclude: new Set(['USD', 'EUR']) },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(screen.queryByRole('option', { name: /USD/ })).toBeNull()
        await expect(screen.getAllByRole('option')).toHaveLength(items.length - 2)
    },
}

// With the short built-in list any code can be typed.
export const ManualCode: Story = {
    args: { allowManual: true },
    play: async ({ canvasElement, args }) => {
        await userEvent.type(await open(canvasElement), 'thb')
        await userEvent.click(screen.getByRole('option', { name: /THB use this code/i }))
        await expect(args.onChange).toHaveBeenCalledWith('THB')
    },
}

export const NoManualCodeWithTheFullList: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.type(await open(canvasElement), 'thb')
        await expect(screen.queryByText(/use this code/i)).toBeNull()
    },
}

export const LongList: Story = {
    args: {
        catalog: Array.from({ length: 200 }, (_, i) =>
            catalogItem({ code: `C${String(i).padStart(2, '0')}`, name: `Currency ${i}` }),
        ),
    },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(screen.getByText(/more: keep typing to narrow down/i)).toBeVisible()
    },
}
