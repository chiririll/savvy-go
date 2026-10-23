import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'

import AdminChecksPage from './checks'

const meta = {
    title: 'pages/admin/Checks',
    component: AdminChecksPage,
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/admin/checks', [
                { id: 'backup_stale', severity: 'error', value: 12 },
                { id: 'disk_filling', severity: 'warning', value: 91 },
            ]),
        ),
} satisfies Meta<typeof AdminChecksPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('heading', { level: 1 })).toBeVisible()
        await expect(await canvas.findByText('Last server backup is 12 days old')).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
