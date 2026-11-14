import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { accounts, currencies } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { reconciliationDay, reconciliationPoint } from '@/test/reconciliation'
import { useCurrenciesStore } from '@/stores/currencies'

import ReconciliationsPage from './index'

const meta = {
    title: 'pages/Reconciliations',
    component: ReconciliationsPage,
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts), reply('get', '/currencies', currencies))
    },
} satisfies Meta<typeof ReconciliationsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/reconciliations-days', [
                reconciliationDay(),
                reconciliationDay({ date: '2020-04-30', points: [reconciliationPoint({ id: 2 })] }),
            ]),
        ),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/4 May 2020|May 4, 2020/)).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reconciliations-days')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reconciliations-days', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No reconciliations yet')).toBeVisible()
    },
}
