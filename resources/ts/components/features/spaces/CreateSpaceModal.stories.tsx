import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { fail, hang, reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { catalog, catalogItem, categories, currencies, tags } from '@/test/fixtures'
import { spaceDraft } from '@/stores/space-draft'
import type { SpacePresets } from '@/types/spaces'

import { emptyDraft, type SpaceDraft } from './create/draft'
import { CreateSpaceModal } from './CreateSpaceModal'

const presets: SpacePresets = {
    categories: [
        { name: '#OTHER', type: 'expense', icon: '📦', color: '#6b7280', isDefault: true },
        { name: 'Groceries', type: 'expense', icon: '🛒', color: '#16a34a', isDefault: false },
        { name: '#OTHER_INCOME', type: 'income', icon: '💰', color: '#2563eb', isDefault: true },
    ],
    tags: ['holiday'],
}

const items = [
    catalogItem({ code: 'USD', name: 'US Dollar', symbol: '$', rate: 1 }),
    catalogItem({ code: 'EUR', name: 'Euro', symbol: '€', rate: 1.25 }),
    catalogItem({ code: 'JPY', name: 'Japanese Yen', symbol: '¥', decimals: 0, rate: 0.005 }),
    ...catalog,
]

const created = { ...space(), id: 9, name: 'Family', emoji: null, color: null }

const sent: { bodies: unknown[] } = { bodies: [] }

// Saved in this browser, as the store would keep it for user 1.
const saveDraft = (draft: Partial<SpaceDraft>) =>
    localStorage.setItem(`${spaceDraft.key}:1`, JSON.stringify({ state: { draft: { ...emptyDraft(), ...draft } } }))

const meta = {
    title: 'features/spaces/CreateSpaceModal',
    component: CreateSpaceModal,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        const create = spy('post', '/spaces', created)
        sent.bodies = create.bodies
        msw.use(
            create.handler,
            reply('get', '/presets', presets),
            reply('get', '/currencies/catalog', items),
            reply('get', '/invitation-methods', { email: true, link: true }),
        )
    },
    args: { open: true, onOpenChange: fn() },
} satisfies Meta<typeof CreateSpaceModal>

export default meta
type Story = StoryObj<typeof meta>

const dialog = async () => within(await screen.findByRole('dialog'))
const next = async () => userEvent.click((await dialog()).getByRole('button', { name: /^continue$/i }))

export const NameStep: Story = {
    play: async () => {
        const d = await dialog()
        await expect(await d.findByLabelText('Name')).toBeVisible()
        // Nothing to go on with until the space has a name.
        await expect(d.getByRole('button', { name: /^continue$/i })).toBeDisabled()
    },
}

// The presets are needed to leave the first step.
export const WaitingForPresets: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/presets')),
    play: async () => {
        const d = await dialog()
        await userEvent.type(await d.findByLabelText('Name'), 'Family')
        await expect(d.getByRole('button', { name: /^continue$/i })).toBeDisabled()
    },
}

// A user without any space has to make one.
export const NotDismissible: Story = {
    args: { dismissible: false },
    play: async () => {
        const d = await dialog()
        await d.findByLabelText('Name')
        await expect(d.queryByRole('button', { name: /^cancel$/i })).toBeNull()
    },
}

