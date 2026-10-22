import type { Meta, StoryObj } from '@storybook/react-vite'
import { GripVertical } from 'lucide-react'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { account, currencies } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { AccountRow } from './AccountRow'

const [, eur, jpy, kwd] = currencies

const meta = {
    title: 'features/accounts/AccountRow',
    component: AccountRow,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { account: account(), onEdit: fn(), onDelete: fn() },
} satisfies Meta<typeof AccountRow>

export default meta
type Story = StoryObj<typeof meta>

export const Cash: Story = {}
export const Bank: Story = { args: { account: account({ name: 'Euro card', type: 'bank', currency: eur }) } }
export const Crypto: Story = { args: { account: account({ name: 'Cold wallet', type: 'crypto' }) } }
export const Inactive: Story = { args: { account: account({ isActive: false }) } }
export const Overdrawn: Story = { args: { account: account({ currentBalance: -35.2 }) } }
export const NoCurrency: Story = { args: { account: account({ currency: null }) } }

// The opening balance is shown only once it differs from the current one.
export const WithInitialBalance: Story = { args: { account: account({ initialBalance: 100, currentBalance: 120.5 }) } }

// Balances use the precision of their currency: 0 and 3 decimals.
export const ZeroDecimals: Story = { args: { account: account({ name: 'Yen', currency: jpy, currentBalance: 15000 }) } }
export const ThreeDecimals: Story = { args: { account: account({ name: 'Dinar', currency: kwd, currentBalance: 12.345 }) } }

export const LongName: Story = {
    args: { account: account({ name: 'An account with a name far too long for the line it has to fit on' }) },
}

export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const WithDragHandle: Story = {
    args: { dragHandle: <GripVertical className="size-4" aria-hidden /> },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Cash'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.account)
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(args.account.id)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
