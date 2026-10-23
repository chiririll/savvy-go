import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply, settingsApi } from '@/test/api-mocks'
import { presets } from '@/components/features/sso/fixtures'

import ServerSetupPage from './server'

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/onboarding/ServerSetup',
    component: ServerSetupPage,
    parameters: { a11y: overlayA11y, route: '/setup/server' },
    render: () => (
        <Routes>
            <Route path="/setup/server/*" element={<ServerSetupPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) =>
        msw.use(
            ...settingsApi({ onboarding_server_done: false, app_url: '' }),
            reply('get', '/admin/mail', { enabled: false, problem: null, appUrlSet: false, writable: true }),
            reply('get', '/identity-providers', []),
            reply('get', '/auth/sso/presets', presets),
        ),
} satisfies Meta<typeof ServerSetupPage>

export default meta
type Story = StoryObj<typeof meta>

export const AppUrl: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('textbox')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /continue/i })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/settings')) }

export const OptionalSteps: Story = {
    parameters: { route: '/setup/server/more' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /skip for now/i })).toBeVisible()
    },
}

export const Mail: Story = { parameters: { route: '/setup/server/mail' } }
export const SingleSignOn: Story = { parameters: { route: '/setup/server/sso' } }

export const UnknownStepGoesBack: Story = {
    parameters: { route: '/setup/server/nope' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('textbox')).toBeVisible()
    },
}

export const AlreadyDone: Story = {
    beforeEach: ({ msw }) => msw.use(...settingsApi({ onboarding_server_done: true })),
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByTestId('where')).toHaveTextContent(/^\/$/))
    },
}

export const NotAnAdmin: Story = {
    parameters: { app: { serverRole: 'user' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByTestId('where')).toHaveTextContent(/^\/$/))
    },
}

export const SkipLeavesForTwoFactor: Story = {
    parameters: { route: '/setup/server/more' },
    beforeEach: ({ msw }) => msw.use(reply('patch', '/settings', { onboarding_server_done: true })),
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /skip for now/i }))
        await waitFor(() => expect(within(canvasElement).getByTestId('where')).toHaveTextContent('/setup-2fa'))
    },
}