export const Cancelled: Story = {
    play: async ({ args }) => {
        const d = await dialog()
        await userEvent.type(await d.findByLabelText('Name'), 'Family')
        await userEvent.click(d.getByRole('button', { name: /^cancel$/i }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
        await expect(sent.bodies).toHaveLength(0)
    },
}

// From the defaults to the request: nothing is sent before Create.
export const CreatedFromTheDefaults: Story = {
    play: async () => {
        let d = await dialog()
        await userEvent.type(await d.findByLabelText('Name'), ' Family ')
        await next()
        // Another space exists, so the copy step comes next; the defaults are already chosen.
        await expect(await d.findByRole('combobox', { name: /start from/i })).toHaveTextContent(/the defaults/i)
        await next()
        d = await dialog()
        await userEvent.click(await d.findByRole('combobox', { name: 'Base currency' }))
        await userEvent.click(await screen.findByRole('option', { name: /EUR/ }))
        // No account yet: Continue stays disabled until one exists.
        await expect(d.getByRole('button', { name: /^continue$/i })).toBeDisabled()
        await userEvent.click(d.getByRole('button', { name: /add an account/i }))
        await userEvent.type(d.getByRole('textbox', { name: 'Name' }), 'Cash')
        await userEvent.type(d.getByRole('textbox', { name: 'Opening balance' }), '10,50')
        await next()
        // Categories, then tags, then the summary.
        await expect(await d.findByRole('textbox', { name: /new category/i })).toBeVisible()
        await next()
        await expect(await d.findByRole('textbox', { name: /new tag/i })).toBeVisible()
        await next()
        await expect(await d.findByText('Family')).toBeVisible()
        await expect(sent.bodies).toHaveLength(0)
        await userEvent.click(d.getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(sent.bodies[0]).toEqual({
            name: 'Family',
            baseCurrency: 'EUR',
            currencies: [],
            // A comma is a decimal point; the amount stays a string.
            accounts: [{ name: 'Cash', type: 'bank', currency: 'EUR', initialBalance: '10.50' }],
            categories: presets.categories,
            tags: ['holiday'],
        })
        await expect(await screen.findByText(/"Family" is ready/)).toBeVisible()
    },
}

export const BadBalanceBlocksTheStep: Story = {
    beforeEach: () =>
        saveDraft({
            step: 'currencies',
            name: 'Family',
            baseCurrency: 'EUR',
            accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'EUR', initialBalance: '1.2.3' }],
            categories: [presets.categories[0]],
        }),
    play: async () => {
        const d = await dialog()
        await expect(await d.findByDisplayValue('1.2.3')).toBeVisible()
        await expect(d.getByRole('button', { name: /^continue$/i })).toBeDisabled()
    },
}

// A draft saved when currencies, categories and tags were one step resumes on the first of them, edits kept.
export const ResumedOldEntitiesDraft: Story = {
    beforeEach: () =>
        saveDraft({
            step: 'entities' as SpaceDraft['step'],
            name: 'Family',
            baseCurrency: 'EUR',
            accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'EUR', initialBalance: '' }],
            categories: [presets.categories[0]],
        }),
    play: async () => {
        const d = await dialog()
        await expect(await d.findByDisplayValue('Cash')).toBeVisible()
        await expect(d.getByRole('combobox', { name: 'Base currency' })).toHaveTextContent('EUR')
        // Forward and back again keeps what was edited.
        await next()
        await expect(await d.findByRole('textbox', { name: /new category/i })).toBeVisible()
        await userEvent.click(d.getByRole('button', { name: /^back$/i }))
        await expect(await d.findByDisplayValue('Cash')).toBeVisible()
    },
}

// The draft survives closing the modal and a reload.
export const ResumedDraft: Story = {
    beforeEach: () =>
        saveDraft({
            step: 'confirm',
            name: 'Trip',
            baseCurrency: 'JPY',
            currencies: ['EUR'],
            accounts: [{ key: 'a1', name: 'Yen wallet', type: 'cash', currency: 'JPY', initialBalance: '15000' }],
        }),
    play: async () => {
        const d = await dialog()
        await expect(await d.findByText('Trip')).toBeVisible()
        await expect(d.getByText('EUR')).toBeVisible()
        // Back from the summary: tags, categories, then the currencies and accounts.
        for (let i = 0; i < 3; i++) await userEvent.click(d.getByRole('button', { name: /^back$/i }))
        await expect(await d.findByDisplayValue('Yen wallet')).toBeVisible()
    },
}

