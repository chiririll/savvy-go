import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { raw, fail } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import { AuthCardShell } from './AuthCardShell'
import { PasswordLoginForm } from './PasswordLoginForm'
import { SetPasswordForm } from './SetPasswordForm'
import { TwoFactorStep } from './TwoFactorStep'

const session = {
    user: { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', emailVerified: true },
    expires_at: '2099-01-01T00:00:00Z',
    refresh_at: null,
}

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

// What the screens call back with, cleared before each story.
const onTwoFactor = fn()
const onSubmitPassword = fn()
const onBack = fn()

const meta = {
    title: 'features/login/Login',
    parameters: { layout: 'fullscreen' },
    beforeEach: () => {
        onTwoFactor.mockClear()
        onSubmitPassword.mockClear()
        onBack.mockClear()
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

// ---- the card both screens sit in

export const Shell: Story = {
    render: () => (
        <AuthCardShell title="Sign in" description="Welcome back">
            <p>Body</p>
        </AuthCardShell>
    ),
}

export const ShellCompact: Story = {
    render: () => (
        <AuthCardShell title="Set up" description="A longer screen" compact className="max-w-2xl">
            <p>Body</p>
        </AuthCardShell>
    ),
}

export const ShellInRussian: Story = { ...Shell, globals: { locale: 'ru' } }

// ---- the password form

const loginPage = () => (
    <Routes>
        <Route
            path="/"
            element={
                <AuthCardShell title="Sign in" description="Welcome back">
                    <PasswordLoginForm afterLoginPath="/home" onTwoFactor={onTwoFactor} />
                </AuthCardShell>
            }
        />
        <Route path="/home" element={<Where />} />
    </Routes>
)

const fill = async (email: string, password: string) => {
    await userEvent.type(await screen.findByLabelText(/email/i), email)
    await userEvent.type(screen.getByLabelText(/^password/i), password)
}

export const LoginEmpty: Story = { render: () => loginPage() }
export const LoginInRussian: Story = { render: () => loginPage(), globals: { locale: 'ru' } }

export const LoginRemembersByDefault: Story = {
    render: () => loginPage(),
    play: async () => {
        await expect(await screen.findByRole('checkbox', { name: /remember/i })).toBeChecked()
    },
}

export const LoginRejectsABadAddress: Story = {
    render: () => loginPage(),
    play: async () => {
        await fill('not-an-address', 'secret')
        await userEvent.click(screen.getByRole('button', { name: /sign in/i }))
        // The browser's own email check stops the submit before the schema runs.
        await expect(screen.getByLabelText(/email/i)).toBeInvalid()
        await expect(screen.queryByTestId('where')).toBeNull()
    },
}

export const LoginNeedsAPassword: Story = {
    render: () => loginPage(),
    play: async () => {
        await userEvent.type(await screen.findByLabelText(/email/i), 'kirill@example.com')
        await userEvent.click(screen.getByRole('button', { name: /sign in/i }))
        await expect(await screen.findByText(/password/i, { selector: 'p' })).toBeVisible()
    },
}

export const LoginSucceeds: Story = {
    render: () => loginPage(),
    beforeEach: ({ msw }) => {
        msw.use(raw('post', '/auth/login', session))
    },
    play: async () => {
        await fill('kirill@example.com', 'secret')
        await userEvent.click(screen.getByRole('button', { name: /sign in/i }))
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/home'))
    },
}

export const LoginWithTheWrongPassword: Story = {
    render: () => loginPage(),
    beforeEach: ({ msw }) => {
        msw.use(fail('post', '/auth/login', 'Invalid credentials', 401))
    },
    play: async () => {
        await fill('kirill@example.com', 'wrong')
        await userEvent.click(screen.getByRole('button', { name: /sign in/i }))
        await expect(await screen.findByText(/invalid credentials/i)).toBeVisible()
        await expect(screen.queryByTestId('where')).toBeNull()
    },
}

export const LoginAsksForASecondFactor: Story = {
    render: () => loginPage(),
    beforeEach: ({ msw }) => {
        msw.use(raw('post', '/auth/login', { requires_2fa: true, two_factor_token: 'tok' }))
    },
    play: async () => {
        await fill('kirill@example.com', 'secret')
        await userEvent.click(screen.getByRole('button', { name: /sign in/i }))
        await waitFor(() => expect(onTwoFactor).toHaveBeenCalledWith('tok', true))
    },
}

// ---- choosing a new password

const setPassword = (submitting = false) => (
    <AuthCardShell title="New password" description="Choose one">
        <SetPasswordForm submitting={submitting} onSubmit={onSubmitPassword} />
    </AuthCardShell>
)

export const NewPassword: Story = { render: () => setPassword() }
export const NewPasswordSubmitting: Story = {
    render: () => setPassword(true),
    play: async () => {
        await expect(await screen.findByRole('button', { name: /save password/i })).toBeDisabled()
    },
}
export const NewPasswordInRussian: Story = { render: () => setPassword(), globals: { locale: 'ru' } }

export const NewPasswordTooShort: Story = {
    render: () => setPassword(),
    play: async () => {
        const [password] = await screen.findAllByPlaceholderText(/./)
        await userEvent.type(password, 'short')
        await userEvent.click(screen.getByRole('button', { name: /save password/i }))
        await expect(await screen.findByText(/at least 8/i)).toBeVisible()
    },
}

export const NewPasswordsMustMatch: Story = {
    render: () => setPassword(),
    play: async () => {
        const fields = await screen.findAllByPlaceholderText(/./)
        await userEvent.type(fields[0], 'a-long-enough-one')
        await userEvent.type(fields[1], 'a-different-one!')
        await userEvent.click(screen.getByRole('button', { name: /save password/i }))
        await expect(await screen.findByText(/don't match/i)).toBeVisible()
    },
}

export const NewPasswordSubmitted: Story = {
    render: () => setPassword(),
    play: async () => {
        const fields = await screen.findAllByPlaceholderText(/./)
        await userEvent.type(fields[0], 'a-long-enough-one')
        await userEvent.type(fields[1], 'a-long-enough-one')
        await userEvent.click(screen.getByRole('button', { name: /save password/i }))
        await waitFor(() => expect(onSubmitPassword).toHaveBeenCalled())
        await expect(onSubmitPassword.mock.calls[0][0]).toEqual({
            password: 'a-long-enough-one',
            password_confirmation: 'a-long-enough-one',
        })
    },
}

// ---- the second factor

const twoFactor = () => (
    <Routes>
        <Route path="/" element={<TwoFactorStep token="tok" rememberMe afterLoginPath="/home" onBack={onBack} />} />
        <Route path="/home" element={<Where />} />
    </Routes>
)

export const SecondFactor: Story = {
    render: () => twoFactor(),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /verify/i })).toBeDisabled()
    },
}

export const SecondFactorInRussian: Story = { render: () => twoFactor(), globals: { locale: 'ru' } }

export const SecondFactorByRecoveryCode: Story = {
    render: () => twoFactor(),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /recovery/i }))
        await expect(within(canvasElement).getByRole('textbox')).toBeVisible()
        await userEvent.type(within(canvasElement).getByRole('textbox'), 'abcd-efgh')
        await expect(within(canvasElement).getByRole('button', { name: /verify/i })).toBeEnabled()
    },
}

