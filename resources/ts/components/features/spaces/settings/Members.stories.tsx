import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, spy } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import type { SpaceMember } from '@/types/spaces'

import { Members } from './Members'

// The signed-in user of every story is user 1.
const members: SpaceMember[] = [
    { userId: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', createdAt: null },
    { userId: 2, name: 'Alex', email: 'alex@example.com', role: 'editor', createdAt: null },
    { userId: 3, name: 'Sam', email: 'sam@example.com', role: 'viewer', createdAt: null },
]

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/settings/Members',
    component: Members,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/members', members)),
    args: { space: space(), isAdmin: true },
} satisfies Meta<typeof Members>

export default meta
type Story = StoryObj<typeof meta>

export const AsAdmin: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('(you)')).toBeVisible()
        await expect(canvas.getAllByRole('combobox')).toHaveLength(3)
        // There is no button to remove yourself.
        await expect(canvas.getAllByRole('button', { name: /remove/i })).toHaveLength(2)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const AsMember: Story = {
    args: { space: space('editor'), isAdmin: false },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Viewer')).toBeVisible()
        await expect(canvas.queryByRole('combobox')).toBeNull()
        await expect(canvas.queryByRole('button')).toBeNull()
    },
}

export const RoleChanged: Story = {
    beforeEach: ({ msw }) => {
        const role = spy('patch', '/members/2', members)
        sent.bodies = role.bodies
        msw.use(role.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('combobox'))[1])
        await userEvent.click(await screen.findByRole('option', { name: 'Viewer' }))
        await waitFor(() => expect(sent.bodies).toEqual([{ role: 'viewer' }]))
        await expect(await screen.findByText(/role changed/i)).toBeVisible()
    },
}

export const Removed: Story = {
    beforeEach: ({ msw }) => {
        const remove = spy('delete', '/members/3')
        sent.bodies = remove.bodies
        msw.use(remove.handler)
    },
    play: async ({ canvasElement }) => {
        await userEvent.click((await within(canvasElement).findAllByRole('button', { name: /remove/i }))[1])
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(await screen.findByText(/member removed/i)).toBeVisible()
    },
}

export const LongNames: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/members', [
                members[0],
                {
                    ...members[1],
                    name: 'Alexandra Konstantinopolskaya-Vasilievskaya',
                    email: 'alexandra.konstantinopolskaya-vasilievskaya@a-long-company-domain.example.com',
                },
            ]),
        ),
}
