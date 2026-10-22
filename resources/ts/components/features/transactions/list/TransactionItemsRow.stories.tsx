import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { accounts, currencies, transaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionItemsRow } from './TransactionItemsRow'

const meta = {
    title: 'features/transactions/list/TransactionItemsRow',
    component: TransactionItemsRow,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
} satisfies Meta<typeof TransactionItemsRow>

export default meta
type Story = StoryObj<typeof meta>

const items = [
    { id: 1, name: 'Bread', quantity: 2, pricePerUnit: 2.5, totalPrice: 5 },
    { id: 2, name: 'Cheese', quantity: 1, pricePerUnit: 12.5, totalPrice: 12.5 },
    { name: 'Unsaved item', quantity: 3, pricePerUnit: 1, totalPrice: 3 },
]

export const Items: Story = {
    args: { transaction: transaction({ items }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByRole('row')).toHaveLength(items.length + 1)
    },
}

// Prices follow the account's currency: no decimals for yen, three for dinars.
export const InYen: Story = {
    args: { transaction: transaction({ account: accounts[2], items: [{ id: 1, name: 'Ramen', quantity: 2, pricePerUnit: 900, totalPrice: 1800 }] }) },
}
export const InDinars: Story = {
    args: { transaction: transaction({ account: accounts[3], items: [{ id: 1, name: 'Tea', quantity: 3, pricePerUnit: 1.125, totalPrice: 3.375 }] }) },
}

export const NoItems: Story = {
    args: { transaction: transaction({ items: [] }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('table')).toBeNull()
    },
}
