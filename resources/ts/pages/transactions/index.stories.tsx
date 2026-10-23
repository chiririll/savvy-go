import type { Meta, StoryObj } from '@storybook/react-vite'
import { http, HttpResponse } from 'msw'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, categories, currencies, pendingTransaction, tags, transaction } from '@/test/fixtures'
import { fail, hang, raw, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'
import { CreateTransactionProvider } from '@/components/features/transactions/form/CreateTransactionProvider'

import TransactionsPage from './index'

const page = (data: unknown[], total = data.length) => ({
    data,
    meta: { current_page: 1, last_page: 1, per_page: 20, total, from: data.length ? 1 : 0, to: data.length },
})

// The page asks twice: for the list, and for what is pending in the coming week.
const list = (rows: unknown[], pending: unknown[] = []) =>
    http.get(/\/api(\/spaces\/\d+)?\/transactions(\?.*)?$/, ({ request }) =>
        HttpResponse.json(page(new URL(request.url).searchParams.get('status') === 'pending' ? pending : rows)),
    )

// One account per currency precision: a yen has no fraction, a dinar has three digits.
const rows = [
    transaction({ id: 1, description: 'Weekly shop', amount: 42.5 }),
    transaction({ id: 2, description: 'Ramen', amount: 1200, account: accounts[2], date: '2020-05-03' }),
    transaction({ id: 3, description: 'Taxi', amount: 1.234, account: accounts[3], date: '2020-05-02' }),
]

const meta = {
    title: 'pages/Transactions',
    component: TransactionsPage,
    parameters: { a11y: overlayA11y },
    // The page opens the transaction dialog the layout provides.
    render: () => (
        <CreateTransactionProvider>
            <TransactionsPage />
        </CreateTransactionProvider>
    ),
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            list(rows),
            reply('get', '/categories', categories),
            reply('get', '/tags', tags),
            reply('get', '/accounts', accounts),
            reply('get', '/currencies', currencies),
        )
    },
} satisfies Meta<typeof TransactionsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Weekly shop')).toBeVisible()
        await expect(canvas.getByText('Ramen')).toBeVisible()
        await expect(canvas.getByText('Taxi')).toBeVisible()
        await expect(canvas.getByText(/1,200 ¥/)).toBeVisible()
        await expect(canvas.getByText(/1\.234/)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new transaction/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/transactions')) }

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(list([])),
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByRole('button', { name: /new transaction/i }))[0]).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/transactions')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /try again/i })).toBeVisible()
    },
}

export const WithUpcomingPending: Story = {
    beforeEach: ({ msw }) => msw.use(list(rows, [pendingTransaction({ id: 9, description: 'Rent due' })])),
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByText('Rent due'))[0]).toBeInTheDocument()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new transaction/i })).toBeNull())
        await expect(await within(canvasElement).findByText('Weekly shop')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new transaction/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
