import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes } from 'react-router-dom'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply, settingsApi } from '@/test/api-mocks'
import { presets, provider } from '@/components/features/sso/fixtures'

import ProviderEditPage from './edit'

const meta = {
    title: 'pages/admin/ProviderEdit',
    component: ProviderEditPage,
    parameters: { a11y: overlayA11y, route: '/admin/providers/1/edit' },
    // The id comes from the address.
    render: () => (
        <Routes>
            <Route path="/admin/providers/:id/edit" element={<ProviderEditPage />} />
        </Routes>
    ),
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/identity-providers/1', provider()), reply('get', '/auth/sso/presets', presets), ...settingsApi()),
} satisfies Meta<typeof ProviderEditPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByDisplayValue('Company GitHub')).toBeVisible()
        await expect(await within(canvasElement).findByDisplayValue('abc')).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/identity-providers/1')) }