export const SecondFactorBackToTheCode: Story = {
    render: () => twoFactor(),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /recovery/i }))
        await userEvent.click(within(canvasElement).getByRole('button', { name: /authenticator/i }))
        await expect(within(canvasElement).getByRole('textbox', { name: /verification code/i })).toBeVisible()
    },
}

export const SecondFactorLeftBehind: Story = {
    render: () => twoFactor(),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /back/i }))
        await expect(onBack).toHaveBeenCalled()
    },
}

export const SecondFactorAccepted: Story = {
    render: () => twoFactor(),
    beforeEach: ({ msw }) => {
        msw.use(raw('post', '/auth/2fa/verify', session))
    },
    play: async ({ canvasElement }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox', { name: /verification code/i }), '123456')
        await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/home'))
    },
}

export const SecondFactorRefused: Story = {
    render: () => twoFactor(),
    beforeEach: ({ msw }) => {
        msw.use(fail('post', '/auth/2fa/verify', 'Invalid verification code', 422))
    },
    play: async ({ canvasElement }) => {
        const code = within(canvasElement).getByRole('textbox', { name: /verification code/i })
        await userEvent.type(code, '123456')
        await expect(await screen.findByText(/invalid verification code/i)).toBeVisible()
        await waitFor(() => expect(code).toHaveValue(''))
    },
}
