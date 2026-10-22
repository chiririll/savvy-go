import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { TransactionTypeTabs, type TransactionFormKind } from './TransactionTypeTabs'

const meta = {
    title: 'shared/TransactionTypeTabs',
    component: TransactionTypeTabs,
    args: { value: 'expense', onChange: () => {} },
    render: (args) => {
        const [value, setValue] = useState<TransactionFormKind>(args.value)
        return <TransactionTypeTabs {...args} value={value} onChange={setValue} />
    },
} satisfies Meta<typeof TransactionTypeTabs>

export default meta
type Story = StoryObj<typeof meta>

export const Expense: Story = {}
export const Income: Story = { args: { value: 'income' } }
export const Transfer: Story = { args: { value: 'transfer' } }
export const WithSpaceTransfer: Story = { args: { withSpaceTransfer: true } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const SwitchToSpaceTransfer: Story = {
    args: { withSpaceTransfer: true },
    play: async ({ canvasElement }) => {
        const buttons = within(canvasElement).getAllByRole('button')
        await userEvent.click(buttons[buttons.length - 1])
        await expect(buttons[buttons.length - 1]).toBeVisible()
    },
}
