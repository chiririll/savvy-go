import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { accounts, currencies, debt } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtFormDialog } from './DebtFormDialog'

const meta = {
    title: 'features/debts/DebtFormDialog',
    component: DebtFormDialog,
    parameters: { a11y: overlayA11y },
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/currencies', currencies), reply('get', '/accounts', accounts))
    },
} satisfies Meta<typeof DebtFormDialog>

export default meta
type Story = StoryObj<typeof meta>

// New debts offer to record the money moving through an account.
export const Create: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByRole('checkbox', { name: /transaction/i })).toBeChecked()
    },
}

export const CreateWithoutTransaction: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('checkbox', { name: /transaction/i }))
        await expect(await within(dialog).findByRole('combobox', { name: /currency/i })).toBeVisible()
    },
}

export const Edit: Story = {
    args: { debt: debt() },
    play: async () => {
        await expect(await screen.findByDisplayValue('Loan from Alex')).toBeVisible()
    },
}

export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } }, args: { debt: debt() } }
