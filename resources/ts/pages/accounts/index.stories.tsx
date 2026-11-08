import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, catalog, currencies } from '@/test/fixtures'
import { hang, raw, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import AccountsPage from './index'

const meta = {
    title: 'pages/Accounts',
    component: AccountsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/currencies', currencies),
            reply('get', '/currencies/catalog', catalog),
        )
    },
} satisfies Meta<typeof AccountsPage>

export default meta
type Story = StoryObj<typeof meta>

// One account per currency precision: none, two and three decimals.
export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Cash')).toBeVisible()
        await expect(canvas.getByText('120.50 $')).toBeVisible()
        await expect(canvas.getByText('15,000 ¥')).toBeVisible()
        await expect(canvas.getByText(/12\.345/)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new account/i })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/accounts')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/accounts', [])),
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByRole('button', { name: /new account/i }))[0]).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new account/i })).toBeNull())
        await expect(await within(canvasElement).findByText('Cash')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new account/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}

export const ReconcileButtonForEditor: Story = {
    parameters: { app: { role: 'editor' } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /reconcile/i })).toBeVisible()
    },
}

export const ReconcileButtonHiddenWhenReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Cash')).toBeVisible()
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /reconcile/i })).toBeNull())
    },
}

// The dialog starts on today, where the balances are the ones the list already has. The history route is mocked
// for the days a user may pick afterwards.
export const OpensTheReconcileDialog: Story = {
    beforeEach: ({ msw }) => {
        msw.use(raw('get', '/accounts-balance-history', { dates: [], series: [], currency: 'USD', decimals: 2 }))
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /reconcile/i }))
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByText('Reconcile accounts')).toBeVisible()
        await expect(await within(dialog).findByRole('textbox', { name: /actual balance of cash/i })).toHaveValue('120.50')
        await expect(within(dialog).getByRole('textbox', { name: /actual balance of kuwait savings/i })).toHaveValue('12.345')
    },
}
