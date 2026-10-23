import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { hang, reply, settingsApi, spy } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'
import { settings } from '@/test/fixtures'
import type { Settings } from '@/types'

import { AuthenticationSection } from './AuthenticationSection'
import { ServerSection, SpacesLimitsSection } from './ServerAndSpacesSections'
import { NumberRow, TextRow } from './SettingRows'

const mail = (enabled: boolean, appUrlSet = true) =>
    reply('get', '/admin/mail', { enabled, problem: null, appUrlSet, writable: true })

const sso = (count: number) =>
    reply(
        'get',
        '/auth/sso/providers',
        Array.from({ length: count }, (_, i) => ({ slug: `p${i}`, name: 'GitHub', preset: 'github', protocol: 'oauth2' })),
    )

const sent: { bodies: unknown[] } = { bodies: [] }

const saving = (overrides: Partial<Settings> = {}) => {
    const update = spy('patch', '/settings', { ...settings, ...overrides })
    sent.bodies = update.bodies
    return update.handler
}

// The admin system page: authentication, the public URL and the limits of spaces.
const meta = {
    title: 'features/admin/System',
    beforeEach: ({ msw }) => msw.use(...settingsApi(), sso(1), mail(true)),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Authentication: Story = {
    render: () => <AuthenticationSection />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('switch', { name: /password sign-in/i })).toBeChecked()
        // The mail status arrives after the settings do.
        await waitFor(() => expect(canvas.getByRole('switch', { name: /require email confirmation/i })).toBeEnabled())
    },
}

export const AuthenticationInRussian: Story = { render: () => <AuthenticationSection />, globals: { locale: 'ru' } }
export const AuthenticationLoading: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: ({ msw }) => msw.use(hang('get', '/settings')),
}

// Without a single sign-on provider, turning passwords off would lock everyone out.
export const PasswordLoginLocked: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: ({ msw }) => msw.use(sso(0)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('switch', { name: /password sign-in/i })).toBeDisabled()
        await expect(canvas.getByText('Locked')).toBeVisible()
    },
}

export const PasswordLoginTurnedOff: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: ({ msw }) => msw.use(saving({ password_login_enabled: false })),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const passwordLogin = await canvas.findByRole('switch', { name: /password sign-in/i })
        // The switch stays locked until the single sign-on providers are known.
        await waitFor(() => expect(passwordLogin).toBeEnabled())
        await userEvent.click(passwordLogin)
        await waitFor(() => expect(sent.bodies).toEqual([{ password_login_enabled: false }]))
    },
}

// Without just-in-time sign-up there is nothing to verify.
export const VerifiedEmailNeedsSignUp: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: ({ msw }) => msw.use(...settingsApi({ sso_allow_signup: false })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('switch', { name: /verified email/i })).toBeDisabled()
    },
}

// Confirmation links go by email, so it waits for mail and the public URL.
export const ConfirmationNeedsMail: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: ({ msw }) => msw.use(mail(false)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/set up mail and the public url first/i)).toBeVisible()
        await expect(canvas.getByRole('switch', { name: /require email confirmation/i })).toBeDisabled()
    },
}

// The admin must confirm their own address first, or they would lock themselves out.
export const ConfirmationNeedsTheAdmin: Story = {
    render: () => <AuthenticationSection />,
    beforeEach: () => {
        const user = useAuthStore.getState().user!
        useAuthStore.setState({ user: { ...user, emailVerified: false } })
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/confirm your own email address first/i)).toBeVisible()
        await expect(canvas.getByRole('switch', { name: /require email confirmation/i })).toBeDisabled()
        await expect(canvas.getByRole('button', { name: /email me a confirmation link/i })).toBeVisible()
    },
}

export const Server: Story = {
    render: () => <ServerSection />,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByLabelText('Public URL')).toHaveValue('https://savvy.example.com')
    },
}

export const ServerInRussian: Story = { render: () => <ServerSection />, globals: { locale: 'ru' } }

export const ServerUrlSaved: Story = {
    render: () => <ServerSection />,
    beforeEach: ({ msw }) => msw.use(saving()),
    play: async ({ canvasElement }) => {
        const field = await within(canvasElement).findByLabelText('Public URL')
        await userEvent.clear(field)
        await userEvent.type(field, ' https://money.example.org ')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ app_url: 'https://money.example.org' }]))
    },
}

// The config file cannot be written, so the URL can only be read here.
export const ServerReadOnly: Story = {
    render: () => <ServerSection />,
    beforeEach: ({ msw }) => msw.use(...settingsApi({ config_writable: false })),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByLabelText('Public URL')).toHaveAttribute('readonly')
        await expect(canvas.getByText(/can't write to config\.toml/i)).toBeVisible()
    },
}

export const SpacesLimits: Story = {
    render: () => <SpacesLimitsSection />,
    beforeEach: ({ msw }) => msw.use(...settingsApi({ max_spaces_per_user: 3, space_quota_mb: 100 })),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByLabelText('Spaces per user')).toHaveValue(3)
        await expect(canvas.getByLabelText(/default size limit/i)).toHaveValue(100)
    },
}

export const SpacesLimitsInRussian: Story = { render: () => <SpacesLimitsSection />, globals: { locale: 'ru' } }

// Empty means no limit for spaces per user.
export const NoSpacesLimit: Story = {
    render: () => <SpacesLimitsSection />,
    beforeEach: ({ msw }) => msw.use(...settingsApi({ max_spaces_per_user: 3 }), saving()),
    play: async ({ canvasElement }) => {
        await userEvent.clear(await within(canvasElement).findByLabelText('Spaces per user'))
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ max_spaces_per_user: null }]))
    },
}

export const InvitesCreateAccounts: Story = {
    render: () => <SpacesLimitsSection />,
    beforeEach: ({ msw }) => msw.use(saving({ space_invites_can_register: true })),
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('switch', { name: /invitations create accounts/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ space_invites_can_register: true }]))
    },
}

const save = fn()

export const NumberRowRejectsNegative: Story = {
    render: () => <NumberRow id="n" label="Backups" description="How many to keep" value={5} onSave={save} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const field = canvas.getByLabelText('Backups')
        await userEvent.clear(field)
        await userEvent.type(field, '-1')
        await expect(canvas.getByRole('button', { name: /^save$/i })).toBeDisabled()
        // Empty is not allowed where the value cannot be null.
        await userEvent.clear(field)
        await expect(canvas.getByRole('button', { name: /^save$/i })).toBeDisabled()
    },
}

export const TextRowWithSuggestion: Story = {
    render: () => (
        <TextRow id="t" label="Public URL" description="Where Savvy is" value="" suggestion="https://here.example" onSave={save} />
    ),
    beforeEach: () => save.mockClear(),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /use https:\/\/here\.example/i }))
        await expect(canvas.getByLabelText('Public URL')).toHaveValue('https://here.example')
        // The suggestion only fills the field; saving stays explicit.
        await expect(save).not.toHaveBeenCalled()
        await userEvent.click(canvas.getByRole('button', { name: /^save$/i }))
        await expect(save).toHaveBeenCalledWith('https://here.example')
        await expect(screen.queryByRole('alert')).toBeNull()
    },
}
