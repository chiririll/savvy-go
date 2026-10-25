import type { Meta, StoryObj } from '@storybook/react-vite'

import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from './table'

const meta = { title: 'ui/Table', component: Table } satisfies Meta<typeof Table>

export default meta
type Story = StoryObj<typeof meta>

const rows = [
    ['Groceries', '−42.50 USD'],
    ['Salary', '+3,200.00 USD'],
    ['Rent', '−1,100 JPY'],
]

export const Filled: Story = {
    render: (args) => (
        <Table {...args}>
            <TableCaption>Latest transactions</TableCaption>
            <TableHeader>
                <TableRow><TableHead>Name</TableHead><TableHead className="text-right">Amount</TableHead></TableRow>
            </TableHeader>
            <TableBody>
                {rows.map(([name, amount]) => (
                    <TableRow key={name}><TableCell>{name}</TableCell><TableCell className="text-right">{amount}</TableCell></TableRow>
                ))}
            </TableBody>
        </Table>
    ),
}

export const Empty: Story = {
    render: (args) => (
        <Table {...args}>
            <TableHeader>
                <TableRow><TableHead>Name</TableHead><TableHead>Amount</TableHead></TableRow>
            </TableHeader>
            <TableBody>
                <TableRow><TableCell colSpan={2} className="text-center">No rows</TableCell></TableRow>
            </TableBody>
        </Table>
    ),
}
