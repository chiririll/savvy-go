import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { account, currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { ReconcileRowItem } from './ReconcileRowItem'

const meta = {
    title: 'features/accounts/ReconcileRowItem',
    // The row is an <li>, so it needs a list around it.
    render: (args) => (
        <ul>
            <ReconcileRowItem {...args} />
        </ul>
    ),
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        account: account(),
        row: { accountId: 1, amount: '120.50', computed: '120.50', edited: false },
        difference: '0.00',
        onChange: fn(),
        onRemove: fn(),
    },
} satisfies Meta<typeof ReconcileRowItem>

export default meta
type Story = StoryObj<typeof meta>

export const Zero: Story = {
    play: async ({ canvasElement }) => {
        const difference = within(canvasElement).getByText(/difference:/i)
        await expect(difference).toHaveClass('text-muted-foreground')
        await expect(difference).not.toHaveClass('text-success')
        await expect(difference).not.toHaveClass('text-destructive')
        await expect(difference).not.toHaveTextContent('+')
    },
}

export const Positive: Story = {
    args: { row: { accountId: 1, amount: '130.00', computed: '120.50', edited: true }, difference: '9.50' },
    play: async ({ canvasElement }) => {
        const difference = within(canvasElement).getByText(/difference:/i)
        await expect(difference).toHaveClass('text-success')
        await expect(difference).toHaveTextContent(/\+.*9\.50/)
    },
}

export const Negative: Story = {
    args: { row: { accountId: 1, amount: '100.00', computed: '120.50', edited: true }, difference: '-20.50' },
    play: async ({ canvasElement }) => {
        const difference = within(canvasElement).getByText(/difference:/i)
        await expect(difference).toHaveClass('text-destructive')
        await expect(difference).toHaveTextContent(/20\.50/)
        await expect(difference).not.toHaveTextContent('+')
    },
}

// The difference is null while the balances of the date are loading.
export const NoDifference: Story = {
    args: { difference: null, disabled: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/difference:/i)).toBeNull()
        await expect(within(canvasElement).getByRole('textbox', { name: /actual balance of cash/i })).toBeDisabled()
    },
}

export const ZeroDecimalCurrency: Story = {
    args: {
        account: account({ id: 3, name: 'Yen wallet', currency: currencies[2] }),
        row: { accountId: 3, amount: '15500', computed: '15000', edited: true },
        difference: '500',
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/difference:/i)).toHaveClass('text-success')
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox', { name: /actual balance of cash/i }), '1')
        await expect(args.onChange).toHaveBeenCalled()
    },
}

export const Removed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /remove cash/i }))
        await expect(args.onRemove).toHaveBeenCalledOnce()
    },
}
