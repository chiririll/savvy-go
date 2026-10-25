import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'

import SystemBackupsPage from './backups'

const meta = {
    title: 'pages/admin/Backups',
    component: SystemBackupsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/backups', [
                {
                    filename: 'savvy-2030-01-05.zip',
                    size: 2_345_678,
                    note: 'Before the upgrade',
                    appVersion: '1.4.0',
                    kind: 'server',
                    status: 'current',
                    signature: 'own',
                    restorable: true,
                    createdAt: '2030-01-05T10:00:00Z',
                },
            ]),
        ),
} satisfies Meta<typeof SystemBackupsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Before the upgrade')).toBeVisible()
    },
}
export const Empty: Story = { beforeEach: ({ msw }) => msw.use(reply('get', '/backups', [])) }
