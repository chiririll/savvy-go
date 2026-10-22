import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { fail, hang, reply, spy } from '@/test/api-mocks'
import { categories } from '@/test/fixtures'
import type { SyncDiff } from '@/types/spaces'

import { SyncReconcileDialog } from './SyncReconcileDialog'

const spaces = [
    { id: 1, name: 'Home' },
    { id: 2, name: 'Trip' },
]

// Home has Housing with transactions and Trip lacks it; both have Transport but drawn differently;
// a tag only Home has; and each space has its own default expense category.
const diff: SyncDiff = {
    spaces,
    categories: [
        {
            type: 'expense',
            name: 'Housing',
            present: [{ spaceId: 1, icon: '🏠', color: '#111111', isDefault: false, transactions: 4 }],
            missing: [2],
        },
        {
            type: 'expense',
            name: 'Transport',
            present: [
                { spaceId: 1, icon: '🚌', color: '#d97706', isDefault: false, transactions: 2 },
                { spaceId: 2, icon: '🚕', color: '#2563eb', isDefault: false, transactions: 1 },
            ],
            missing: [],
        },
    ],
    tags: [{ name: 'shared', present: [{ spaceId: 1, transactions: 0 }], missing: [2] }],
    defaults: [
        {
            type: 'expense',
            defaults: [
                { spaceId: 1, name: 'Other' },
                { spaceId: 2, name: 'Misc' },
            ],
        },
    ],
}

const sent: { diff: unknown[]; apply: unknown[] } = { diff: [], apply: [] }

const answer = (data: SyncDiff) => {
    const compare = spy('post', '/sync-groups/diff', data)
    const apply = spy('post', '/sync-groups/apply', { id: 7 })
    sent.diff = compare.bodies
    sent.apply = apply.bodies
    return [compare.handler, apply.handler]
}

const meta = {
    title: 'features/spaces/SyncReconcileDialog',
    component: SyncReconcileDialog,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/categories', categories), ...answer(diff)),
    args: { change: { add_space_ids: [1, 2], categories: true, tags: true }, onClose: fn() },
} satisfies Meta<typeof SyncReconcileDialog>

export default meta
type Story = StoryObj<typeof meta>

const dialog = async () => within(await screen.findByRole('dialog'))
const next = async () => userEvent.click((await dialog()).getByRole('button', { name: /^next$/i }))

