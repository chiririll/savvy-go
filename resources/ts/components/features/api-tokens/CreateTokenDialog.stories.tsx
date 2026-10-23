import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { fail, raw, reply, spy } from '@/test/api-mocks'

import { useAuthStore } from '@/stores/auth'
import { capabilities } from './story-fixtures'
import { CreateTokenDialog } from './CreateTokenDialog'

const spaces = [
    { id: 1, name: 'Home' },
    { id: 2, name: 'Trip' },
]

const created = spy('post', '/auth/api-tokens', { token: 'sv_ab12_secret' })

const meta = {
    title: 'features/api-tokens/CreateTokenDialog',
    component: CreateTokenDialog,
    parameters: { a11y: overlayA11y },
    args: { open: true, spaces, onOpenChange: fn(), onCreated: fn() },
    beforeEach: ({ msw }) => {
        // An account that signs in through SSO has nothing to confirm a credential change with.
        useAuthStore.setState((state) => ({ user: state.user && { ...state.user, isSsoOnly: true } }))
        msw.use(raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }))
        created.bodies.length = 0
        msw.use(
            created.handler,
            reply('get', '/auth/api-token-capabilities', capabilities),
            reply('get', '/sync-group', null),
        )
    },
} satisfies Meta<typeof CreateTokenDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
    play: async () => {
        const dialog = within(await screen.findByRole('dialog'))
        await expect(dialog.getByLabelText(/^name/i)).toBeVisible()
        // No name and no space yet.
        await expect(dialog.getByRole('button', { name: /^create$/i })).toBeDisabled()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const NeedsASpace: Story = {
    play: async () => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.type(dialog.getByLabelText(/^name/i), 'Dashboard')
        await expect(dialog.getByRole('button', { name: /^create$/i })).toBeDisabled()
        await expect(dialog.getByText(/choose at least one space/i)).toBeVisible()
    },
}

export const CreatedForOneSpace: Story = {
    play: async ({ args }) => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.type(dialog.getByLabelText(/^name/i), '  Dashboard ')
        await userEvent.click(dialog.getByRole('checkbox', { name: 'Trip' }))
        await userEvent.click(dialog.getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(args.onCreated).toHaveBeenCalledWith('sv_ab12_secret'))
        await expect(created.bodies[0]).toMatchObject({
            name: 'Dashboard',
            all_spaces: false,
            space_ids: [2],
            permissions: ['accounts:read', 'categories:read', 'transactions:read'],
            expires_at: null,
        })
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const AllSpaces: Story = {
    play: async ({ args }) => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.type(dialog.getByLabelText(/^name/i), 'Everything')
        await userEvent.click(dialog.getByRole('switch', { name: /all my spaces/i }))
        await userEvent.click(dialog.getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(args.onCreated).toHaveBeenCalled())
        await expect(created.bodies[0]).toMatchObject({ all_spaces: true, space_ids: [] })
    },
}

export const Cancel: Story = {
    play: async ({ args }) => {
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const Refused: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/api-tokens', 'Name already taken', 422)),
    play: async ({ args }) => {
        const dialog = within(await screen.findByRole('dialog'))
        await userEvent.type(dialog.getByLabelText(/^name/i), 'Dashboard')
        await userEvent.click(dialog.getByRole('switch', { name: /all my spaces/i }))
        await userEvent.click(dialog.getByRole('button', { name: /^create$/i }))
        await expect(await screen.findByText(/name already taken/i)).toBeVisible()
        await expect(args.onCreated).not.toHaveBeenCalled()
    },
}
