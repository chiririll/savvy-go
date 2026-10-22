import type { Meta, StoryObj } from '@storybook/react-vite'
import { http, HttpResponse } from 'msw'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, spy } from '@/test/api-mocks'
import type { OAuthClient } from '@/api/oauth'

import { McpAccess } from './McpAccess'
import { SigningKeys } from './SigningKeys'

const app = (overrides: Partial<OAuthClient> = {}): OAuthClient => ({
    client_id: 'c1',
    name: 'Claude',
    redirect_uris: ['https://claude.ai/api/mcp/auth_callback'],
    ip: '203.0.113.7',
    user_agent: 'Claude-User',
    origin: '',
    registered_at: '2030-01-05T10:00:00Z',
    ...overrides,
})

const keys = {
    own: { kid: 'k-own', publicKey: 'MCowBQYDK2VwAyEAown' },
    trusted: [{ kid: 'k-friend', publicKey: 'MCowBQYDK2VwAyEAfriend', name: 'Parents server', createdAt: null }],
}

const registration = (open: boolean) =>
    reply('get', '/admin/oauth/registration', { open, expires_at: open ? '2030-01-05T10:05:00Z' : null })

const sent: { bodies: unknown[] } = { bodies: [] }

const copyable = () => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: fn(async () => {}) }, configurable: true })
}

// The admin security page: AI app registration and the server's signing keys.
const meta = {
    title: 'features/admin/Security',
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(registration(false), reply('get', '/admin/oauth/clients', [app()]), reply('get', '/admin/keys', keys)),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const McpClosed: Story = {
    render: () => <McpAccess />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/closed\. no new app can register/i)).toBeVisible()
        await expect(await canvas.findByText('203.0.113.7')).toBeVisible()
        // What the app did not send says so.
        await expect(canvas.getByText('not sent')).toBeVisible()
        await expect(canvas.getByDisplayValue(`${window.location.origin}/mcp`)).toBeVisible()
    },
}

export const McpInRussian: Story = { render: () => <McpAccess />, globals: { locale: 'ru' } }

export const McpOpened: Story = {
    render: () => <McpAccess />,
    beforeEach: ({ msw }) => {
        let open = false
        msw.use(
            http.get(/\/api\/admin\/oauth\/registration$/, () =>
                HttpResponse.json({ data: { open, expires_at: open ? '2030-01-05T10:05:00Z' : null } }),
            ),
            http.post(/\/api\/admin\/oauth\/registration$/, () => {
                open = true
                return HttpResponse.json({ data: { open, expires_at: '2030-01-05T10:05:00Z' } })
            }),
        )
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /allow a new app/i }))
        await expect(await canvas.findByText(/open until/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /close now/i })).toBeVisible()
    },
}

export const McpNoApps: Story = {
    render: () => <McpAccess />,
    beforeEach: ({ msw }) => msw.use(registration(true), reply('get', '/admin/oauth/clients', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no app has registered yet/i)).toBeVisible()
    },
}

export const McpAddressCopied: Story = {
    render: () => <McpAccess />,
    beforeEach: copyable,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^copy$/i }))
        await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`${window.location.origin}/mcp`))
    },
}

export const McpAppDisconnected: Story = {
    render: () => <McpAccess />,
    beforeEach: ({ msw }) => {
        const remove = spy('delete', '/admin/oauth/clients/c1')
        sent.bodies = remove.bodies
        msw.use(remove.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /disconnect/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText('Remove Claude?')).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /disconnect/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

export const Keys: Story = {
    render: () => <SigningKeys />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByDisplayValue('MCowBQYDK2VwAyEAown')).toBeVisible()
        await expect(canvas.getByText(/key id k-own/i)).toBeVisible()
        await expect(canvas.getByText('Parents server')).toBeVisible()
    },
}

export const KeysInRussian: Story = { render: () => <SigningKeys />, globals: { locale: 'ru' } }

export const KeysNoneTrusted: Story = {
    render: () => <SigningKeys />,
    beforeEach: ({ msw }) => msw.use(reply('get', '/admin/keys', { ...keys, trusted: [] })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no other keys are trusted/i)).toBeVisible()
    },
}

export const KeyTrusted: Story = {
    render: () => <SigningKeys />,
    beforeEach: ({ msw }) => {
        const trust = spy('post', '/admin/keys', keys)
        sent.bodies = trust.bodies
        msw.use(trust.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const button = canvas.getByRole('button', { name: /^trust$/i })
        await expect(button).toBeDisabled()
        await userEvent.type(canvas.getByPlaceholderText('Name'), ' Work server ')
        await userEvent.type(canvas.getByPlaceholderText(/public key/i), ' MCowWORK ')
        await userEvent.click(button)
        await waitFor(() => expect(sent.bodies).toEqual([{ name: 'Work server', public_key: 'MCowWORK' }]))
        await waitFor(() => expect(canvas.getByPlaceholderText('Name')).toHaveValue(''))
    },
}

export const KeyUntrusted: Story = {
    render: () => <SigningKeys />,
    beforeEach: ({ msw }) => {
        const untrust = spy('delete', '/admin/keys/k-friend')
        sent.bodies = untrust.bodies
        msw.use(untrust.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /^remove$/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
    },
}

export const KeyRotated: Story = {
    render: () => <SigningKeys />,
    beforeEach: ({ msw }) => {
        const rotate = spy('post', '/admin/keys/rotate', keys)
        sent.bodies = rotate.bodies
        msw.use(rotate.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /rotate key/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/old public key stays trusted/i)).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /rotate key/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText('Key rotated')).toBeVisible()
    },
}

export const KeyCopied: Story = {
    render: () => <SigningKeys />,
    beforeEach: copyable,
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('MCowBQYDK2VwAyEAown')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^copy$/i }))
        await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('MCowBQYDK2VwAyEAown'))
    },
}
