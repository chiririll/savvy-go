import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, settingsApi } from '@/test/api-mocks'

import AdminSystemPage from './system'

const meta = {
    title: 'pages/admin/System',
    component: AdminSystemPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            ...settingsApi(),
            reply('get', '/auth/sso/providers', [{ slug: 'gh', name: 'GitHub', preset: 'github', protocol: 'oauth2' }]),
            reply('get', '/admin/mail', { enabled: true, problem: null, appUrlSet: true, writable: true }),
        ),
} satisfies Meta<typeof AdminSystemPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('switch', { name: /password sign-in/i })).toBeChecked()
        await expect(canvas.getByRole('heading', { level: 1 })).toBeVisible()
    },
}
