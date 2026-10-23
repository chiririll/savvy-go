import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import NotFoundPage from './not-found'

const meta = {
    title: 'pages/NotFound',
    component: NotFoundPage,
    parameters: { route: '/no/such/page' },
} satisfies Meta<typeof NotFoundPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('heading', { name: '404' })).toBeVisible()
        await expect(canvas.getByRole('heading', { name: /page not found/i })).toBeVisible()
        // The way out leads to the dashboard.
        await expect(canvas.getByRole('link', { name: /back to dashboard/i })).toHaveAttribute('href', '/')
    },
}

export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
