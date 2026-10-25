import type { Meta, StoryObj } from '@storybook/react-vite'

import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from './pagination'

const meta = { title: 'ui/Pagination', component: Pagination } satisfies Meta<typeof Pagination>

export default meta
type Story = StoryObj<typeof meta>

export const Middle: Story = {
    render: (args) => (
        <Pagination {...args}>
            <PaginationContent>
                <PaginationItem><PaginationPrevious href="#" /></PaginationItem>
                <PaginationItem><PaginationLink href="#">1</PaginationLink></PaginationItem>
                <PaginationItem><PaginationEllipsis /></PaginationItem>
                <PaginationItem><PaginationLink href="#" isActive>5</PaginationLink></PaginationItem>
                <PaginationItem><PaginationEllipsis /></PaginationItem>
                <PaginationItem><PaginationLink href="#">9</PaginationLink></PaginationItem>
                <PaginationItem><PaginationNext href="#" /></PaginationItem>
            </PaginationContent>
        </Pagination>
    ),
}

export const SinglePage: Story = {
    render: (args) => (
        <Pagination {...args}>
            <PaginationContent>
                <PaginationItem><PaginationLink href="#" isActive>1</PaginationLink></PaginationItem>
            </PaginationContent>
        </Pagination>
    ),
}