// Ticked kinds come from the other space; accounts come without their balances.
export const CopiedFromAnotherSpace: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/spaces/1/currencies', currencies),
            reply('get', '/spaces/1/accounts', [
                { id: 1, name: 'Euro card', type: 'bank', currency: currencies[1], currentBalance: 980.1 },
            ]),
            reply('get', '/spaces/1/categories', categories),
            reply('get', '/spaces/1/tags', tags),
        ),
    play: async () => {
        const d = await dialog()
        await userEvent.type(await d.findByLabelText('Name'), 'Copy')
        await next()
        await userEvent.click(await d.findByRole('combobox', { name: /start from/i }))
        await userEvent.click(await screen.findByRole('option', { name: 'Home' }))
        await userEvent.click(await d.findByRole('checkbox', { name: 'Currencies' }))
        await userEvent.click(d.getByRole('checkbox', { name: /accounts/i }))
        await next()
        await expect(await d.findByRole('combobox', { name: 'Base currency' })).toHaveTextContent('USD')
        await expect(d.getByDisplayValue('Euro card')).toBeVisible()
        await expect(d.getByRole('textbox', { name: 'Opening balance' })).toHaveValue('')
        // Not ticked: the default categories, not the other space's.
        await expect(d.queryByDisplayValue('Transport')).toBeNull()
    },
}

export const CreateFailed: Story = {
    beforeEach: ({ msw }) => {
        msw.use(fail('post', '/spaces', 'You have reached the limit of spaces'))
        saveDraft({
            step: 'confirm',
            name: 'Family',
            baseCurrency: 'EUR',
            accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'EUR', initialBalance: '' }],
        })
    },
    play: async () => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('button', { name: /^create$/i }))
        await expect(await screen.findAllByText(/limit of spaces/i)).not.toHaveLength(0)
        // The draft stays, so nothing is lost.
        await expect(d.getByText('Family')).toBeVisible()
    },
}

// Invitations wait until the done screen is left, and then go out one by one.
export const CreatedAndInvited: Story = {
    beforeEach: ({ msw }) => {
        saveDraft({
            step: 'confirm',
            name: 'Family',
            baseCurrency: 'EUR',
            accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'EUR', initialBalance: '' }],
        })
        const invite = spy('post', '/spaces/9/invitations', { id: 1, token: 'tok123', emailSent: false })
        sent.bodies = invite.bodies
        msw.use(invite.handler, reply('get', '/spaces', [space(), created]))
    },
    play: async () => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('button', { name: /^create$/i }))
        await userEvent.click(await d.findByRole('button', { name: /invite members/i }))
        await userEvent.click(await d.findByRole('button', { name: /copy a link/i }))
        await userEvent.type(d.getByLabelText(/note/i), 'Bob{Enter}')
        await expect(await d.findByText('Bob')).toBeVisible()
        await expect(sent.bodies).toHaveLength(0)
        await userEvent.click(d.getByRole('button', { name: /^continue$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ role: 'editor', note: 'Bob' }]))
        // A link has to be copied, so the screen stays to show it.
        await expect(await d.findByRole('button', { name: /copy link/i })).toBeVisible()
        await expect(d.getByRole('button', { name: /^done$/i })).toBeVisible()
    },
}

export const CreatedAndLeft: Story = {
    beforeEach: ({ msw }) => {
        saveDraft({
            step: 'confirm',
            name: 'Family',
            baseCurrency: 'EUR',
            accounts: [{ key: 'a1', name: 'Cash', type: 'cash', currency: 'EUR', initialBalance: '' }],
        })
        msw.use(reply('get', '/spaces', [space(), created]))
    },
    play: async ({ args }) => {
        const d = await dialog()
        await userEvent.click(await d.findByRole('button', { name: /^create$/i }))
        await userEvent.click(await d.findByRole('button', { name: /^continue$/i }))
        await waitFor(() => expect(args.onOpenChange).toHaveBeenCalledWith(false))
    },
}
