import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'

import SpaceBackupsPage from './backups'

const backup = {
    filename: 'home-2030-01-05.zip',
    size: 2_345_678,
    note: 'Before the move',
    appVersion: '1.4.0',
    kind: 'space',
    status: 'current',
    signature: 'own',
    restorable: true,
    createdAt: '2030-01-05T10:00:00Z',
}

const meta = {
    title: 'pages/settings/SpaceBackups',
    component: SpaceBackupsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces/1/backups', [backup])),
} satisfies Meta<typeof SpaceBackupsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Before the move')).toBeVisible()
        // The name of the space is in the title, and the admin may import a whole space.
        await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Home')
        await expect(canvas.getByRole('button', { name: /import/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }

export const AdminsOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByText(/only .*admin/i)).toBeVisible())
    },
}
