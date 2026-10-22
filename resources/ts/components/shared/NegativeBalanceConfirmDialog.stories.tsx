import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { NegativeBalanceConfirmDialog } from './NegativeBalanceConfirmDialog'

const [usd, , jpy, kwd] = currencies

const meta = {
    title: 'shared/NegativeBalanceConfirmDialog',
    component: NegativeBalanceConfirmDialog,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        open: true,
        onOpenChange: fn(),
        onConfirm: fn(),
        warnings: [{ accountName: 'Cash', currentBalance: 20, resultingBalance: -22.5, currency: usd }],
    },
} satisfies Meta<typeof NegativeBalanceConfirmDialog>

export default meta
type Story = StoryObj<typeof meta>

export const OneAccount: Story = {}

// Each account is shown in its own currency's precision.
export const SeveralCurrencies: Story = {
    args: {
        warnings: [
            { accountName: 'Cash', currentBalance: 20, resultingBalance: -22.5, currency: usd },
            { accountName: 'Yen wallet', currentBalance: 1000, resultingBalance: -500, currency: jpy },
            { accountName: 'Kuwait', currentBalance: 1.25, resultingBalance: -0.125, currency: kwd },
        ],
    },
    play: async () => {
        await expect(await screen.findByText(/-500/)).toBeVisible()
    },
}

export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
    },
}

export const Confirmed: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await expect(args.onConfirm).toHaveBeenCalledOnce()
    },
}

export const Cancelled: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await expect(args.onConfirm).not.toHaveBeenCalled()
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
