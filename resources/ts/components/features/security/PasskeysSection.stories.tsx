import type { Meta, StoryObj } from '@storybook/react-vite'
import { http, HttpResponse } from 'msw'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import ConfirmDialog from '@/components/providers/ConfirmDialog'
import { overlayA11y } from '@/test/a11y'
import { fail, hang, raw } from '@/test/api-mocks'
import { useConfirmStore } from '@/stores/confirm'

import { PasskeysSection } from './PasskeysSection'

const passkeys = [
    { id: 1, name: 'MacBook Touch ID', createdAt: '2030-01-05T10:00:00Z', lastUsedAt: '2030-02-01T08:00:00Z' },
    { id: 2, name: null, createdAt: '2030-01-10T10:00:00Z', lastUsedAt: null },
]

const credentials = (list: unknown[]) => raw('get', '/auth/webauthn/credentials', { credentials: list })

const sent: { bodies: unknown[] } = { bodies: [] }

// The confirmation dialog sits next to the section, as the app's providers mount it.
const meta = {
    title: 'features/security/PasskeysSection',
    component: PasskeysSection,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        msw.use(
            credentials(passkeys),
            raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }),
        )
    },
    afterEach: () => useConfirmStore.setState({ pending: null }),
    render: () => (
        <>
            <PasskeysSection />
            <ConfirmDialog />
        </>
    ),
} satisfies Meta<typeof PasskeysSection>

export default meta
type Story = StoryObj<typeof meta>

export const Registered: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('MacBook Touch ID')).toBeVisible()
        await expect(canvas.getByText('Unnamed passkey')).toBeVisible()
        await expect(canvas.getByText(/never used/i)).toBeVisible()
        await expect(canvas.getAllByRole('button', { name: /remove passkey/i })).toHaveLength(2)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/webauthn/credentials')) }

export const None: Story = {
    beforeEach: ({ msw }) => msw.use(credentials([])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no passkeys registered yet/i)).toBeVisible()
    },
}

// A viewer of the space still owns their account, but the read-only mode locks the buttons.
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByText('MacBook Touch ID')
        await expect(canvas.getByRole('button', { name: /add passkey/i })).toBeDisabled()
    },
}

export const AddAsksForAName: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /add passkey/i }))
        const dialog = within(await screen.findByRole('dialog'))
        await expect(dialog.getByPlaceholderText(/macbook touch id/i)).toHaveFocus()
        await userEvent.click(dialog.getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

// Adding asks for the password first; a refusal from the server is toasted and the dialog stays.
export const AddFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/webauthn/register/options', 'Passkeys are off on this server')),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /add passkey/i }))
        await userEvent.type(await screen.findByPlaceholderText(/macbook touch id/i), 'YubiKey{Enter}')
        await userEvent.type(await screen.findByLabelText(/password/i), 'secret{Enter}')
        await expect(await screen.findByText(/passkeys are off on this server/i)).toBeVisible()
    },
}

export const RemoveAsksFirst: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /remove passkey/i }))[0])
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/no longer be able to sign in/i)).toBeVisible()
        await userEvent.click(dialog.getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull())
    },
}

// Removing is confirmed with the password, which travels with the request.
export const Removed: Story = {
    beforeEach: ({ msw }) => {
        sent.bodies = []
        msw.use(
            http.delete(/\/api\/auth\/webauthn\/credentials\/1$/, async ({ request }) => {
                sent.bodies.push(await request.json())
                return new HttpResponse(null, { status: 204 })
            }),
        )
    },
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /remove passkey/i }))[0])
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await userEvent.type(await screen.findByLabelText(/password/i), 'secret{Enter}')
        await waitFor(() => expect(sent.bodies).toEqual([{ current_password: 'secret' }]))
        await expect(await screen.findByText('Passkey removed')).toBeVisible()
    },
}
