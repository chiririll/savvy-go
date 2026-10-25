import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { currencies, debt } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtsCard } from './DebtsCard'

const summary = { totalIOwe: 400, totalOwedToMe: 150, netDebt: 250, debtsCount: 2, currency: 'USD', decimals: 2 }

const meta = {
    title: 'features/dashboard/DebtsCard',
    component: DebtsCard,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        summary,
        debts: [
            debt({ id: 1 }),
            debt({ id: 2, name: 'Loan to Sam', debtType: 'owed_to_me', counterparty: 'Sam', currentBalance: 150, paymentProgress: 50 }),
        ],
    },
} satisfies Meta<typeof DebtsCard>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Empty: Story = {
    args: { summary: undefined, debts: [] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /add/i })).toHaveAttribute('href', '/debts?create=1')
    },
}
export const NothingOwedEitherWay: Story = {
    args: { summary: { ...summary, totalIOwe: 0, totalOwedToMe: 0, netDebt: 0, debtsCount: 0 }, debts: [] },
}
export const NetPositive: Story = { args: { summary: { ...summary, totalIOwe: 100, totalOwedToMe: 400, netDebt: 300 } } }
export const NetNegative: Story = { args: { summary: { ...summary, totalIOwe: 400, totalOwedToMe: 100, netDebt: -300 } } }
export const SummaryWithoutTiles: Story = { args: { debts: [] } }
export const NoCounterparty: Story = { args: { debts: [debt({ counterparty: null })] } }
export const LongNames: Story = {
    args: {
        debts: [debt({ name: 'A debt with a name far too long for its tile', counterparty: 'A counterparty with a very long name too' })],
    },
}
export const InYen: Story = {
    args: {
        summary: { ...summary, totalIOwe: 450000, totalOwedToMe: 0, netDebt: -450000, currency: 'JPY', decimals: 0 },
        debts: [debt({ currency: currencies[2], currentBalance: 450000, targetAmount: 500000 })],
    },
}
