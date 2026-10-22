import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { PageHeader } from './PageHeader'

const meta = {
    title: 'shared/page/PageHeader',
    component: PageHeader,
    args: { title: 'Accounts', description: 'Where your money lives.' },
} satisfies Meta<typeof PageHeader>

export default meta
type Story = StoryObj<typeof meta>

export const TitleOnly: Story = { args: { description: undefined } }
export const WithDescription: Story = {}
export const WithCreateButton: Story = { args: { onCreateClick: fn() } }
export const WithCreateLink: Story = { args: { createLink: '/accounts/create', createLabel: 'New account' } }
export const WithBackLink: Story = { args: { backLink: '/accounts' } }
export const WithLeadingAndActions: Story = {
    args: { leading: <Badge>Shared</Badge>, actions: <Button variant="outline">Export</Button> },
}
export const LongTitle: Story = {
    args: { title: 'A page title that is much too long for the space it has beside the buttons', onCreateClick: fn() },
}

export const CreateClicked: Story = {
    args: { onCreateClick: fn() },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /create/i }))
        await expect(args.onCreateClick).toHaveBeenCalledOnce()
    },
}
