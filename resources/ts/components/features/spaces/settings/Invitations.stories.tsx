import type { Meta, StoryObj } from '@storybook/react-vite'
import { http, HttpResponse } from 'msw'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import type { SpaceInvitation } from '@/types/spaces'

import { Invitations } from './Invitations'

const invitation = (overrides: Partial<SpaceInvitation> = {}): SpaceInvitation => ({
    id: 1,
    email: 'alex@example.com',
    note: null,
    role: 'editor',
    inviter: 'Kirill',
    expiresAt: '2099-01-01T00:00:00Z',
    acceptedAt: null,
    createdAt: '2030-01-01T00:00:00Z',
    ...overrides,
})

const open = [invitation(), invitation({ id: 2, email: null, note: 'For grandma', role: 'viewer' })]

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/settings/Invitations',
    component: Invitations,
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/invitations', open), reply('get', '/invitation-methods', { email: true, link: true })),
    args: { space: space() },
} satisfies Meta<typeof Invitations>

export default meta
type Story = StoryObj<typeof meta>

export const Open: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('alex@example.com')).toBeVisible()
        await expect(canvas.getByText('For grandma')).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

// Accepted and expired ones are not open any more.
export const OnlyOpenOnesListed: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/invitations', [
                invitation({ acceptedAt: '2030-01-02T00:00:00Z' }),
                invitation({ id: 2, email: 'old@example.com', expiresAt: '2020-01-01T00:00:00Z' }),
            ]),
        ),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no open invitations/i)).toBeVisible()
    },
}

// The server keeps only a hash, so the link can be copied only right after it was made.
export const NewLinkCanBeCopied: Story = {
    beforeEach: ({ msw }) => {
        const made = invitation({ id: 3, email: null, note: 'Bob' })
        let list = open
        msw.use(
            http.post(/\/api\/spaces\/1\/invitations$/, () => {
                list = [...open, made]
                return HttpResponse.json({ data: { id: 3, token: 'tok123', emailSent: false } })
            }),
            http.get(/\/api\/spaces\/1\/invitations$/, () => HttpResponse.json({ data: list })),
        )
        Object.defineProperty(navigator, 'clipboard', { value: { writeText: fn(async () => {}) }, configurable: true })
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByText('alex@example.com')
        await expect(canvas.queryByRole('button', { name: /copy link/i })).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /copy a link/i }))
        await userEvent.type(canvas.getByLabelText(/note/i), 'Bob{Enter}')
        await expect(await canvas.findByText(/shown only once/i)).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /copy link/i }))
        await waitFor(() =>
            expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`${window.location.origin}/invite/tok123`),
        )
        await expect(await screen.findByText(/link copied/i)).toBeVisible()
    },
}

export const RevokeAsksFirst: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /revoke/i }))[1])
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/the link for anyone with the link will stop working/i)).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /keep it/i }))
        await waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull())
    },
}

export const Revoked: Story = {
    beforeEach: ({ msw }) => {
        const revoke = spy('delete', '/invitations/1')
        sent.bodies = revoke.bodies
        msw.use(revoke.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /revoke/i }))[0])
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/alex@example.com will stop working/i)).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /cancel invitation/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText(/invitation revoked/i)).toBeVisible()
    },
}
