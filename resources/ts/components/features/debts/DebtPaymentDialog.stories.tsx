import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { account, accounts, currencies, debt } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtPaymentDialog } from './DebtPaymentDialog'

const meta = {
    title: 'features/debts/DebtPaymentDialog',
    component: DebtPaymentDialog,
    parameters: { a11y: overlayA11y },
    args: { open: true, onOpenChange: fn(), onSubmit: fn(), debt: debt(), mode: 'payment' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts))
    },
} satisfies Meta<typeof DebtPaymentDialog>

export default meta
type Story = StoryObj<typeof meta>

const pickAccount = async (dialog: HTMLElement, name: RegExp) => {
    await userEvent.click(within(dialog).getByRole('combobox'))
    await userEvent.click(await screen.findByRole('option', { name }))
}

export const MakePayment: Story = {}
export const CollectPayment: Story = { args: { mode: 'collection', debt: debt({ debtType: 'owed_to_me' }) } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutDebt: Story = { args: { debt: null } }
export const FullAmountButton: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /full/i }))
        await expect(within(dialog).getByRole('spinbutton')).toHaveValue(400)
    },
}

export const HalfAmountButton: Story = {
    args: { debt: debt({ currentBalance: 125.5 }) },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /half/i }))
        await expect(within(dialog).getByRole('spinbutton')).toHaveValue(62.75)
    },
}

export const RejectsEmptyPayment: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /payment/i }))
        await waitFor(() => expect(within(dialog).getAllByText(/select|greater|required/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const SubmitsPayment: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await pickAccount(dialog, /Euro card/)
        await userEvent.click(within(dialog).getByRole('button', { name: /full/i }))
        await userEvent.click(within(dialog).getByRole('button', { name: /^make payment$|^payment$/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect(args.onSubmit).toHaveBeenCalledWith(20, expect.objectContaining({ account_id: 2, amount: 400 }))
    },
}

// Paying more than the account holds asks first; the payment goes through only once confirmed.
export const OverdrawingAsksFirst: Story = {
    args: { debt: debt({ currentBalance: 500 }) },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await pickAccount(dialog, /Cash/)
        await userEvent.click(within(dialog).getByRole('button', { name: /full/i }))
        await userEvent.click(within(dialog).getByRole('button', { name: /^make payment$|^payment$/i }))
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}

export const OverdrawingCanBeCancelled: Story = {
    args: { debt: debt({ currentBalance: 500 }) },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await pickAccount(dialog, /Cash/)
        await userEvent.click(within(dialog).getByRole('button', { name: /full/i }))
        await userEvent.click(within(dialog).getByRole('button', { name: /^make payment$|^payment$/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// Receiving money never overdraws an account, so a collection is not questioned.
export const CollectionNeverAsks: Story = {
    args: { mode: 'collection', debt: debt({ debtType: 'owed_to_me', currentBalance: 500 }) },
    beforeEach: ({ msw }) => msw.use(reply('get', '/accounts', [account({ currentBalance: 1 })])),
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await pickAccount(dialog, /Cash/)
        await userEvent.click(within(dialog).getByRole('button', { name: /full/i }))
        await userEvent.click(within(dialog).getByRole('button', { name: /collect/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect(screen.queryByRole('alertdialog')).toBeNull()
    },
}
