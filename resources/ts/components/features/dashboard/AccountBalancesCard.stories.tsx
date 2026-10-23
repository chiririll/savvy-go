import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { account, accounts, currencies } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { AccountBalancesCard } from './AccountBalancesCard'

const meta = {
    title: 'features/dashboard/AccountBalancesCard',
    component: AccountBalancesCard,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { accounts, openCreate: fn() },
} satisfies Meta<typeof AccountBalancesCard>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Empty: Story = { args: { accounts: [] } }
export const Loading: Story = { args: { accounts: undefined } }
export const Overdrawn: Story = { args: { accounts: [account({ currentBalance: -35.5 })] } }
export const LongName: Story = {
    args: { accounts: [account({ name: 'An account with a name much too long for the line it has to fit on' })] },
}
// Each balance is shown in its own currency's precision, without repeating the symbol in the number.
export const Precisions: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('15,000')).toBeVisible()
        await expect(canvas.getByText('12.345')).toBeVisible()
    },
}

export const QuickAddIncome: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /create/i })[0])
        await userEvent.click(await screen.findByRole('menuitem', { name: /income/i }))
        await expect(args.openCreate).toHaveBeenCalledWith({ type: 'income', account_id: 1 })
    },
}

export const QuickAddTransfer: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /create/i })[1])
        await userEvent.click(await screen.findByRole('menuitem', { name: /transfer/i }))
        await expect(args.openCreate).toHaveBeenCalledWith({ type: 'transfer', account_id: 2 })
    },
}
