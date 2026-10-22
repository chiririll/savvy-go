import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { accounts } from '@/test/fixtures'
import type { SpaceTransfer } from '@/types/spaces'

import { TransfersNeedingAttention } from './TransfersNeedingAttention'

// The two sides differ in currency and precision: 50 EUR out of here became 7500 JPY there.
const transfer = (overrides: Partial<SpaceTransfer> = {}): SpaceTransfer => ({
    uuid: 'a1',
    from: { amount: '50.00', currency: 'EUR', accountId: 2 },
    to: { amount: '7500', currency: 'JPY' },
    outgoing: true,
    date: '2030-01-05',
    description: 'Rent share',
    version: 1,
    status: 'ok',
    review: null,
    frozen: false,
    verified: true,
    deleted: false,
    otherSpace: { id: 2, name: 'Work' },
    ...overrides,
})

const transfers = (list: SpaceTransfer[]) => reply('get', '/transfers', list)

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/TransfersNeedingAttention',
    component: TransfersNeedingAttention,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/accounts', accounts)),
    args: { space: space() },
} satisfies Meta<typeof TransfersNeedingAttention>

export default meta
type Story = StoryObj<typeof meta>

// A restore brought back an older version: the other space has the newer one.
const restored = transfer({
    status: 'pending',
    review: 'changed_remote',
    remote: {
        from: { amount: '55.00', currency: 'EUR' },
        to: { amount: '8250', currency: 'JPY' },
        date: '2030-01-06',
        description: 'Rent share',
        deleted: false,
    },
})

export const AfterARestore: Story = {
    beforeEach: ({ msw }) => msw.use(transfers([restored])),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('−50.00 EUR')).toBeVisible()
        await expect(canvas.getByText(/the other space has 55.00 EUR on 2030-01-06/i)).toBeVisible()
        await expect(canvas.getByText(/changed in the other space/i)).toBeVisible()
    },
}

export const InRussian: Story = { ...AfterARestore, play: undefined, globals: { locale: 'ru' } }

// Nothing to decide, so nothing is shown.
export const AllFine: Story = {
    beforeEach: ({ msw }) => msw.use(transfers([transfer()])),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(canvasElement.firstElementChild).toBeEmptyDOMElement()
    },
}

// An incoming transfer shows the side that landed here.
export const Incoming: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            transfers([
                transfer({
                    outgoing: false,
                    from: { amount: '7500', currency: 'JPY' },
                    to: { amount: '12.345', currency: 'KWD' },
                    status: 'needs_attention',
                    description: null,
                }),
            ]),
        ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('+12.345 KWD')).toBeVisible()
        await expect(canvas.getByText(/^transfer · work/i)).toBeVisible()
    },
}

export const Accepted: Story = {
    beforeEach: ({ msw }) => {
        const accept = spy('post', '/transfers/a1/accept')
        sent.bodies = accept.bodies
        msw.use(transfers([restored]), accept.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /other space's version/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText(/transfer updated/i)).toBeVisible()
    },
}

export const Rejected: Story = {
    beforeEach: ({ msw }) => {
        const reject = spy('post', '/transfers/a1/reject')
        sent.bodies = reject.bodies
        msw.use(transfers([restored]), reject.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /keep this version/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

// The other space deleted it after the backup was made.
export const DeletedThere: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            transfers([
                transfer({
                    status: 'pending',
                    review: 'deleted_remote',
                    remote: { ...restored.remote!, deleted: true },
                }),
            ]),
        ),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/the other space deleted it/i)).toBeVisible()
    },
}

// An unverified signature cannot be kept as it is, only taken from the other side or trusted.
export const Unverified: Story = {
    beforeEach: ({ msw }) => {
        const trust = spy('post', '/transfers/a1/trust')
        sent.bodies = trust.bodies
        msw.use(transfers([transfer({ status: 'pending', review: 'changed_remote', verified: false })]), trust.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/signature not verified/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /keep this version/i })).toBeDisabled()
        await expect(canvas.queryByRole('button', { name: /delete/i })).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /trust as is/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

// One made after the backup has nothing here to keep, so keeping this side is allowed unsigned.
export const UnverifiedMadeAfterTheBackup: Story = {
    beforeEach: ({ msw }) =>
        msw.use(transfers([transfer({ status: 'pending', review: 'created_remote', verified: false })])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /keep this version/i })).toBeEnabled()
    },
}

export const AccountChosen: Story = {
    beforeEach: ({ msw }) => {
        const update = spy('patch', '/transfers/a1', transfer())
        sent.bodies = update.bodies
        msw.use(transfers([transfer({ status: 'needs_attention' })]), update.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/account missing/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /^save$/i })).toBeDisabled()
        await userEvent.click(canvas.getByRole('combobox', { name: /choose an account/i }))
        await userEvent.click(await screen.findByRole('option', { name: 'Euro card' }))
        await userEvent.click(canvas.getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ from_account_id: 2 }]))
    },
}

export const IncomingAccountChosen: Story = {
    beforeEach: ({ msw }) => {
        const update = spy('patch', '/transfers/a1', transfer())
        sent.bodies = update.bodies
        msw.use(transfers([transfer({ status: 'needs_attention', outgoing: false })]), update.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('combobox', { name: /choose an account/i }))
        await userEvent.click(await screen.findByRole('option', { name: 'Yen wallet' }))
        await userEvent.click(canvas.getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ to_account_id: 3 }]))
    },
}

export const Deleted: Story = {
    beforeEach: ({ msw }) => {
        const remove = spy('delete', '/transfers/a1')
        sent.bodies = remove.bodies
        msw.use(transfers([transfer({ status: 'needs_attention' })]), remove.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /^delete$/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

// The other space is not linked any more: it can only be looked at.
export const Frozen: Story = {
    beforeEach: ({ msw }) => msw.use(transfers([transfer({ frozen: true, otherSpace: undefined })])),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/space not linked/i)).toBeVisible()
        await expect(canvas.getByText(/a space not on this server/i)).toBeVisible()
        await expect(canvas.queryByRole('button')).toBeNull()
    },
}

export const Several: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            transfers([
                restored,
                transfer({ uuid: 'b2', status: 'needs_attention', description: 'Groceries split' }),
                transfer({ uuid: 'c3', frozen: true, description: 'Old loan' }),
            ]),
        ),
}
