import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { raw, reply } from '@/test/api-mocks'
import { settings } from '@/test/fixtures'
import { space } from '@/test/app-state'
import { overlayA11y } from '@/test/a11y'
import { useAuthStore } from '@/stores/auth'
import { ConfirmCancelled, useConfirmStore } from '@/stores/confirm'
import { useSpaceStore } from '@/stores/space'

import { AbilityProvider, Can } from './AbilityProvider'
import { AuthProvider } from './AuthProvider'
import ConfirmDialog from './ConfirmDialog'
import { EmailVerificationGate } from './EmailVerificationGate'
import { ServerSetupGate } from './ServerSetupGate'
import { SessionExpiredDialog } from './SessionExpiredDialog'
import { NoSpace, SpaceGate } from './SpaceGate'

const user = { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin' as const, emailVerified: true }

const meta = {
    title: 'providers/Providers',
    parameters: { a11y: overlayA11y, layout: 'fullscreen' },
    afterEach: () => useConfirmStore.setState({ pending: null }),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const Page = () => <p>The app</p>
const Where = () => <p data-testid="where">{useLocation().pathname}</p>

// ---- asking for the password or a code

const ask = (mode: 'password' | 'code', run = fn().mockResolvedValue('done')) => {
    const pending = new Promise((resolve, reject) => useConfirmStore.setState({ pending: { mode, run, resolve, reject } }))
    pending.catch(() => {})
    return run
}

const typeAndSend = async (value: string) => {
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(/password|code/i), value)
    await userEvent.click(within(dialog).getByRole('button', { name: /^confirm$/i }))
}

export const ConfirmByPassword: Story = {
    render: () => <ConfirmDialog />,
    beforeEach: () => void ask('password'),
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByLabelText(/password/i)).toHaveAttribute('type', 'password')
        await expect(within(dialog).getByRole('button', { name: /^confirm$/i })).toBeDisabled()
    },
}

export const ConfirmByCode: Story = {
    render: () => <ConfirmDialog />,
    beforeEach: () => void ask('code'),
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByLabelText(/code/i)).toHaveAttribute('autocomplete', 'one-time-code')
    },
}

export const ConfirmNothingPending: Story = {
    render: () => <ConfirmDialog />,
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

export const ConfirmSent: Story = {
    render: () => <ConfirmDialog />,
    beforeEach: () => {
        useConfirmStore.setState({ pending: null })
    },
    play: async () => {
        const run = ask('password')
        await typeAndSend('hunter2')
        await waitFor(() => expect(run).toHaveBeenCalledWith({ current_password: 'hunter2' }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const ConfirmCodeSent: Story = {
    render: () => <ConfirmDialog />,
    play: async () => {
        const run = ask('code')
        await typeAndSend('123456')
        await waitFor(() => expect(run).toHaveBeenCalledWith({ code: '123456' }))
    },
}

// A wrong answer keeps the dialog open, shows the server's message and clears the field.
export const ConfirmWrongAnswer: Story = {
    render: () => <ConfirmDialog />,
    play: async () => {
        ask('password', fn().mockRejectedValue({ reason: 'confirmation_invalid', message: 'That password is wrong' }))
        await typeAndSend('nope')
        const dialog = await screen.findByRole('dialog')
        await expect(await within(dialog).findByRole('alert')).toHaveTextContent('That password is wrong')
        await expect(within(dialog).getByLabelText(/password/i)).toHaveValue('')
    },
}

export const ConfirmCancelledByUser: Story = {
    render: () => <ConfirmDialog />,
    play: async () => {
        const rejected = fn()
        useConfirmStore.setState({
            pending: { mode: 'password', run: fn(), resolve: fn(), reject: rejected },
        })
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(rejected).toHaveBeenCalledWith(expect.any(ConfirmCancelled)))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const ConfirmInRussian: Story = {
    ...ConfirmByPassword,
    globals: { locale: 'ru' },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByLabelText(/пароль/i)).toHaveAttribute('type', 'password')
    },
}

// ---- the session ended

export const SessionExpired: Story = {
    render: () => (
        <Routes>
            <Route path="/" element={<><SessionExpiredDialog /><Where /></>} />
            <Route path="/login" element={<Where />} />
        </Routes>
    ),
    beforeEach: () => useAuthStore.setState({ sessionExpired: true, isAuthenticated: true }),
    play: async () => {
        await expect(await screen.findByRole('alertdialog')).toBeVisible()
    },
}

export const SessionExpiredSignsInAgain: Story = {
    ...SessionExpired,
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /sign in/i }))
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/login'))
    },
}

export const SessionStillValid: Story = {
    render: () => <SessionExpiredDialog />,
    play: async () => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
    },
}

// ---- an address that is not confirmed yet

const unverified = () =>
    useAuthStore.setState({
        user: { ...user, emailVerified: false },
        isAuthenticated: true,
        emailVerificationRequired: true,
    })

export const EmailNotConfirmed: Story = {
    render: () => <EmailVerificationGate><Page /></EmailVerificationGate>,
    beforeEach: unverified,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/kirill@example\.com/)).toBeVisible()
        await expect(within(canvasElement).queryByText('The app')).toBeNull()
    },
}

export const EmailNotConfirmedInRussian: Story = { ...EmailNotConfirmed, globals: { locale: 'ru' } }

