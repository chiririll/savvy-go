import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { fail, hang, reply } from '@/test/api-mocks'
import { resources, storage } from '@/components/features/monitoring/fixtures'

import AdminMonitoringPage from './monitoring'

const meta = {
    title: 'pages/admin/Monitoring',
    component: AdminMonitoringPage,
    beforeEach: ({ msw }) => msw.use(reply('get', '/monitoring/resources', resources()), reply('get', '/monitoring/storage', storage())),
} satisfies Meta<typeof AdminMonitoringPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 1 })).toBeVisible()
        await expect(await within(canvasElement).findByText(/1.4.0/)).toBeVisible()
    },
}
export const Loading: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/monitoring/resources'), hang('get', '/monitoring/storage')),
}
export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/monitoring/resources'), fail('get', '/monitoring/storage')),
}
