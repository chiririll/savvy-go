import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { currencies, debt } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtPaymentSummary } from './DebtPaymentSummary'

const meta = {
    title: 'features/debts/DebtPaymentSummary',
    component: DebtPaymentSummary,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { debt: debt(), mode: 'payment' },
} satisfies Meta<typeof DebtPaymentSummary>

export default meta
type Story = StoryObj<typeof meta>

export const Payment: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Alex')).toBeVisible()
        await expect(within(canvasElement).getByText(/pay to/i)).toBeVisible()
    },
}

export const Collection: Story = {
    args: { mode: 'collection' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/receive from/i)).toBeVisible()
    },
}

export const WithoutCounterparty: Story = {
    args: { debt: debt({ counterparty: null }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/pay to|receive from/i)).toBeNull()
    },
}

export const InYen: Story = { args: { debt: debt({ currency: currencies[2], currentBalance: 45000 }) } }
export const InRussian: Story = { globals: { locale: 'ru' } }