export const EmailNotRequired: Story = {
    render: () => <EmailVerificationGate><Page /></EmailVerificationGate>,
    beforeEach: () => useAuthStore.setState({ user: { ...user, emailVerified: false }, emailVerificationRequired: false }),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('The app')).toBeVisible()
    },
}

export const EmailConfirmedAlready: Story = {
    render: () => <EmailVerificationGate><Page /></EmailVerificationGate>,
    beforeEach: () => useAuthStore.setState({ user, emailVerificationRequired: true }),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('The app')).toBeVisible()
    },
}

export const ResendLink: Story = {
    ...EmailNotConfirmed,
    beforeEach: ({ msw }) => {
        unverified()
        msw.use(reply('post', '/auth/verify-email/resend', {}))
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /link again/i }))
        await expect(await screen.findByText(/confirmation email sent/i)).toBeVisible()
    },
}

// ---- the server's first-run setup

export const ServerSetupPending: Story = {
    render: () => (
        <Routes>
            <Route path="/" element={<ServerSetupGate><Page /></ServerSetupGate>} />
            <Route path="/setup/server" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => msw.use(reply('get', '/settings', { ...settings, onboarding_server_done: false })),
    play: async () => {
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/setup/server'))
    },
}

export const ServerSetupDone: Story = {
    render: () => <ServerSetupGate><Page /></ServerSetupGate>,
    beforeEach: ({ msw }) => msw.use(reply('get', '/settings', settings)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('The app')).toBeVisible()
    },
}

export const ServerSetupOnlyForAdmins: Story = {
    render: () => <ServerSetupGate><Page /></ServerSetupGate>,
    parameters: { app: { serverRole: 'user' } },
    beforeEach: ({ msw }) => msw.use(reply('get', '/settings', { ...settings, onboarding_server_done: false })),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('The app')).toBeVisible()
    },
}

export const ServerSetupLoading: Story = {
    render: () => <ServerSetupGate><Page /></ServerSetupGate>,
    beforeEach: ({ msw }) => {
        msw.use(raw('get', '/settings', {}))
    },
}

// ---- the space

export const SpaceKnown: Story = {
    render: () => <SpaceGate><Page /></SpaceGate>,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('The app')).toBeVisible()
    },
}

// A remembered space the user left falls back to their first.
export const SpaceForgottenFallsBack: Story = {
    render: () => <SpaceGate><Page /></SpaceGate>,
    beforeEach: () => {
        useSpaceStore.setState({ currentId: 999 })
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('The app')).toBeVisible()
        await expect(useSpaceStore.getState().currentId).toBe(1)
    },
}

export const NoSpaceYet: Story = {
    render: () => <SpaceGate><Page /></SpaceGate>,
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces', [])),
    play: async () => {
        await expect(await screen.findByRole('heading')).toBeVisible()
    },
}

export const NoSpaceForAGuest: Story = {
    render: () => <NoSpace />,
    parameters: { app: { serverRole: 'guest' } },
    play: async () => {
        await waitFor(() => expect(screen.queryByRole('button', { name: /create/i })).toBeNull())
    },
}

export const NoSpaceForAViewer: Story = {
    render: () => <NoSpace />,
    parameters: { app: { role: 'viewer' } },
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces', [space('viewer')])),
}

export const NoSpaceInRussian: Story = { ...NoSpaceYet, globals: { locale: 'ru' } }

// ---- the whole gate

export const SignedInAndReady: Story = {
    render: () => (
        <Routes>
            <Route element={<AuthProvider />}>
                <Route index element={<Page />} />
            </Route>
            <Route path="/login" element={<Where />} />
            <Route path="/setup" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: true })
        msw.use(
            raw('get', '/auth/status', { needs_registration: false }),
            raw('get', '/auth/me', { user, expires_at: null, refresh_at: null }),
            reply('get', '/settings', settings),
        )
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('The app', {}, { timeout: 5000 })).toBeVisible()
    },
}

export const SignedOutGoesToLogin: Story = {
    ...SignedInAndReady,
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: true })
        msw.use(
            raw('get', '/auth/status', { needs_registration: false }),
            raw('get', '/auth/me', { user: null }),
        )
    },
    play: async () => {
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/login'), { timeout: 5000 })
    },
}

export const FirstRunGoesToSetup: Story = {
    ...SignedInAndReady,
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: true })
        msw.use(raw('get', '/auth/status', { needs_registration: true }))
    },
    play: async () => {
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/setup'), { timeout: 5000 })
    },
}

// ---- what the server role may do

const Abilities = () => (
    <AbilityProvider>
        <Can I="manage" a="User">
            <p>Manages users</p>
        </Can>
        <Can I="read" a="User">
            <p>Reads users</p>
        </Can>
        <Can I="read" a="all">
            <p>Reads everything else</p>
        </Can>
    </AbilityProvider>
)

export const AdminManagesUsers: Story = {
    render: () => <Abilities />,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Manages users')).toBeVisible()
    },
}

export const MemberOnlyReadsUsers: Story = {
    parameters: { app: { serverRole: 'user' } },
    render: () => <Abilities />,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Reads users')).toBeVisible()
        await expect(within(canvasElement).queryByText('Manages users')).toBeNull()
    },
}

export const NobodyMayDoAnything: Story = {
    parameters: { app: { signedOut: true } },
    render: () => <Abilities />,
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).queryByText('Reads everything else')).toBeNull())
    },
}
