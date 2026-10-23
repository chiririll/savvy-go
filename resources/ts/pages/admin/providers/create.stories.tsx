import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, settingsApi } from '@/test/api-mocks'
import { presets } from '@/components/features/sso/fixtures'

import ProviderCreatePage from './create'

const meta = {
    title: 'pages/admin/ProviderCreate',
    component: ProviderCreatePage,
    parameters: { a11y: overlayA11y, route: '/admin/providers/create' },
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/presets', presets), ...settingsApi()),
} satisfies Meta<typeof ProviderCreatePage>

export default meta
type Story = StoryObj<typeof meta>

export const ChoosePreset: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /github/i })).toBeVisible()
        await expect(canvas.getByRole('link', { name: /back/i })).toHaveAttribute('href', '/admin/sso')
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }

export const FillsTheForm: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /github/i }))
        await expect(await canvas.findByLabelText(/client id/i)).toBeVisible()
        // The picked type can be changed again.
        await userEvent.click(canvas.getByRole('button', { name: /change/i }))
        await expect(await canvas.findByRole('button', { name: /keycloak/i })).toBeVisible()
    },
}
