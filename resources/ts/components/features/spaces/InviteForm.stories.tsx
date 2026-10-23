import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { fail, reply, spy } from '@/test/api-mocks'

import { InviteForm } from './InviteForm'

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/InviteForm',
    component: InviteForm,
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => msw.use(reply('get', '/invitation-methods', { email: true, link: true })),
    args: { spaceId: 1, onCreated: fn() },
} satisfies Meta<typeof InviteForm>

export default meta
type Story = StoryObj<typeof meta>

export const Form: Story = {}
export const Emailed: Story = {
    beforeEach: ({ msw }) => {
        const invite = spy('post', '/invitations', { id: 5, emailSent: true })
        sent.bodies = invite.bodies
        msw.use(invite.handler)
    },
    play: async ({ canvasElement, args }) => {
        await userEvent.type(await within(canvasElement).findByPlaceholderText(/email/i), 'alex@example.com{Enter}')
        await waitFor(() => expect(args.onCreated).toHaveBeenCalledWith({ id: 5, link: null }))
        await expect(sent.bodies).toEqual([{ role: 'editor', email: 'alex@example.com' }])
        await expect(await screen.findByText(/emailed to alex@example.com/i)).toBeVisible()
    },
}

// The server returns a token only for a link, and the link is built from it here.
export const LinkMade: Story = {
    beforeEach: ({ msw }) => {
        const invite = spy('post', '/invitations', { id: 6, token: 'tok123', emailSent: false })
        sent.bodies = invite.bodies
        msw.use(invite.handler)
    },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /copy a link/i }))
        await userEvent.type(canvas.getByLabelText(/note/i), 'Bob{Enter}')
        await waitFor(() =>
            expect(args.onCreated).toHaveBeenCalledWith({ id: 6, link: `${window.location.origin}/invite/tok123` }),
        )
        await expect(sent.bodies).toEqual([{ role: 'editor', note: 'Bob' }])
    },
}

// The server's reason is shown, and the address stays to be corrected.
export const Refused: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/invitations', 'alex@example.com is already a member', 422)),
    play: async ({ canvasElement, args }) => {
        const field = await within(canvasElement).findByPlaceholderText(/email/i)
        await userEvent.type(field, 'alex@example.com{Enter}')
        await expect(await screen.findByText(/already a member/i)).toBeVisible()
        await expect(field).toHaveValue('alex@example.com')
        await expect(args.onCreated).not.toHaveBeenCalled()
    },
}
