import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, settingsApi } from '@/test/api-mocks'

import { presets } from '../sso/fixtures'
import { AppUrlStep } from './AppUrlStep'
import { OnboardingLayout } from './OnboardingLayout'
import { ServerMailStep } from './ServerMailStep'
import { ServerOptions } from './ServerOptions'
import { ServerSsoStep } from './ServerSsoStep'
import { serverSetupPaths } from './server-setup-paths'

const onContinue = fn()
const onFinish = fn()

const mail = (enabled: boolean) =>
    reply('get', '/admin/mail', {
        enabled,
        problem: null,
        host: enabled ? 'smtp.example.com' : '',
        port: 587,
        protocol: 'starttls',
        user: '',
        from: '',
        skipVerify: false,
        timeout: 10,
        passwordSet: false,
        appUrlSet: true,
        writable: true,
    })

const providers = (count: number) =>
    reply('get', '/identity-providers', Array.from({ length: count }, (_, i) => ({ id: i + 1, name: 'GitHub' })))

const card = (node: React.ReactNode) => (
    <OnboardingLayout title="Server setup" description="How this server is reached and what it can do.">
        {node}
    </OnboardingLayout>
)

// The server setup after the first admin: the public URL, then the optional steps.
const meta = {
    title: 'features/onboarding/ServerSetup',
    parameters: { layout: 'fullscreen', a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        onContinue.mockClear()
        onFinish.mockClear()
        msw.use(mail(false), providers(0), reply('get', '/auth/sso/presets', presets), ...settingsApi())
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const AppUrl: Story = {
    render: () => card(<AppUrlStep value="" writable saving={false} onContinue={onContinue} />),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        // Nothing typed: nothing to continue with.
        await expect(canvas.getByRole('button', { name: /continue/i })).toBeDisabled()
        await userEvent.click(canvas.getByRole('button', { name: /use http/i }))
        await expect(canvas.getByLabelText(/public url/i)).toHaveValue(window.location.origin)
        await userEvent.click(canvas.getByRole('button', { name: /continue/i }))
        await expect(onContinue).toHaveBeenCalledWith(window.location.origin)
    },
}

// The URL saved before is kept: continuing saves nothing.
export const AppUrlUnchanged: Story = {
    render: () => card(<AppUrlStep value="https://savvy.example.com" writable saving={false} onContinue={onContinue} />),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /continue/i }))
        await expect(onContinue).toHaveBeenCalledWith(null)
    },
}

export const AppUrlTrimmed: Story = {
    render: () => card(<AppUrlStep value="" writable saving={false} onContinue={onContinue} />),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByLabelText(/public url/i), '  https://money.example.org  ')
        await userEvent.click(canvas.getByRole('button', { name: /continue/i }))
        await expect(onContinue).toHaveBeenCalledWith('https://money.example.org')
    },
}

// config.toml cannot be written: the URL is set in the file.
export const AppUrlReadOnly: Story = {
    render: () => card(<AppUrlStep value="" writable={false} saving={false} onContinue={onContinue} />),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByRole('textbox')).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /continue/i }))
        await expect(onContinue).toHaveBeenCalledWith(null)
    },
}

export const AppUrlSaving: Story = {
    render: () => card(<AppUrlStep value="" writable saving onContinue={onContinue} />),
}

export const Options: Story = {
    render: () => card(<ServerOptions busy={false} onFinish={onFinish} />),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('link', { name: /email/i })).toHaveAttribute('href', serverSetupPaths.mail)
        // Nothing set up yet: the button skips.
        await userEvent.click(await canvas.findByRole('button', { name: /skip/i }))
        await expect(onFinish).toHaveBeenCalled()
    },
}

export const OptionsDone: Story = {
    render: () => card(<ServerOptions busy={false} onFinish={onFinish} />),
    beforeEach: ({ msw }) => msw.use(mail(true), providers(1)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getAllByText(/done/i)).toHaveLength(2))
        await expect(canvas.getByRole('button', { name: /finish/i })).toBeVisible()
    },
}

export const OptionsFinishing: Story = {
    render: () => card(<ServerOptions busy onFinish={onFinish} />),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByRole('button')[0]).toBeDisabled()
    },
}

export const Mail: Story = {
    render: () => card(<ServerMailStep />),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('switch')).toBeVisible()
        await expect(within(canvasElement).getByRole('link', { name: /back/i })).toHaveAttribute(
            'href',
            serverSetupPaths.options,
        )
    },
}

export const Sso: Story = {
    render: () => card(<ServerSsoStep />),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /github/i }))
        await expect(await canvas.findByText(/continue with/i)).toBeVisible()
        // The banner goes back to the presets.
        await userEvent.click(canvas.getByRole('button', { name: /change/i }))
        await expect(await screen.findByRole('button', { name: /keycloak/i })).toBeVisible()
    },
}
