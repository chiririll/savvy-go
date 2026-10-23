import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, categories, currencies, recurring, tags } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import RecurringPage from './index'

const meta = {
    title: 'pages/Recurring',
    component: RecurringPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/recurring', [recurring(), recurring({ id: 2, description: 'Gym', amount: 35.5, isActive: false })]),
            reply('get', '/currencies', currencies),
            reply('get', '/categories', categories),
            reply('get', '/tags', tags),
            reply('get', '/accounts', accounts),
        )
    },
} satisfies Meta<typeof RecurringPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Rent')).toBeVisible()
        await expect(canvas.getByText('Gym')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new recurring/i })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/recurring')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/recurring', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/nothing|no .* yet|empty/i)).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(async () => expect(await within(canvasElement).findByText('Rent')).toBeVisible())
        // The role comes from the space list, which can land after the rows.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new recurring/i })).toBeNull())
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new recurring/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
