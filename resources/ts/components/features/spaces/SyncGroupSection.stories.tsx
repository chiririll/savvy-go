import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { categories } from '@/test/fixtures'
import type { SyncGroup } from '@/types/spaces'

import { SyncGroupSection } from './SyncGroupSection'

const mine = [
    space(),
    { ...space(), id: 2, name: 'Work', emoji: '💼' },
    { ...space(), id: 3, name: 'Trip', emoji: '✈️' },
    { ...space('viewer'), id: 4, name: 'Parents', emoji: null },
]

const group = (overrides: Partial<SyncGroup> = {}): SyncGroup => ({
    id: 7,
    categories: true,
    tags: false,
    members: [
        { id: 1, name: 'Home', role: 'admin' },
        { id: 2, name: 'Work', role: 'admin' },
    ],
    hidden: 0,
    ...overrides,
})

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/SyncGroupSection',
    component: SyncGroupSection,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/spaces', mine),
            reply('get', '/sync-group', group()),
            reply('get', '/categories', categories),
            // The reconcile dialog waits for the comparison unless a story answers it.
            hang('post', '/sync-groups/diff'),
        ),
    args: { space: space(), isAdmin: true },
} satisfies Meta<typeof SyncGroupSection>

export default meta
type Story = StoryObj<typeof meta>

export const InAGroup: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Work')).toBeVisible()
        await expect(canvas.getByRole('switch', { name: 'Categories' })).toBeChecked()
        await expect(canvas.getByRole('switch', { name: 'Tags' })).not.toBeChecked()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const WithSpacesYouAreNotIn: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', group({ hidden: 2 }))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/2 more spaces you are not a member of/i)).toBeVisible()
    },
}

// Switching a kind on has to reconcile the spaces first.
export const SwitchingOnAsksToReconcile: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('switch', { name: 'Tags' }))
        const dialog = within(await screen.findByRole('dialog'))
        await expect(dialog.getByText(/comparing the spaces/i)).toBeVisible()
    },
}

// Switching off needs no decisions and applies at once.
export const SwitchedOff: Story = {
    beforeEach: ({ msw }) => {
        const apply = spy('post', '/sync-groups/apply', { id: 7 })
        sent.bodies = apply.bodies
        msw.use(apply.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('switch', { name: 'Categories' }))
        await waitFor(() =>
            expect(sent.bodies).toEqual([{ group_id: 7, categories: false, tags: false, decisions: [], defaults: [] }]),
        )
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

export const SpaceAdded: Story = {
    beforeEach: ({ msw }) => {
        const diff = spy('post', '/sync-groups/diff', { categories: [], tags: [], defaults: [], spaces: [] })
        sent.bodies = diff.bodies
        msw.use(diff.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('combobox', { name: /space to add/i }))
        // Only spaces the user administers can join.
        await expect(await screen.findAllByRole('option')).toHaveLength(1)
        await userEvent.click(screen.getByRole('option', { name: 'Trip' }))
        await userEvent.click(canvas.getByRole('button', { name: /^add$/i }))
        await waitFor(() =>
            expect(sent.bodies).toEqual([{ group_id: 7, add_space_ids: [3], categories: true, tags: false }]),
        )
        await expect(await screen.findByText(/already identical/i)).toBeVisible()
    },
}

export const RemoveAsksFirst: Story = {
    beforeEach: ({ msw }) => {
        const apply = spy('post', '/sync-groups/apply', null)
        sent.bodies = apply.bodies
        msw.use(apply.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /remove from group/i }))[1])
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText('Take Work out of the group?')).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /remove from group/i }))
        await waitFor(() =>
            expect(sent.bodies).toEqual([
                { group_id: 7, remove_space_ids: [2], categories: true, tags: false, decisions: [], defaults: [] },
            ]),
        )
    },
}

// A member sees the group but changes nothing.
export const AsMember: Story = {
    args: { space: space('editor'), isAdmin: false },
    beforeEach: ({ msw }) =>
        msw.use(
            reply(
                'get',
                '/sync-group',
                group({
                    members: [
                        { id: 1, name: 'Home', role: 'editor' },
                        { id: 2, name: 'Work', role: 'viewer' },
                    ],
                }),
            ),
        ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('switch', { name: 'Categories' })).toBeDisabled()
        await expect(canvas.queryByRole('button')).toBeNull()
    },
}

// A member of a space in no group has nothing to see here.
export const MemberWithoutAGroup: Story = {
    args: { space: space('editor'), isAdmin: false },
    beforeEach: ({ msw }) => msw.use(noGroup()),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(canvasElement.firstElementChild).toBeEmptyDOMElement()
    },
}

export const SetUp: Story = {
    beforeEach: ({ msw }) => {
        const diff = spy('post', '/sync-groups/diff', { categories: [], tags: [], defaults: [], spaces: [] })
        sent.bodies = diff.bodies
        msw.use(noGroup(), diff.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/not in a group/i)).toBeVisible()
        const setUp = canvas.getByRole('button', { name: /set up/i })
        await expect(setUp).toBeDisabled()
        await userEvent.click(canvas.getByRole('checkbox', { name: 'Trip' }))
        await userEvent.click(canvas.getByRole('switch', { name: 'Tags' }))
        await userEvent.click(setUp)
        await waitFor(() => expect(sent.bodies).toEqual([{ add_space_ids: [1, 3], categories: true, tags: false }]))
    },
}

// Nothing kept identical is no group at all.
export const SetUpNeedsAKind: Story = {
    beforeEach: ({ msw }) => msw.use(noGroup()),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('checkbox', { name: 'Work' }))
        await userEvent.click(canvas.getByRole('switch', { name: 'Categories' }))
        await userEvent.click(canvas.getByRole('switch', { name: 'Tags' }))
        await expect(canvas.getByRole('button', { name: /set up/i })).toBeDisabled()
    },
}

export const NothingToGroupWith: Story = {
    beforeEach: ({ msw }) => msw.use(noGroup(), reply('get', '/spaces', [space()])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/ask them to invite you as an admin/i)).toBeVisible()
    },
}

// The API drops a null `data`, so a space in no group answers with an empty envelope.
function noGroup() {
    return reply('get', '/sync-group', undefined)
}
