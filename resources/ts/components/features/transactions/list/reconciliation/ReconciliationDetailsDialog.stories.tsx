import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, currencies } from '@/test/fixtures'
import { reconciliationDay, reconciliationPoint } from '@/test/reconciliation'
import { useCurrenciesStore } from '@/stores/currencies'

import { ReconciliationDetailsDialog } from './ReconciliationDetailsDialog'

const day = reconciliationDay({
    differenceBase: 4.5,
    points: [
        reconciliationPoint({ id: 1, accountId: 1, actualBalance: 130, expectedBalance: 120.5, difference: 9.5, note: 'Found a receipt' }),
        reconciliationPoint({ id: 2, accountId: 2, actualBalance: 900, expectedBalance: 980.1, difference: -80.1, differenceBase: -100.13 }),
        reconciliationPoint({ id: 3, accountId: 3, actualBalance: 15000, expectedBalance: 15000, difference: 0, differenceBase: 0 }),
    ],
})

const meta = {
    title: 'features/transactions/list/ReconciliationDetailsDialog',
    component: ReconciliationDetailsDialog,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { day, accounts, dateLabel: 'May 4, 2020', onOpenChange: fn(), onDelete: fn() },
} satisfies Meta<typeof ReconciliationDetailsDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByText('Reconciliation on May 4, 2020')).toBeVisible()
        for (const name of ['Cash', 'Euro card', 'Yen wallet']) await expect(within(dialog).getByText(name)).toBeVisible()
        // The exact point (Yen wallet) is one compact line, without the Actual and Expected rows.
        await expect(within(dialog).getAllByText('Actual')).toHaveLength(2)
        await expect(within(dialog).getAllByText('Expected')).toHaveLength(2)
        await expect(within(dialog).getByText('Found a receipt')).toBeVisible()
        const positive = within(dialog).getByText(/\+.*9\.50/)
        await expect(positive).toHaveClass('text-success')
        await expect(within(dialog).getByText(/80\.10/, { selector: 'dd.text-destructive' })).toBeVisible()
    },
}

export const Closed: Story = {
    args: { day: null },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

// Without the account list the points fall back to the id.
export const UnknownAccounts: Story = {
    args: { accounts: [] },
    play: async () => {
        await expect(await screen.findByText('#1')).toBeVisible()
    },
}

export const DeleteAsksFirst: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getAllByRole('button', { name: /delete point/i })[1])
        const confirm = await screen.findByRole('alertdialog')
        await expect(within(confirm).getByText(/balance of euro card will no longer be pinned on may 4, 2020/i)).toBeVisible()
        await expect(args.onDelete).not.toHaveBeenCalled()
        await userEvent.click(within(confirm).getByRole('button', { name: /^delete$/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(2)
    },
}

export const DeleteCancelled: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getAllByRole('button', { name: /delete point/i })[0])
        const confirm = await screen.findByRole('alertdialog')
        await userEvent.click(within(confirm).getByRole('button', { name: /cancel/i }))
        await expect(args.onDelete).not.toHaveBeenCalled()
    },
}

export const ReadOnly: Story = {
    args: { onDelete: undefined },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByText('Cash')).toBeVisible()
        await expect(within(dialog).queryByRole('button', { name: /delete point/i })).toBeNull()
    },
}

// An exact point is one line: name, actual total, a tick and the trash button. A point with a gap keeps its block.
export const ExactPointIsCompact: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const ticks = within(dialog).getAllByRole('img', { name: /matches the ledger/i })
        await expect(ticks).toHaveLength(1)
        const compact = ticks[0].closest('li') as HTMLElement
        await expect(within(compact).getByText('Yen wallet')).toBeVisible()
        await expect(within(compact).queryByText('Expected')).toBeNull()
        await expect(within(compact).queryByText('Difference')).toBeNull()
        const gap = within(dialog).getByText('Euro card').closest('li') as HTMLElement
        await expect(within(gap).getByText('Expected')).toBeVisible()
        await expect(within(gap).queryByRole('img', { name: /matches the ledger/i })).toBeNull()

        await userEvent.click(within(compact).getByRole('button', { name: /delete point/i }))
        const confirm = await screen.findByRole('alertdialog')
        await expect(within(confirm).getByText(/balance of yen wallet will no longer be pinned/i)).toBeVisible()
        await expect(args.onDelete).not.toHaveBeenCalled()
        await userEvent.click(within(confirm).getByRole('button', { name: /^delete$/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(3)
    },
}

export const ExactPointWithNote: Story = {
    args: {
        day: reconciliationDay({
            points: [
                reconciliationPoint({
                    id: 4, accountId: 1, actualBalance: 120.5, expectedBalance: 120.5,
                    difference: 0, differenceBase: 0, note: 'Counted the drawer',
                }),
            ],
        }),
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByRole('img', { name: /matches the ledger/i })).toBeVisible()
        await expect(within(dialog).getByText('Counted the drawer')).toBeVisible()
    },
}

export const ReadOnlyWithExactPoint: Story = {
    args: { onDelete: undefined },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByRole('img', { name: /matches the ledger/i })).toBeVisible()
        await expect(within(dialog).queryByRole('button', { name: /delete point/i })).toBeNull()
    },
}
