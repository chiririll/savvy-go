import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { accounts } from '@/test/fixtures'
import { reconciliationDay, reconciliationPoint } from '@/test/reconciliation'

import { ReconciliationDayList } from './ReconciliationDayList'

const days = [
    reconciliationDay({
        date: '2020-05-04',
        differenceBase: -9.5,
        points: [
            reconciliationPoint({ id: 1, accountId: 1, difference: -9.5, differenceBase: -9.5 }),
            reconciliationPoint({ id: 2, accountId: 2, difference: 0, differenceBase: 0, actualBalance: 980.1, expectedBalance: 980.1 }),
        ],
    }),
    reconciliationDay({ date: '2020-04-30', differenceBase: 0, points: [reconciliationPoint({ id: 3, difference: 0, differenceBase: 0 })] }),
]

const meta = {
    title: 'features/accounts/ReconciliationDayList',
    component: ReconciliationDayList,
    args: { days, accounts, baseCurrency: 'USD' },
} satisfies Meta<typeof ReconciliationDayList>

export default meta
type Story = StoryObj<typeof meta>

// Days are folded: the sum of a day shows, the accounts appear on opening it.
export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getAllByRole('button')).toHaveLength(2)
        await expect(canvas.queryByText('Cash')).toBeNull()
        await userEvent.click(canvas.getAllByRole('button')[0])
        await expect(await canvas.findByText('Cash')).toBeVisible()
        await expect(canvas.getByText('Euro card')).toBeVisible()
    },
}

export const Loading: Story = { args: { days: [], isLoading: true } }
export const Empty: Story = { args: { days: [] } }
