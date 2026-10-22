import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import type { SpaceDetails } from '@/types/spaces'

import { General } from './General'

const details = (overrides: Partial<SpaceDetails> = {}) =>
    reply('get', '/spaces/1', { ...space(), quota: 0, size: 3_500_000, ...overrides })

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/settings/General',
    component: General,
    beforeEach: ({ msw }) => msw.use(details()),
    args: { space: space(), isAdmin: true },
} satisfies Meta<typeof General>

export default meta
type Story = StoryObj<typeof meta>

const nameField = (canvasElement: HTMLElement) => within(canvasElement).getByLabelText('Name')

export const AsAdmin: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Uses 3.3 MB.')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /delete space/i })).toBeVisible()
        // Nothing changed yet, so nothing to save.
        await expect(canvas.queryByRole('button', { name: /^save$/i })).toBeNull()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const WithQuota: Story = {
    beforeEach: ({ msw }) => msw.use(details({ quota: 10_000_000 })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Uses 3.3 MB of 9.5 MB.')).toBeVisible()
    },
}

export const LoadingUsage: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/spaces/1')) }

// A member can only leave.
export const AsMember: Story = {
    args: { space: space('editor'), isAdmin: false },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(nameField(canvasElement)).toBeDisabled()
        await expect(canvas.getByRole('button', { name: /leave/i })).toBeVisible()
        await expect(canvas.queryByRole('button', { name: /delete space/i })).toBeNull()
    },
}

export const Renamed: Story = {
    beforeEach: ({ msw }) => {
        const update = spy('patch', '/spaces/1', { ...space(), name: 'Family', quota: 0, size: 0 })
        sent.bodies = update.bodies
        msw.use(update.handler)
    },
    play: async ({ canvasElement }) => {
        const field = nameField(canvasElement)
        await userEvent.clear(field)
        await userEvent.type(field, ' Family ')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ name: 'Family', emoji: '🏠', color: '#2563eb' }]))
        await expect(await screen.findByText(/space updated/i)).toBeVisible()
    },
}

export const EmptyNameCannotBeSaved: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.clear(nameField(canvasElement))
        await expect(within(canvasElement).getByRole('button', { name: /^save$/i })).toBeDisabled()
    },
}

export const RenameFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('patch', '/spaces/1', 'The space is not available')),
    play: async ({ canvasElement }) => {
        await userEvent.type(nameField(canvasElement), '!')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await expect(await screen.findByText(/the space is not available/i)).toBeVisible()
    },
}

export const DeleteAsksFirst: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /delete space/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText('Delete Home?')).toBeVisible()
        await expect(dialog.getByText(/kept for the server administrators/i)).toBeVisible()
    },
}

export const Deleted: Story = {
    beforeEach: ({ msw }) => {
        const remove = spy('delete', '/spaces/1')
        sent.bodies = remove.bodies
        msw.use(remove.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /delete space/i }))
        await userEvent.click(
            within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete space/i }),
        )
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText(/space deleted/i)).toBeVisible()
    },
}

export const Left: Story = {
    args: { space: space('viewer'), isAdmin: false },
    beforeEach: ({ msw }) => {
        const leave = spy('post', '/spaces/1/leave')
        sent.bodies = leave.bodies
        msw.use(leave.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /leave/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/lose access/i)).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /leave/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

export const LeaveCancelled: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /leave/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull())
    },
}
