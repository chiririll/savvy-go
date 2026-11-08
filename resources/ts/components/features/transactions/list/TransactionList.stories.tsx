import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { accounts, currencies, pendingTransaction, transaction } from '@/test/fixtures'
import { reconciliationDay, reconciliationPoint } from '@/test/reconciliation'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionList } from './TransactionList'

const transactions = [
    transaction({ id: 1, description: 'Coffee', date: '2020-05-04', amount: 3.5 }),
    transaction({ id: 2, description: 'Bus ticket', date: '2020-05-04', amount: 2 }),
    transaction({ id: 3, description: 'Rent', date: '2020-05-01', amount: 1100 }),
    transaction({ id: 4, description: 'Gift', date: null, amount: 30 }),
]

const meta = {
    title: 'features/transactions/list/TransactionList',
    component: TransactionList,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        transactions,
        emptyTitle: 'No transactions',
        emptyDescription: 'Add one to see it here.',
        onDelete: fn(),
        onDuplicate: fn(),
        onEdit: fn(),
    },
} satisfies Meta<typeof TransactionList>

export default meta
type Story = StoryObj<typeof meta>

export const GroupedByDay: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByRole('heading', { level: 2 }).length).toBeGreaterThanOrEqual(2)
    },
}

// Without groups every row carries its own date.
export const Ungrouped: Story = { args: { grouped: false } }
export const Loading: Story = { args: { isLoading: true } }
export const Empty: Story = { args: { transactions: [], onCreate: fn(), createLabel: 'Add transaction' } }
export const EmptyReadOnly: Story = {
    args: { transactions: [], onCreate: fn(), createLabel: 'Add transaction', isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}
export const PendingList: Story = {
    args: {
        transactions: [pendingTransaction({ id: 7 }), pendingTransaction({ id: 8, date: '2099-07-01' })],
        onConfirm: fn(),
        onSkip: fn(),
    },
}
export const ReadOnly: Story = { args: { isReadOnly: true } }

// Coffee and Bus ticket share a day: the first can only go down, the second only up.
export const MovedWithMenu: Story = {
    tags: ['desktop-only'],
    args: { onReorder: fn() },
    play: async ({ canvasElement, args }) => {
        const menus = within(canvasElement).getAllByRole('button', { name: /more actions/i })
        await userEvent.click(menus[0])
        await expect(screen.queryByRole('menuitem', { name: /move up/i })).toBeNull()
        await userEvent.click(await screen.findByRole('menuitem', { name: /move down/i }))
        await expect(args.onReorder).toHaveBeenCalledWith('2020-05-04', [2, 1])
    },
}

export const NoMoveWhenReadOnly: Story = {
    tags: ['desktop-only'],
    args: { onReorder: fn(), isReadOnly: true },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /more actions/i })[0])
        await expect(screen.queryByRole('menuitem', { name: /move/i })).toBeNull()
        await userEvent.keyboard('{Escape}')
    },
}

// May 4 has transactions and a point; May 3 has only a point and gets a group of its own, between May 4 and May 1.
const reconciliationDays = [
    reconciliationDay({ date: '2020-05-04', differenceBase: 9.5 }),
    reconciliationDay({
        date: '2020-05-03',
        differenceBase: -20.5,
        points: [
            reconciliationPoint({ id: 2, date: '2020-05-03', actualBalance: 100, expectedBalance: 120.5, difference: -20.5, differenceBase: -20.5 }),
        ],
    }),
]

export const WithReconciliationLines: Story = {
    args: { reconciliationDays, accounts, baseCurrency: 'USD', onDeleteReconciliation: fn() },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const lines = canvas.getAllByText('Reconciled')
        await expect(lines).toHaveLength(2)
        await expect(lines[0]).toHaveClass('text-success')
        await expect(lines[1]).toHaveClass('text-destructive')
        // The day with only a point has no transaction rows, but it has a heading.
        await expect(canvas.getAllByRole('heading', { level: 2 }).length).toBeGreaterThanOrEqual(3)
    },
}

export const ReconciliationDetailsOpen: Story = {
    args: { reconciliationDays, accounts, baseCurrency: 'USD', onDeleteReconciliation: fn() },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /reconciliation details/i })[1])
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByText('Cash')).toBeVisible()
        await userEvent.click(within(dialog).getByRole('button', { name: /delete point/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /^delete$/i }))
        await expect(args.onDeleteReconciliation).toHaveBeenCalledWith(2)
    },
}

export const ReconciliationReadOnly: Story = {
    args: { reconciliationDays, accounts, baseCurrency: 'USD', onDeleteReconciliation: fn(), isReadOnly: true },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /reconciliation details/i })[0])
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).queryByRole('button', { name: /delete point/i })).toBeNull()
    },
}

// The line is the balance at the end of its day: it comes before that day's rows when the list runs newest first,
// and after them when it runs oldest first.
const lineOfMay4 = (canvasElement: HTMLElement) =>
    within(canvasElement)
        .getAllByText('Reconciled')
        .find((node) => node.classList.contains('text-success')) as HTMLElement

export const LineBeforeRowsNewestFirst: Story = {
    args: { reconciliationDays, accounts, baseCurrency: 'USD' },
    play: async ({ canvasElement }) => {
        const line = lineOfMay4(canvasElement)
        const coffee = within(canvasElement).getByText('Coffee')
        const heading = coffee.closest('section')?.querySelector('h2')
        await expect(line.compareDocumentPosition(coffee) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
        // Above the day's heading, not inside its group.
        await expect(heading).not.toBeNull()
        await expect(line.compareDocumentPosition(heading as Element) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
        await expect(line.closest('section')).toBeNull()
    },
}

export const LineAfterRowsOldestFirst: Story = {
    args: { reconciliationDays, accounts, baseCurrency: 'USD', sortDirection: 'asc' },
    play: async ({ canvasElement }) => {
        const line = lineOfMay4(canvasElement)
        const coffee = within(canvasElement).getByText('Coffee')
        const bus = within(canvasElement).getByText('Bus ticket')
        await expect(line.compareDocumentPosition(coffee) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
        await expect(line.compareDocumentPosition(bus) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
    },
}
