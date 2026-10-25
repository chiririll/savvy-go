import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { fail, hang, raw, spy } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import { RecoverySection, TwoFactorSection } from './TwoFactorSections'
import { TwoFactorDialogs } from './TwoFactorDialogs'
import { useTwoFactorSetup } from './useTwoFactorSetup'

const status = (enabled: boolean, left: number | null = null) =>
    raw('get', '/auth/2fa/status', { enabled, pending_confirmation: false, recovery_codes_remaining: left })

const secret = { secret: 'JBSWY3DPEHPK3PXP', qr_code_url: 'otpauth://totp/GoSavvy:kirill?secret=JBSWY3DPEHPK3PXP' }
const codes = ['aaaa-1111', 'bbbb-2222', 'cccc-3333', 'dddd-4444']

// The page wires these three to one hook; so does the story.
function Panel() {
    const setup = useTwoFactorSetup()
    return (
        <div className="space-y-6">
            <TwoFactorSection setup={setup} />
            <RecoverySection setup={setup} />
            <TwoFactorDialogs setup={setup} />
        </div>
    )
}

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/security/TwoFactor',
    component: Panel,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        // A single-sign-on account has nothing to confirm with, so enabling goes straight through.
        const user = useAuthStore.getState().user
        if (user) useAuthStore.setState({ user: { ...user, isSsoOnly: true } })
        msw.use(status(false))
    },
} satisfies Meta<typeof Panel>

export default meta
type Story = StoryObj<typeof meta>

const toggle = (canvasElement: HTMLElement) => within(canvasElement).findByRole('switch')
const typeCode = async (code = '123456') => {
    await userEvent.click(await screen.findByRole('textbox'))
    await userEvent.keyboard(code)
}

export const Off: Story = {
    play: async ({ canvasElement }) => {
        await expect(await toggle(canvasElement)).not.toBeChecked()
        await expect(within(canvasElement).queryByText(/recovery/i, { selector: 'h2, h3' })).toBeNull()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/2fa/status')) }

export const On: Story = {
    beforeEach: ({ msw }) => msw.use(status(true, 8)),
    play: async ({ canvasElement }) => {
        await expect(await toggle(canvasElement)).toBeChecked()
        await expect(await within(canvasElement).findByRole('button', { name: /regenerate/i })).toBeVisible()
    },
}

export const OnWithOneCodeLeft: Story = { beforeEach: ({ msw }) => msw.use(status(true, 1)) }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    beforeEach: ({ msw }) => msw.use(status(true, 8)),
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the status.
        const sw = await toggle(canvasElement)
        await waitFor(() => expect(sw).toBeDisabled())
        await waitFor(() => expect(within(canvasElement).getByRole('button', { name: /regenerate/i })).toBeDisabled())
    },
}

// ---- turning it on

export const EnableShowsTheQrCode: Story = {
    beforeEach: ({ msw }) => msw.use(raw('post', '/auth/2fa/enable', secret)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        const dialog = within(await screen.findByRole('dialog'))
        await expect(await dialog.findByText(secret.secret)).toBeVisible()
    },
}

export const EnableAsksForTheCode: Story = {
    beforeEach: ({ msw }) => msw.use(raw('post', '/auth/2fa/enable', secret)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /continue/i }))
        await expect(await screen.findByRole('textbox')).toBeVisible()
        await expect(screen.getByRole('button', { name: /^verify/i })).toBeDisabled()
    },
}

export const EnableGoesBackToTheQrCode: Story = {
    beforeEach: ({ msw }) => msw.use(raw('post', '/auth/2fa/enable', secret)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /continue/i }))
        await userEvent.click(await screen.findByRole('button', { name: /back/i }))
        await expect(await screen.findByText(secret.secret)).toBeVisible()
    },
}

export const EnableFinishesWithRecoveryCodes: Story = {
    beforeEach: ({ msw }) => {
        const confirm = spy('post', '/auth/2fa/confirm', { message: 'ok', recovery_codes: codes }, 200, false)
        sent.bodies = confirm.bodies
        msw.use(raw('post', '/auth/2fa/enable', secret), confirm.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /continue/i }))
        await typeCode()
        await userEvent.click(screen.getByRole('button', { name: /^verify/i }))
        await expect(await screen.findByText(codes[0])).toBeVisible()
        await expect(sent.bodies).toEqual([{ code: '123456' }])
        await userEvent.click(screen.getByRole('button', { name: /done/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const EnableRefused: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/2fa/enable', 'Could not start', 500)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

// ---- turning it off

export const DisableAsksForTheCode: Story = {
    beforeEach: ({ msw }) => msw.use(status(true, 8)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await expect(await screen.findByRole('textbox')).toBeVisible()
        await expect(screen.getByRole('button', { name: /^disable/i })).toBeDisabled()
    },
}

export const DisableCancelled: Story = {
    beforeEach: ({ msw }) => msw.use(status(true, 8)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const Disabled: Story = {
    beforeEach: ({ msw }) => {
        const disable = spy('post', '/auth/2fa/disable', { message: 'ok' }, 200, false)
        sent.bodies = disable.bodies
        msw.use(status(true, 8), disable.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await toggle(canvasElement))
        await typeCode('654321')
        await userEvent.click(screen.getByRole('button', { name: /^disable/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ code: '654321' }]))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

// ---- new recovery codes

export const RegenerateAsksForTheCode: Story = {
    beforeEach: ({ msw }) => msw.use(status(true, 2)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /regenerate/i }))
        await expect(await screen.findByRole('textbox')).toBeVisible()
    },
}

export const Regenerated: Story = {
    beforeEach: ({ msw }) => {
        msw.use(status(true, 2), raw('post', '/auth/2fa/recovery-codes/regenerate', { message: 'ok', recovery_codes: codes }))
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /regenerate/i }))
        await typeCode()
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /regenerate/i }))
        await expect(await screen.findByText(codes[1])).toBeVisible()
    },
}
