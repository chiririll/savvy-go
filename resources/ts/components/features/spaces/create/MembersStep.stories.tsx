import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'

import type { DraftInvitation } from './draft'
import { MembersStep } from './MembersStep'

const listed: DraftInvitation[] = [
    { key: 'i1', email: 'alex@example.com', note: null, role: 'editor' },
    { key: 'i2', email: null, note: 'For grandma', role: 'viewer' },
]

const Harness = ({ initial = [] as DraftInvitation[] }) => {
    const [invitations, setInvitations] = useState(initial)
    return <MembersStep invitations={invitations} onChange={setInvitations} />
}

const meta = {
    title: 'features/spaces/create/MembersStep',
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => msw.use(reply('get', '/invitation-methods', { email: true, link: true })),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const emailField = (canvasElement: HTMLElement) => within(canvasElement).findByRole('textbox', { name: /email/i })

export const NoOneYet: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/no one added yet/i)).toBeVisible()
    },
}

export const Listed: Story = {
    render: () => <Harness initial={listed} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/editor · by email/i)).toBeVisible()
        await expect(canvas.getByText(/viewer · link to copy/i)).toBeVisible()
    },
}

// Nothing is sent from here: the list only grows.
export const Added: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        await userEvent.type(await emailField(canvasElement), 'sam@example.com{Enter}')
        await expect(await within(canvasElement).findByText('sam@example.com')).toBeVisible()
        // A new address starts as an editor.
        await expect(within(canvasElement).getByText(/editor · by email/i)).toBeVisible()
    },
}

export const InvalidAddress: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        const field = await emailField(canvasElement)
        await userEvent.type(field, 'not-an-address{Enter}')
        await expect(within(canvasElement).getByText(/enter a valid email/i)).toBeVisible()
        await expect(field).toHaveValue('not-an-address')
        // Typing again clears the complaint.
        await userEvent.type(field, '@example.com')
        await expect(within(canvasElement).queryByText(/enter a valid email/i)).toBeNull()
    },
}

export const DuplicateAddress: Story = {
    render: () => <Harness initial={listed} />,
    play: async ({ canvasElement }) => {
        await userEvent.type(await emailField(canvasElement), 'ALEX@example.com{Enter}')
        await expect(within(canvasElement).getByText(/already in the list/i)).toBeVisible()
    },
}

// Links have no address, so several can be listed.
export const SeveralLinks: Story = {
    render: () => <Harness initial={listed} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /copy a link/i }))
        await userEvent.click(canvas.getByRole('button', { name: /^add$/i }))
        await expect(canvas.getAllByText(/link to copy/i)).toHaveLength(2)
    },
}

export const Removed: Story = {
    render: () => <Harness initial={listed} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /remove anyone with the link/i }))
        await expect(canvas.queryByText('For grandma')).toBeNull()
        await expect(canvas.getByText('alex@example.com')).toBeVisible()
    },
}
