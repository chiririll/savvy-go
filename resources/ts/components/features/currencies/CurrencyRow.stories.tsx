import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { currencies, currency } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'

import { CurrencyRow } from './CurrencyRow'

const [usd, eur, jpy, kwd] = currencies

const meta = {
    title: 'features/currencies/CurrencyRow',
    component: CurrencyRow,
    parameters: { a11y: overlayA11y },
    args: { currency: eur, onEdit: fn(), onDelete: fn(), onSetBase: fn() },
} satisfies Meta<typeof CurrencyRow>

export default meta
type Story = StoryObj<typeof meta>

const openMenu = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))

export const Foreign: Story = {}
export const Base: Story = { args: { currency: usd } }
export const ZeroDecimals: Story = { args: { currency: jpy } }
export const ThreeDecimals: Story = { args: { currency: kwd } }
export const AutoUpdated: Story = { args: { autoUpdated: true } }
export const BaseIsNeverAutoUpdated: Story = { args: { currency: usd, autoUpdated: true } }
export const UnknownCodeGetsATone: Story = {
    args: { currency: currency({ id: 9, code: 'XYZ', name: 'Made-up money', symbol: 'Ӿ', isBase: false, rate: 3.14159265 }) },
}
export const TinyRate: Story = {
    args: {
        currency: currency({ id: 10, code: 'BTC', name: 'Bitcoin', symbol: '₿', decimals: 8, isBase: false, rate: 0.000015 }),
    },
}
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const DeleteDisabled: Story = {
    args: { currency: usd, deleteDisabled: true, deleteDisabledLabel: 'The base currency cannot be deleted' },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: /base currency/i })).toHaveAttribute('aria-disabled', 'true')
    },
}

export const MadeBase: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /base/i }))
        await expect(args.onSetBase).toHaveBeenCalledWith(eur.id)
    },
}

export const BaseCannotBeMadeBase: Story = {
    args: { currency: usd },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await screen.findByRole('menuitem', { name: /edit/i })
        await expect(screen.queryByRole('menuitem', { name: /set as base/i })).toBeNull()
    },
}

export const SettingBase: Story = {
    args: { isSettingBase: true },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: /base/i })).toHaveAttribute('aria-disabled', 'true')
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(eur.id)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
