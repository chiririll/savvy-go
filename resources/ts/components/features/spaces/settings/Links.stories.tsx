import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'

import { Links } from './Links'

// Spaces the user belongs to: one is linked already, one can be, and in one they are only an editor.
const mine = [
    space(),
    { ...space(), id: 2, name: 'Work', emoji: '💼' },
    { ...space(), id: 3, name: 'Trip', emoji: '✈️' },
    { ...space('editor'), id: 4, name: 'Parents', emoji: null },
]

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/settings/Links',
    component: Links,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces', mine), reply('get', '/links', [{ id: 2, name: 'Work' }])),
    args: { space: space(), isAdmin: true },
} satisfies Meta<typeof Links>

export default meta
type Story = StoryObj<typeof meta>

export const AsAdmin: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Work')).toBeVisible()
        await expect(await canvas.findByRole('button', { name: /^link$/i })).toBeDisabled()
    },
}

export const NoLinks: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/links', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no linked spaces/i)).toBeVisible()
    },
}

// Only spaces the user administers, and not linked yet, can be chosen.
export const Candidates: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('combobox'))
        await expect(await screen.findAllByRole('option')).toHaveLength(1)
        await expect(screen.getByRole('option', { name: 'Trip' })).toBeVisible()
    },
}

export const Linked: Story = {
    beforeEach: ({ msw }) => {
        const link = spy('post', '/links', [])
        sent.bodies = link.bodies
        msw.use(link.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: 'Trip' }))
        await userEvent.click(canvas.getByRole('button', { name: /^link$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ target_space_id: 3 }]))
    },
}

export const NothingToLink: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces', [space()])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/ask them to invite you as an admin/i)).toBeVisible()
    },
}

export const UnlinkAsksFirst: Story = {
    beforeEach: ({ msw }) => {
        const unlink = spy('delete', '/links/2')
        sent.bodies = unlink.bodies
        msw.use(unlink.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /unlink/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText('Unlink Work?')).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /unlink/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

// A member sees the links but cannot change them.
export const AsMember: Story = {
    args: { space: space('editor'), isAdmin: false },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Work')).toBeVisible()
        await expect(canvas.queryByRole('button')).toBeNull()
        await expect(canvas.queryByRole('combobox')).toBeNull()
    },
}
