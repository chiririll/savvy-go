import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { unmountedTabsOverlayA11y } from '@/test/a11y'
import { accounts, catalog, currencies, debt } from '@/test/fixtures'
import { hang, raw, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import DebtsPage from './index'

const summary = { totalIOwe: 400, totalOwedToMe: 150.5, netDebt: -249.5, debtsCount: 2, currency: 'USD', decimals: 2 }

const active = [
    debt({ id: 1 }),
    debt({ id: 2, name: 'Loan to Sam', debtType: 'owed_to_me', counterparty: 'Sam', currentBalance: 150.5, paymentProgress: 50 }),
]

const meta = {
    title: 'pages/Debts',
    component: DebtsPage,
    parameters: { a11y: unmountedTabsOverlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            raw('get', '/debts', { data: active, summary }),
            reply('get', '/currencies', currencies),
            reply('get', '/currencies/catalog', catalog),
            reply('get', '/accounts', accounts),
        )
    },
} satisfies Meta<typeof DebtsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Loan from Alex')).toBeVisible()
        await expect(canvas.getByText('Loan to Sam')).toBeVisible()
        // The net position is exact to the cent: 150.50 owed to me against 400 owed by me.
        await expect(canvas.getByText(/249\.50 \$/)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new debt/i })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/debts')) }

export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            raw('get', '/debts', {
                data: active,
                summary: { ...summary, totalIOwe: 60_000, totalOwedToMe: 0, netDebt: -60_000, currency: 'JPY', decimals: 0 },
            }),
        ),
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByText(/60,000 ¥/))[0]).toBeVisible()
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(raw('get', '/debts', { data: [], summary: { ...summary, totalIOwe: 0, totalOwedToMe: 0, netDebt: 0, debtsCount: 0 } })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/nothing|no .* yet|empty/i)).toBeVisible()
    },
}

export const CompletedTab: Story = {
    beforeEach: ({ msw }) =>
        msw.use(raw('get', '/debts', { data: [debt({ id: 3, name: 'Old loan', isPaidOff: true, currentBalance: 0 })], summary })),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('tab', { name: /completed/i }))
        await expect(await canvas.findByText('Old loan')).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new debt/i })).toBeNull())
        await expect(await within(canvasElement).findByText('Loan from Alex')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new debt/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
