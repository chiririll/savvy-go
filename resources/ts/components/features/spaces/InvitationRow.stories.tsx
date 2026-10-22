import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { InvitationList } from './InvitationList'
import { InvitationRow } from './InvitationRow'

// The rows sit in a list, as on every page that shows them.
const meta = {
    title: 'features/spaces/InvitationRow',
    component: InvitationRow,
    parameters: { layout: 'padded' },
    decorators: [(Story) => <div className="max-w-md">{Story()}</div>],
    render: (args) => (
        <InvitationList empty="No invitations">
            <InvitationRow {...args} />
        </InvitationList>
    ),
    args: { email: 'alex@example.com', role: 'editor' },
} satisfies Meta<typeof InvitationRow>

export default meta
type Story = StoryObj<typeof meta>

export const ByEmail: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('alex@example.com')).toBeVisible()
        await expect(within(canvasElement).getByText('Editor')).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const LinkForAnyone: Story = {
    args: { email: null, role: 'viewer' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/anyone with the link/i)).toBeVisible()
    },
}

export const LinkWithItsNote: Story = {
    args: { email: null, label: 'For grandma', role: 'viewer' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('For grandma')).toBeVisible()
    },
}

export const WithNote: Story = {
    args: { note: 'Email sent' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/editor · email sent/i)).toBeVisible()
    },
}

// A problem takes the place of the role line.
export const Refused: Story = {
    args: { problem: 'This address is already a member.' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/already a member/i)).toBeVisible()
        await expect(within(canvasElement).queryByText('Editor')).toBeNull()
    },
}

export const WithActions: Story = {
    args: {
        children: (
            <Button variant="ghost" size="sm">
                Revoke
            </Button>
        ),
    },
}

export const LongAddress: Story = {
    args: {
        email: 'a-very-long-mailbox-name.that-keeps-going-and-going@subdomain.example.com',
        children: <Button size="sm">Copy</Button>,
    },
}

export const EmptyList: Story = {
    render: () => <InvitationList empty="No open invitations." />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('No open invitations.')).toBeVisible()
        await expect(within(canvasElement).queryByRole('list')).toBeNull()
    },
}
