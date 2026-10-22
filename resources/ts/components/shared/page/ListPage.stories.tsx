import type { Meta, StoryObj } from '@storybook/react-vite'
import type { ColumnDef } from '@tanstack/react-table'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { ListPage } from './ListPage'

type Row = { id: number; name: string; balance: string }

const columns: ColumnDef<Row>[] = [
    { accessorKey: 'name', header: 'Name' },
    { accessorKey: 'balance', header: 'Balance' },
]

const data: Row[] = [
    { id: 1, name: 'Cash', balance: '120.50 $' },
    { id: 2, name: 'Euro card', balance: '980.10 €' },
    { id: 3, name: 'Yen wallet', balance: '15,000 ¥' },
]

const meta = {
    title: 'shared/page/ListPage',
    component: ListPage<Row>,
    args: { title: 'Accounts', description: 'Where your money lives.', data, columns, onCreateClick: fn() },
} satisfies Meta<typeof ListPage<Row>>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Loading: Story = { args: { isLoading: true } }

export const EmptyWithCreate: Story = {
    args: { data: [], emptyTitle: 'No accounts', emptyDescription: 'Create the first one.' },
    play: async ({ canvasElement, args }) => {
        const buttons = within(canvasElement).getAllByRole('button', { name: /create/i })
        await userEvent.click(buttons[buttons.length - 1])
        await expect(args.onCreateClick).toHaveBeenCalled()
    },
}

export const EmptyWithLink: Story = { args: { data: [], onCreateClick: undefined, createLink: '/accounts/new' } }
export const WithToolbar: Story = { args: { toolbar: <Button variant="outline">Filter</Button> } }
export const WithoutCreate: Story = { args: { onCreateClick: undefined } }
export const CustomCreateLabel: Story = { args: { createLabel: 'New account' } }
export const Reorderable: Story = { args: { onReorder: fn() } }
export const HighlightedRow: Story = { args: { getRowClassName: (row) => (row.original.id === 2 ? 'bg-muted' : undefined) } }
