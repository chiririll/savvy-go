import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, budget, categories, currencies, tags } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import BudgetsPage from './index'

const meta = {
    title: 'pages/Budgets',
    component: BudgetsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/budgets', [budget(), budget({ id: 2, name: 'Fun', amount: 90.5 })]),
            reply('get', '/currencies', currencies),
            reply('get', '/categories', categories),
            reply('get', '/tags', tags),
            reply('get', '/accounts', accounts),
        )
    },
} satisfies Meta<typeof BudgetsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect((await canvas.findAllByText('Groceries'))[0]).toBeVisible()
        await expect(canvas.getByText('Fun')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new budget/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/budgets')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/budgets', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/nothing|no .* yet|empty/i)).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByText('Groceries'))[0]).toBeVisible()
        // The role comes from the space list, which can land after the rows.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new budget/i })).toBeNull())
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new budget/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
