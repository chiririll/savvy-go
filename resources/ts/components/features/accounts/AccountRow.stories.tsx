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
const creditCard = {
    limit: 2000, cycle: 'fixed_day', statementDay: 28, graceDays: 21, used: 500, available: 1500,
    periodStart: '2026-11-01', periodEnd: '2026-11-30', spent: 100,
    statement: { date: '2026-10-28', balance: 400, dueDate: '2026-11-18', paid: 0, amountDue: 400 },
    status: { level: 'ok' as const, amount: 0 },
}
const card = (overrides = {}) => account({
    name: 'Visa Gold', type: 'credit', currentBalance: -500, creditCard: { ...creditCard, ...overrides },
})

export const CreditCard: Story = { args: { account: card() } }
export const CreditCardWarning: Story = {
    args: { account: card({ used: 1600, available: 400, status: { level: 'warning', reason: 'payment_due', amount: 400, date: '2026-11-18' } }) },
}
export const CreditCardDanger: Story = {
    args: { account: card({ used: 2300, available: -300, status: { level: 'danger', reason: 'over_limit', amount: 300 } }) },
}

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
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(args.account.id)
    },
}

