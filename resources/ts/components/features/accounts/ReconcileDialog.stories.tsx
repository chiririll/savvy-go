import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fireEvent, fn, screen, userEvent, waitFor } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { raw, spy } from '@/test/api-mocks'
import { account, accounts, currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { ReconcileDialog } from './ReconcileDialog'

// Debt and space accounts are kept by the app: they never get a row.
const withServiceAccounts = [
    ...accounts,
    account({ id: 5, name: 'Loan to Sam', type: 'debt', currentBalance: -50 }),
    account({ id: 6, name: 'Holiday space', type: 'space', currentBalance: 10 }),
]

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/accounts/ReconcileDialog',
    component: ReconcileDialog,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { accounts: withServiceAccounts, open: true, onOpenChange: fn() },
} satisfies Meta<typeof ReconcileDialog>

export default meta
type Story = StoryObj<typeof meta>

const amountOf = (name: string) => screen.findByRole('textbox', { name: new RegExp(`actual balance of ${name}`, 'i') })

export const Default: Story = {
    play: async () => {
        await expect(await screen.findByRole('dialog')).toBeVisible()
        await expect(await amountOf('Cash')).toHaveValue('120.50')
        await expect(await amountOf('Euro card')).toHaveValue('980.10')
        await expect(await amountOf('Yen wallet')).toHaveValue('15000')
        await expect(await amountOf('Kuwait savings')).toHaveValue('12.345')
        await expect(screen.queryByText('Loan to Sam')).toBeNull()
        await expect(screen.queryByText('Holiday space')).toBeNull()
        await expect(screen.queryByRole('button', { name: /add account/i })).toBeNull()
        await expect(screen.getByRole('button', { name: /^save$/i })).toBeEnabled()
    },
}

export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

export const NoAccounts: Story = {
    args: { accounts: [withServiceAccounts[4]] },
    play: async () => {
        await expect(await screen.findByText(/no accounts to reconcile/i)).toBeVisible()
        await expect(screen.getByRole('button', { name: /^save$/i })).toBeDisabled()
    },
}

export const RemovedRowComesBack: Story = {
    play: async () => {
        await amountOf('Euro card')
        await userEvent.click(screen.getByRole('button', { name: /remove euro card/i }))
        await expect(screen.queryByRole('textbox', { name: /actual balance of euro card/i })).toBeNull()

        await userEvent.click(screen.getByRole('button', { name: /add account/i }))
        await userEvent.click(await screen.findByRole('button', { name: /euro card/i }))

        await expect(await amountOf('Euro card')).toHaveValue('980.10')
        await waitFor(() => expect(screen.queryByRole('button', { name: /add account/i })).toBeNull())
    },
}

export const AllRowsRemoved: Story = {
    play: async () => {
        await amountOf('Cash')
        for (const name of ['Cash', 'Euro card', 'Yen wallet', 'Kuwait savings']) {
            await userEvent.click(screen.getByRole('button', { name: new RegExp(`remove ${name}`, 'i') }))
        }
        await expect(screen.getByText(/no accounts to reconcile/i)).toBeVisible()
        await expect(screen.getByRole('button', { name: /^save$/i })).toBeDisabled()
        await expect(screen.getByRole('button', { name: /add account/i })).toBeVisible()
    },
}

// What is typed against the balance the ledger has: more is green with a plus, less is red.
export const ShowsPositiveAndNegativeDifference: Story = {
    play: async () => {
        const cash = await amountOf('Cash')
        await userEvent.clear(cash)
        await userEvent.type(cash, '130')
        const euro = await amountOf('Euro card')
        await userEvent.clear(euro)
        await userEvent.type(euro, '900')

        const positive = (await screen.findAllByText(/difference:/i)).find((node) => node.textContent?.includes('9.50'))
        await expect(positive).toBeDefined()
        await expect(positive).toHaveClass('text-success')
        await expect(positive).toHaveTextContent(/\+/)

        const negative = screen.getAllByText(/difference:/i).find((node) => node.textContent?.includes('80.10'))
        await expect(negative).toBeDefined()
        await expect(negative).toHaveClass('text-destructive')
        await expect(negative).not.toHaveTextContent('+')

        // The rows left alone stay exact.
        const untouched = screen.getAllByText(/difference:/i).filter((node) => !node.className.match(/text-(success|destructive)/))
        await expect(untouched).toHaveLength(2)
    },
}

export const SavesTheBatch: Story = {
    beforeEach: ({ msw }) => {
        const create = spy('post', '/reconciliations', [])
        sent.bodies = create.bodies
        msw.use(create.handler)
    },
    play: async ({ args }) => {
        const cash = await amountOf('Cash')
        await userEvent.clear(cash)
        await userEvent.type(cash, '130.25')
        await userEvent.click(screen.getByRole('button', { name: /remove kuwait savings/i }))
        await userEvent.click(screen.getByRole('button', { name: /^save$/i }))

        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(sent.bodies[0]).toMatchObject({
            items: [
                { accountId: 1, actualBalance: 130.25 },
                { accountId: 2, actualBalance: 980.1 },
                { accountId: 3, actualBalance: 15000 },
            ],
        })
        await expect((sent.bodies[0] as { items: unknown[] }).items).toHaveLength(3)
        await waitFor(() => expect(args.onOpenChange).toHaveBeenCalledWith(false))
    },
}

// A past date asks the ledger for the balances at its end; untouched rows take them, so their difference stays zero.
export const PastDateTakesHistoricalBalances: Story = {
    beforeEach: ({ msw }) => {
        const series = (id: number, name: string, native: number, currency: string) => ({
            id,
            name,
            type: 'cash',
            data: [native],
            native_data: [native],
            currency,
        })
        msw.use(
            raw('get', '/accounts-balance-history', {
                dates: ['2020-01-15'],
                series: [
                    series(1, 'Cash', 100, 'USD'),
                    series(2, 'Euro card', 800.5, 'EUR'),
                    series(3, 'Yen wallet', 12000, 'JPY'),
                    series(4, 'Kuwait savings', 5.125, 'KWD'),
                ],
                currency: 'USD',
                decimals: 2,
            }),
        )
    },
    play: async () => {
        await amountOf('Cash')
        fireEvent.change(screen.getByLabelText(/^date$/i), { target: { value: '2020-01-15' } })
        await waitFor(() => expect(screen.getByRole('textbox', { name: /actual balance of cash/i })).toHaveValue('100.00'))
        await expect(screen.getByRole('textbox', { name: /actual balance of euro card/i })).toHaveValue('800.50')
        await expect(screen.getByRole('textbox', { name: /actual balance of yen wallet/i })).toHaveValue('12000')
        await expect(screen.getByRole('textbox', { name: /actual balance of kuwait savings/i })).toHaveValue('5.125')
    },
}
