import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { currencyApi } from '@/test/api-mocks'

import { AccountForm } from './AccountForm'

const meta = {
    title: 'features/accounts/AccountForm',
    component: AccountForm,
    args: { onSubmit: fn(), formId: 'account-form' },
    beforeEach: ({ msw }) => msw.use(...currencyApi()),
} satisfies Meta<typeof AccountForm>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}

export const Filled: Story = {
    args: { defaultValues: { name: 'Euro card', type: 'bank', currency: 'id:2', initial_balance: 250, is_active: true } },
}

// The currency of an existing account cannot change.
export const Editing: Story = {
    args: { isEdit: true, defaultValues: { name: 'Euro card', type: 'bank', currency: 'id:2' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('combobox', { name: /currency/i })).toBeDisabled())
    },
}

export const Submitting: Story = {
    args: { isSubmitting: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /saving/i })).toBeDisabled()
    },
}

export const CustomSubmitLabel: Story = { args: { submitLabel: 'Add account' } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: { name: 'Euro card', currency: 'id:2' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox')).toBeDisabled())
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await expect(await within(canvasElement).findAllByText(/required|select/i)).not.toHaveLength(0)
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const RejectsNegativeBalance: Story = {
    args: { defaultValues: { name: 'X', currency: 'id:1', initial_balance: -5 } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        // The browser's own min=0 check stops it before the schema is asked.
        await expect(within(canvasElement).getByRole('spinbutton')).toBeInvalid()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const SubmitsExistingCurrency: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox', { name: /name/i }), 'Wallet')
        // The picker stays shut until the currencies are in.
        await waitFor(() => expect(canvas.getByRole('combobox', { name: /currency/i })).toBeEnabled())
        await userEvent.click(canvas.getByRole('combobox', { name: /currency/i }))
        await userEvent.click(await screen.findByText(/Euro/))
        await userEvent.click(canvas.getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect(args.onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({ name: 'Wallet', type: 'bank', currency_id: 2 }),
        )
    },
}

// A currency picked from the catalog is sent by code, to be created with the account.
export const SubmitsCatalogCurrency: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox', { name: /name/i }), 'Swiss')
        await waitFor(() => expect(canvas.getByRole('combobox', { name: /currency/i })).toBeEnabled())
        await userEvent.click(canvas.getByRole('combobox', { name: /currency/i }))
        await userEvent.click(await screen.findByText(/Swiss Franc/))
        await userEvent.click(canvas.getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect(args.onSubmit).toHaveBeenCalledWith(expect.objectContaining({ currency_code: 'CHF' }))
    },
}
