import type { Meta, StoryObj } from '@storybook/react-vite'
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Table, TableBody } from '@/components/ui/table'

import { SortableRows } from './SortableRows'
import { SortableTableRow } from './SortableTableRow'

type Item = { id: number; name: string }

const items: Item[] = [
    { id: 1, name: 'Cash' },
    { id: 2, name: 'Card' },
    { id: 3, name: 'Savings' },
]

const List = ({ data, onReorder }: { data: Item[]; onReorder: (items: Item[]) => void }) => {
    const table = useReactTable({
        data,
        columns: [{ accessorKey: 'name', header: 'Name' }],
        getCoreRowModel: getCoreRowModel(),
        getRowId: (row) => String(row.id),
    })
    return (
        <Table>
            <TableBody>
                <SortableRows data={data} onReorder={onReorder}>
                    {table.getRowModel().rows.map((row) => (
                        <SortableTableRow key={row.id} row={row} />
                    ))}
                </SortableRows>
            </TableBody>
        </Table>
    )
}

const meta = {
    title: 'shared/table/SortableRows',
    component: List,
    args: { data: items, onReorder: fn() },
} satisfies Meta<typeof List>

export default meta
type Story = StoryObj<typeof meta>

const handles = (canvasElement: HTMLElement) => within(canvasElement).getAllByRole('button', { name: /reorder|drag/i })

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(handles(canvasElement)).toHaveLength(3)
        await expect(within(canvasElement).getByText('Savings')).toBeVisible()
    },
}

// Space picks a row up, the arrows move it, Space drops it.
export const KeyboardMovesARowDown: Story = {
    play: async ({ canvasElement, args }) => {
        handles(canvasElement)[0].focus()
        await userEvent.keyboard(' ')
        await userEvent.keyboard('{ArrowDown}')
        await userEvent.keyboard(' ')
        await expect(args.onReorder).toHaveBeenCalledTimes(1)
        const reordered = (args.onReorder as ReturnType<typeof fn>).mock.calls[0][0] as Item[]
        await expect(reordered.map((item) => item.id)).toEqual([2, 1, 3])
    },
}

export const DroppedInPlaceChangesNothing: Story = {
    play: async ({ canvasElement, args }) => {
        handles(canvasElement)[1].focus()
        await userEvent.keyboard(' ')
        await userEvent.keyboard(' ')
        await expect(args.onReorder).not.toHaveBeenCalled()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const SingleRow: Story = { args: { data: [items[0]] } }
