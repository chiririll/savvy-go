import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { UserAvatarView } from './UserAvatar'

const GREY_SQUARE =
    'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160"><rect width="160" height="160" fill="%23888"/></svg>'

// The real UserAvatar fetches its picture from Gravatar; these stories draw the view alone, so they need no network.
const meta = {
    tags: ['desktop-only'],
    title: 'shared/appearance/UserAvatar',
    component: UserAvatarView,
    args: { user: { email: 'kirill@example.com', name: 'Kirill' }, className: 'size-12' },
} satisfies Meta<typeof UserAvatarView>

export default meta
type Story = StoryObj<typeof meta>

export const WithImage: Story = {
    args: { src: GREY_SQUARE },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('img')).toBeVisible()
    },
}

export const InitialOfName: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('K')).toBeVisible()
    },
}

export const InitialOfEmail: Story = {
    args: { user: { email: 'zed@example.com' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Z')).toBeVisible()
    },
}
