import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply, settingsApi } from '@/test/api-mocks'

import UsersPage from './users'

const user = (id: number, name: string, role: 'admin' | 'user' | 'guest', overrides = {}) => ({
    id,
    name,
    email: `${name.toLowerCase()}@example.com`,
    role,
    isInactive: false,
    isSsoOnly: false,
    emailVerified: true,
    createdAt: '2030-01-01T00:00:00Z',
    ...overrides,
})

const meta = {
    title: 'pages/admin/Users',
    component: UsersPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/users', [user(1, 'Kirill', 'admin'), user(2, 'Alex', 'user'), user(3, 'Guest', 'guest')]),
            reply('get', '/admin/mail', { enabled: false, problem: null, appUrlSet: true, writable: true }),
            ...settingsApi(),
        ),
} satisfies Meta<typeof UsersPage>

export default meta
type Story = StoryObj<typeof meta>

const createButton = async (canvasElement: HTMLElement) =>
    (await within(canvasElement).findAllByRole('button', { name: /add|create|new/i }))[0]

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByText('alex@example.com'))[0]).toBeVisible()
        await expect(await createButton(canvasElement)).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/users')) }

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await createButton(canvasElement))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
