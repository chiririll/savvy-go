import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import type { ApiTokenSummary } from '@/types'

import { TokenRow } from './TokenRow'

const token = (overrides: Partial<ApiTokenSummary> = {}): ApiTokenSummary => ({
    id: 1,
    name: 'Home dashboard',
    prefix: 'sv_ab12',
    kind: 'personal',
    permissions: ['transactions:read', 'accounts:read'],
    all_spaces: true,
    space_ids: [],
    expires_at: null,
    last_used_at: '2030-01-05T10:00:00Z',
    created_at: '2029-12-01T10:00:00Z',
    ...overrides,
})

const meta = {
    title: 'features/api-tokens/TokenRow',
    component: TokenRow,
    parameters: { a11y: overlayA11y },
    args: { token: token(), revoking: false, onEdit: fn(), onRevoke: fn() },
} satisfies Meta<typeof TokenRow>

export default meta
type Story = StoryObj<typeof meta>

export const ReadOnlyToken: Story = {}
export const NeverUsed: Story = { args: { token: token({ last_used_at: null }) } }
export const Expiring: Story = { args: { token: token({ expires_at: '2031-06-01T00:00:00Z' }) } }
export const SomeSpaces: Story = { args: { token: token({ all_spaces: false, space_ids: [1, 2, 3] }) } }
export const ReadsAndWrites: Story = {
    args: { token: token({ permissions: ['transactions:write', 'accounts:read', 'budgets:delete'] }) },
}
export const NoAccess: Story = { args: { token: token({ permissions: [] }) } }
export const Revoking: Story = {
    args: { revoking: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /revoke/i })).toBeDisabled()
    },
}
export const LongName: Story = {
    args: { token: token({ name: 'A token name that is far too long to fit next to its badge and prefix on a phone' }) },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /change what/i }))
        await expect(args.onEdit).toHaveBeenCalled()
    },
}

export const RevokeAsksFirst: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /revoke/i }))
        const dialog = await screen.findByRole('alertdialog')
        await expect(within(dialog).getByText(/Home dashboard/)).toBeVisible()
        await expect(args.onRevoke).not.toHaveBeenCalled()
    },
}

export const Revoked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /revoke/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /^revoke$/i }))
        await expect(args.onRevoke).toHaveBeenCalled()
    },
}

export const RevokeCancelled: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /revoke/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onRevoke).not.toHaveBeenCalled()
    },
}