export const Categories: Story = {
    play: async () => {
        const d = await dialog()
        await expect(await d.findByText(/step 1 of 3: categories/i)).toBeVisible()
        await expect(d.getByText(/in: home · missing in: trip/i)).toBeVisible()
        // Only Transport looks different, so only it asks whose look to keep.
        await expect(d.getAllByRole('combobox', { name: /icon and color/i })).toHaveLength(1)
        await expect(sent.diff).toEqual([{ add_space_ids: [1, 2], categories: true, tags: true }])
        await expect(d.queryByRole('button', { name: /apply/i })).toBeNull()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const Comparing: Story = {
    beforeEach: ({ msw }) => msw.use(hang('post', '/sync-groups/diff')),
    play: async () => {
        const d = await dialog()
        await expect(d.getByText(/comparing the spaces/i)).toBeVisible()
        await expect(d.queryByRole('button', { name: /next|apply/i })).toBeNull()
    },
}

export const CompareFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/sync-groups/diff', 'A space is unavailable')),
    play: async () => {
        await expect(await (await dialog()).findByText(/a space is unavailable/i)).toBeVisible()
    },
}

export const AlreadyIdentical: Story = {
    beforeEach: ({ msw }) => msw.use(...answer({ spaces, categories: [], tags: [], defaults: [] })),
    play: async () => {
        const d = await dialog()
        await expect(await d.findByText(/already identical/i)).toBeVisible()
        await userEvent.click(d.getByRole('button', { name: /apply/i }))
        await waitFor(() => expect(sent.apply).toHaveLength(1))
    },
}

export const OnlyDefaultsDiffer: Story = {
    beforeEach: ({ msw }) => msw.use(...answer({ ...diff, categories: [], tags: [] })),
    play: async () => {
        const d = await dialog()
        await expect(await d.findByText(/only the default categories differ/i)).toBeVisible()
        await expect(d.getByText('Home: Other · Trip: Misc')).toBeVisible()
    },
}

// Everything left as it starts keeps all, adds what is missing and takes the first look and default.
export const AppliedAsSuggested: Story = {
    play: async ({ args }) => {
        await (await dialog()).findByText(/step 1 of 3/i)
        await next()
        await expect(await (await dialog()).findByText('#shared')).toBeVisible()
        await next()
        const d = await dialog()
        await expect(d.getByText(/categories: 2 stay, 0 deleted everywhere/i)).toBeVisible()
        await expect(d.getByText(/tags: 1 stay, 0 deleted everywhere/i)).toBeVisible()
        await userEvent.click(d.getByRole('button', { name: /apply/i }))
        await waitFor(() => expect(sent.apply).toHaveLength(1))
        await expect(sent.apply[0]).toEqual({
            add_space_ids: [1, 2],
            categories: true,
            tags: true,
            decisions: [
                { kind: 'categories', type: 'expense', name: 'Housing', keep: true },
                { kind: 'categories', type: 'expense', name: 'Transport', keep: true, from_space_id: 1 },
                { kind: 'tags', name: 'shared', keep: true },
            ],
            defaults: [{ type: 'expense', name: 'Other' }],
        })
        await waitFor(() => expect(args.onClose).toHaveBeenCalled())
    },
}

// Deleting a category that holds transactions adds a step to say where they go.
export const DeletedCategoryNeedsASuccessor: Story = {
    play: async () => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('combobox', { name: 'Housing' }))
        await userEvent.click(await screen.findByRole('option', { name: /delete everywhere/i }))
        await expect(await d.findByText(/step 1 of 4/i)).toBeVisible()
        await next()
        await expect(await d.findByText(/step 2 of 4: move transactions/i)).toBeVisible()
        await expect(d.getByRole('button', { name: /^next$/i })).toBeDisabled()
        await userEvent.click(d.getByRole('combobox', { name: /move its transactions/i }))
        // Categories that differ between the spaces cannot take them.
        await expect(screen.queryByRole('option', { name: 'Transport' })).toBeNull()
        await userEvent.click(await screen.findByRole('option', { name: 'Groceries' }))
        await next()
        await next()
        await expect(d.getByText(/categories: 1 stay, 1 deleted everywhere/i)).toBeVisible()
        await userEvent.click(d.getByRole('button', { name: /apply/i }))
        await waitFor(() => expect(sent.apply).toHaveLength(1))
        await expect((sent.apply[0] as { decisions: unknown[] }).decisions[0]).toEqual({
            kind: 'categories',
            type: 'expense',
            name: 'Housing',
            keep: false,
            successor: 'Groceries',
        })
    },
}

export const OtherLookAndDefaultChosen: Story = {
    play: async () => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('combobox', { name: /icon and color/i }))
        await userEvent.click(await screen.findByRole('option', { name: /trip/i }))
        await next()
        await next()
        await userEvent.click(d.getByRole('combobox', { name: /default expense category/i }))
        await userEvent.click(await screen.findByRole('option', { name: 'Misc' }))
        await userEvent.click(d.getByRole('button', { name: /apply/i }))
        await waitFor(() => expect(sent.apply).toHaveLength(1))
        const plan = sent.apply[0] as { decisions: { from_space_id?: number }[]; defaults: unknown[] }
        await expect(plan.decisions[1].from_space_id).toBe(2)
        await expect(plan.defaults).toEqual([{ type: 'expense', name: 'Misc' }])
    },
}

export const TagDropped: Story = {
    play: async () => {
        const d = await dialog()
        await d.findByText(/step 1 of 3/i)
        await next()
        await userEvent.click(await d.findByRole('combobox', { name: '#shared' }))
        await userEvent.click(await screen.findByRole('option', { name: /transactions lose the tag/i }))
        await next()
        await expect(d.getByText(/tags: 0 stay, 1 deleted everywhere/i)).toBeVisible()
    },
}

// Back keeps what was chosen.
export const BackKeepsChoices: Story = {
    play: async () => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('combobox', { name: 'Transport' }))
        await userEvent.click(await screen.findByRole('option', { name: /delete everywhere/i }))
        await next()
        await userEvent.click(d.getByRole('button', { name: /back/i }))
        await expect(d.getByRole('combobox', { name: 'Transport' })).toHaveTextContent(/delete everywhere/i)
    },
}

export const Cancelled: Story = {
    play: async ({ args }) => {
        await userEvent.click((await dialog()).getByRole('button', { name: /cancel/i }))
        await expect(args.onClose).toHaveBeenCalled()
        await expect(sent.apply).toHaveLength(0)
    },
}

export const Closed: Story = {
    args: { change: null },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
        await expect(sent.diff).toHaveLength(0)
    },
}
