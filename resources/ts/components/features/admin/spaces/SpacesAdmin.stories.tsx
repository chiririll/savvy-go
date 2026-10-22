import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, spy } from '@/test/api-mocks'
import type { Backup } from '@/types/backup'
import type { SpaceOverview } from '@/types/spaces'

import { DeletedSpacesSection } from './DeletedSpacesSection'
import { DeleteSpaceButton } from './DeleteSpaceButton'
import { AssignAdmin, QuotaCell } from './SpaceCells'

const MB = 1024 * 1024

const overview = (overrides: Partial<SpaceOverview> = {}): SpaceOverview => ({
    id: 3,
    uuid: '00000000-0000-4000-8000-000000000003',
    name: 'Family',
    emoji: '🏡',
    color: null,
    members: 3,
    admins: 1,
    size: 4 * MB,
    quota: 0,
    unavailable: '',
    createdAt: '2030-01-01T00:00:00Z',
    ...overrides,
})

const users = [
    { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', emailVerified: true },
    { id: 2, name: 'Alex', email: 'alex@example.com', role: 'user', emailVerified: true },
    { id: 3, name: 'Guest', email: 'guest@example.com', role: 'guest', emailVerified: true },
]

const deleted = (overrides: Partial<Backup> & { spaceName?: string } = {}) => ({
    filename: 'space-family-2030-01-05.zip',
    size: 2 * MB,
    note: null,
    appVersion: '1.4.0',
    kind: 'space',
    status: 'current',
    signature: 'own',
    restorable: true,
    createdAt: '2030-01-05T10:00:00Z',
    spaceName: 'Family',
    ...overrides,
})

const sent: { bodies: unknown[] } = { bodies: [] }

// The controls of the admin spaces table and the deleted spaces under it.
const meta = {
    title: 'features/admin/SpacesAdmin',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/users', users)),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Quota: Story = {
    render: () => <QuotaCell space={overview({ quota: 10 * MB })} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('spinbutton', { name: /size limit/i })).toHaveValue(10)
        await expect(canvas.queryByRole('button', { name: /save/i })).toBeNull()
    },
}

export const QuotaSaved: Story = {
    render: () => <QuotaCell space={overview()} />,
    beforeEach: ({ msw }) => {
        const quota = spy('patch', '/admin/spaces/3', [])
        sent.bodies = quota.bodies
        msw.use(quota.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('spinbutton', { name: /size limit/i }), '1.5')
        await userEvent.click(canvas.getByRole('button', { name: /save/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ quota_bytes: Math.round(1.5 * MB) }]))
    },
}

// Emptying the field goes back to the server's default.
export const QuotaCleared: Story = {
    render: () => <QuotaCell space={overview({ quota: 10 * MB })} />,
    beforeEach: ({ msw }) => {
        const quota = spy('patch', '/admin/spaces/3', [])
        sent.bodies = quota.bodies
        msw.use(quota.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.clear(canvas.getByRole('spinbutton', { name: /size limit/i }))
        await userEvent.click(canvas.getByRole('button', { name: /save/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ quota_bytes: null }]))
    },
}

export const AdminAssigned: Story = {
    render: () => <AssignAdmin space={overview()} />,
    beforeEach: ({ msw }) => {
        const assign = spy('post', '/admin/spaces/3/admins')
        sent.bodies = assign.bodies
        msw.use(assign.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('button', { name: /make admin/i })).toBeDisabled()
        await userEvent.click(canvas.getByRole('combobox', { name: 'User' }))
        // A guest cannot administer a space.
        await expect(await screen.findAllByRole('option')).toHaveLength(2)
        await userEvent.click(screen.getByRole('option', { name: 'Alex' }))
        await userEvent.click(canvas.getByRole('button', { name: /make admin/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ user_id: 2 }]))
        await expect(await screen.findByText('Admin assigned')).toBeVisible()
    },
}

export const SpaceDeleted: Story = {
    render: () => <DeleteSpaceButton space={overview()} />,
    beforeEach: ({ msw }) => {
        const remove = spy('delete', '/admin/spaces/3')
        sent.bodies = remove.bodies
        msw.use(remove.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /delete/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText('Delete Family?')).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /delete/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

export const Deleted: Story = {
    render: () => <DeletedSpacesSection />,
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/admin/deleted-spaces', [
                deleted(),
                deleted({ filename: 'space-old.zip', spaceName: undefined }),
            ]),
        ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Family')).toBeVisible()
        // An old backup without its space's name shows the file name.
        await expect(canvas.getByText('space-old.zip')).toBeVisible()
    },
}

export const DeletedInRussian: Story = { ...Deleted, play: undefined, globals: { locale: 'ru' } }

export const DeletedRestored: Story = {
    render: () => <DeletedSpacesSection />,
    beforeEach: ({ msw }) => {
        const restore = spy('post', '/admin/deleted-spaces/space-family-2030-01-05.zip/restore', { id: 9 })
        sent.bodies = restore.bodies
        msw.use(reply('get', '/admin/deleted-spaces', [deleted()]), restore.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /restore/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText('Space restored')).toBeVisible()
    },
}

export const NoneDeleted: Story = {
    render: () => <DeletedSpacesSection />,
    beforeEach: ({ msw }) => msw.use(reply('get', '/admin/deleted-spaces', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No deleted spaces.')).toBeVisible()
    },
}
