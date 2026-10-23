import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { raw, reply, spy } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'
import type { ApiTokenSummary } from '@/types'

import { capabilities } from './story-fixtures'
import { EditTokenDialog } from './EditTokenDialog'

const token: ApiTokenSummary = {
    id: 7,
    name: 'Home dashboard',
    prefix: 'sv_ab12',
    kind: 'personal',
    permissions: ['transactions:read', 'accounts:read', 'categories:read'],
    all_spaces: false,
    space_ids: [1],
    expires_at: null,
    last_used_at: null,
    created_at: '2029-12-01T10:00:00Z',
}

const spaces = [
    { id: 1, name: 'Home' },
    { id: 2, name: 'Trip' },
]

const updated = spy('patch', '/auth/api-tokens/7', token)

const meta = {
    title: 'features/api-tokens/EditTokenDialog',
    component: EditTokenDialog,
    parameters: { a11y: overlayA11y },
    args: { token, spaces, onClose: fn() },
    beforeEach: ({ msw }) => {
        // An account that signs in through SSO has nothing to confirm a credential change with.
        useAuthStore.setState((state) => ({ user: state.user && { ...state.user, isSsoOnly: true } }))
        msw.use(raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }))
        updated.bodies.length = 0
        msw.use(
            updated.handler,
            reply('get', '/auth/api-token-capabilities', capabilities),
            reply('get', '/sync-group', null),
        )
    },
} satisfies Meta<typeof EditTokenDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Current: Story = {
    play: async () => {
        const dialog = within(await screen.findByRole('dialog', { name: /home dashboard/i }))
        await expect(dialog.getByRole('checkbox', { name: 'Home' })).toBeChecked()
        await expect(dialog.getByRole('checkbox', { name: 'Trip' })).not.toBeChecked()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const AddsASpace: Story = {
    play: async ({ args }) => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.click(dialog.getByRole('checkbox', { name: 'Trip' }))
        await userEvent.click(dialog.getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onClose).toHaveBeenCalled())
        await expect(updated.bodies[0]).toMatchObject({ all_spaces: false, space_ids: [1, 2] })
    },
}

export const NoSpaceLeft: Story = {
    play: async () => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.click(dialog.getByRole('checkbox', { name: 'Home' }))
        await expect(dialog.getByRole('button', { name: /save/i })).toBeDisabled()
        await expect(dialog.getByText(/choose at least one space/i)).toBeVisible()
    },
}

export const Cancel: Story = {
    play: async ({ args }) => {
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onClose).toHaveBeenCalled()
    },
}
