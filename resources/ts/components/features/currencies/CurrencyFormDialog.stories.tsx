import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { currencyApi, settingsApi } from '@/test/api-mocks'

import { CurrencyFormDialog } from './CurrencyFormDialog'

const [usd, eur] = currencies

const meta = {
    title: 'features/currencies/CurrencyFormDialog',
    component: CurrencyFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => msw.use(...currencyApi(), ...settingsApi()),
} satisfies Meta<typeof CurrencyFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Edit: Story = {
    args: { currency: eur },
    play: async () => {
        await expect(await screen.findByDisplayValue('Euro')).toBeVisible()
    },
}
export const EditBase: Story = { args: { currency: usd } }
export const EditWithAutoUpdate: Story = {
    args: { currency: eur },
    beforeEach: ({ msw }) => msw.use(...currencyApi(), ...settingsApi({ auto_update_currencies: true })),
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const Closed: Story = { args: { open: false } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } } }
