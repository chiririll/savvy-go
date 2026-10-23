import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'

import AdminSpacesPage from './spaces'

const MB = 1024 * 1024

const space = (id: number, name: string, overrides = {}) => ({
    id,
    uuid: `00000000-0000-4000-8000-00000000000${id}`,
    name,
    emoji: '🏡',
    color: null,
    members: 3,
    admins: 1,
    size: 4 * MB,
    quota: 0,
    unavailable: '',
    createdAt: '2030-01-01T00:00:00Z',
    ...overrides,
})

const meta = {
    title: 'pages/admin/Spaces',
    component: AdminSpacesPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/admin/spaces', [space(1, 'Family'), space(2, 'Broken', { unavailable: 'file is missing' })]),
            reply('get', '/admin/deleted-spaces', []),
            reply('get', '/users', [{ id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', emailVerified: true }]),
        ),
} satisfies Meta<typeof AdminSpacesPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Family')).toBeVisible()
        await expect(canvas.getByText('file is missing')).toBeVisible()
    },
}
export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
