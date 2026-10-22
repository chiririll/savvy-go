import type { Meta, StoryObj } from '@storybook/react-vite'
import type { ColumnDef } from '@tanstack/react-table'
import { expect, screen, userEvent, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { DataTable } from './DataTable'

type Row = { id: number; name: string; amount: string }

const columns: ColumnDef<Row>[] = [
    { accessorKey: 'name', header: 'Name' },
    { accessorKey: 'amount', header: 'Amount' },
]

const rows = (count: number): Row[] =>
    Array.from({ length: count }, (_, i) => ({ id: i + 1, name: `Item ${i + 1}`, amount: `${(i + 1) * 10}.00` }))

const meta = {
    title: 'shared/table/DataTable',
    component: DataTable<Row>,
    args: { data: rows(3), columns },
} satisfies Meta<typeof DataTable<Row>>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}

export const Loading: Story = { args: { isLoading: true } }

export const Empty: Story = {
    args: { data: [] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('table')).toBeNull()
    },
}

export const EmptyWithAction: Story = {
    args: {
        data: [],
        emptyTitle: 'No accounts',
        emptyDescription: 'Create the first one.',
        emptyAction: <Button>Create account</Button>,
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: 'Create account' })).toBeVisible()
    },
}

export const Paginated: Story = {
    args: { data: rows(35) },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('Item 1')).toBeVisible()
        await expect(canvas.queryByText('Item 35')).toBeNull()
    },
}

export const ServerPaginated: Story = { args: { data: rows(35), manualPagination: true } }

export const ExpandableRows: Story = {
    args: {
        getRowCanExpand: () => true,
        renderSubComponent: ({ row }) => <p>Details of {row.original.name}</p>,
    },
}

export const RowClassName: Story = {
    args: { getRowClassName: (row) => (row.original.id === 2 ? 'bg-muted' : undefined) },
}

export const Reorderable: Story = { args: { onReorder: () => {} } }

export const ReorderStaysUnpaginated: Story = {
    args: { data: rows(35), onReorder: () => {} },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Item 35')).toBeVisible()
    },
}

export const NarrowScreenScrolls: Story = {
    args: {
        columns: [
            ...columns,
            { accessorKey: 'name', id: 'wide', header: 'A wide extra column', cell: () => 'x'.repeat(60) },
        ],
    },
    play: async () => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
        await userEvent.tab()
    },
}
