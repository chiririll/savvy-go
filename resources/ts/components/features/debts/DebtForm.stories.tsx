import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, currencies } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtForm } from './DebtForm'

const meta = {
    title: 'features/debts/DebtForm',
    component: DebtForm,
    parameters: { a11y: overlayA11y },
    args: { onSubmit: fn(), formId: 'debt-form' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/currencies', currencies), reply('get', '/accounts', accounts))
    },
} satisfies Meta<typeof DebtForm>

export default meta
type Story = StoryObj<typeof meta>

const filled = { name: 'Loan from Alex', debt_type: 'i_owe' as const, amount: 500, account_id: 1, counterparty: 'Alex' }

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: filled } }
export const OwedToMe: Story = { args: { defaultValues: { ...filled, debt_type: 'owed_to_me' } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
// Editing has no account (no transaction is made) and shows the currency instead.
export const Editing: Story = {
    args: { mode: 'edit', defaultValues: { ...filled, currency_id: 2 } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('combobox', { name: /currency/i })).toBeVisible())
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /name/i })).toBeDisabled())
    },
}

// Lending more than the chosen account holds is flagged while typing.
export const InsufficientFunds: Story = {
    args: { defaultValues: { ...filled, debt_type: 'owed_to_me', amount: 500, account_id: 1 } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/insufficient|not enough|available/i)).toBeVisible()
    },
}

export const EnoughFunds: Story = {
    args: { defaultValues: { ...filled, debt_type: 'owed_to_me', amount: 50, account_id: 1 } },
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(within(canvasElement).queryByText(/insufficient|not enough/i)).toBeNull()
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save|create/i }))
        await waitFor(() => expect(within(canvasElement).getAllByText(/required|greater|select/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const Submits: Story = {
    args: { defaultValues: { ...filled, amount: 50 } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save|create/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            name: 'Loan from Alex',
            debt_type: 'i_owe',
            amount: 50,
        })
    },
}

// Lending more than the account holds asks before it goes through.
export const OverdrawingAsksFirst: Story = {
    args: { defaultValues: { ...filled, debt_type: 'owed_to_me', amount: 500, account_id: 1 } },
    play: async ({ canvasElement, args }) => {
        // The accounts are fetched: the form knows the balance once it warns about it.
        await within(canvasElement).findByText(/insufficient|not enough|available/i)
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save|create/i }))
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}

export const TypeSwitched: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const buttons = canvas.getAllByRole('button').slice(0, 2)
        await userEvent.click(buttons[1])
        await expect(canvas.getByRole('textbox', { name: /name/i })).toBeVisible()
    },
}
