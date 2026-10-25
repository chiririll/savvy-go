import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { currencyApi } from '@/test/api-mocks'

import { CurrencyForm } from './CurrencyForm'

const meta = {
    title: 'features/currencies/CurrencyForm',
    component: CurrencyForm,
    args: { onSubmit: fn(), formId: 'currency-form' },
    beforeEach: ({ msw }) => msw.use(...currencyApi()),
} satisfies Meta<typeof CurrencyForm>

export default meta
type Story = StoryObj<typeof meta>

const euro = { code: 'EUR', name: 'Euro', symbol: '€', decimals: 2, rate: 0.92 }

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: euro } }
export const Editing: Story = { args: { isEditing: true, defaultValues: euro } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
// The base currency is the unit of every other rate, so its own is fixed at 1.
export const BaseCurrency: Story = {
    args: { isEditing: true, isBase: true, defaultValues: { ...euro, code: 'USD', rate: 1 } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('spinbutton', { name: /rate/i })).toBeDisabled()
    },
}

export const RatesUpdateByThemselves: Story = {
    args: { isEditing: true, autoUpdateEnabled: true, defaultValues: euro },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('spinbutton', { name: /rate/i })).toBeDisabled()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: euro },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /code/i })).toBeDisabled())
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await expect((await within(canvasElement).findAllByText(/required|at least/i)).length).toBeGreaterThan(0)
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const CodeIsUpperCased: Story = {
    play: async ({ canvasElement }) => {
        const code = within(canvasElement).getByRole('textbox', { name: /code/i })
        await userEvent.type(code, 'eu')
        await expect(code).toHaveValue('EU')
    },
}

// Picking a suggestion from the catalog fills code, name, symbol, decimals and rate.
export const CatalogSuggestionFillsTheForm: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox', { name: /code/i }), 'GB')
        await userEvent.click(await screen.findByRole('button', { name: /Pound Sterling/ }))
        await expect(canvas.getByRole('textbox', { name: /^name/i })).toHaveValue('Pound Sterling')
        await expect(canvas.getByRole('textbox', { name: /symbol/i })).toHaveValue('£')
        await expect(canvas.getByRole('spinbutton', { name: /rate/i })).toHaveValue(0.79)
    },
}

export const NoSuggestionsWhileEditing: Story = {
    args: { isEditing: true, defaultValues: euro },
    play: async ({ canvasElement }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox', { name: /code/i }), 'X')
        await expect(screen.queryByRole('button', { name: /Pound/ })).toBeNull()
    },
}

export const Submits: Story = {
    args: { defaultValues: euro },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject(euro)
    },
}
