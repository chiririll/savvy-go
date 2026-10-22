import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { fail, spy } from '@/test/api-mocks'

import { SessionsSection } from './SessionsSection'

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/security/SessionsSection',
    component: SessionsSection,
    parameters: { a11y: overlayA11y },
} satisfies Meta<typeof SessionsSection>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    await userEvent.click(within(canvasElement).getByRole('button', { name: /sign out other/i }))
    return within(await screen.findByRole('dialog'))
}

export const Section: Story = {}
export const InRussian: Story = { globals: { locale: 'ru' } }

export const ConfirmAsked: Story = {
    play: async ({ canvasElement }) => {
        const dialog = await open(canvasElement)
        await expect(dialog.getByRole('button', { name: /cancel/i })).toBeVisible()
    },
}

export const Cancelled: Story = {
    play: async ({ canvasElement }) => {
        const dialog = await open(canvasElement)
        await userEvent.click(dialog.getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

export const LoggedOut: Story = {
    beforeEach: ({ msw }) => {
        const logout = spy('post', '/auth/logout-others', { message: 'ok', revoked: 2 }, 200, false)
        sent.bodies = logout.bodies
        msw.use(logout.handler)
    },
    play: async ({ canvasElement }) => {
        const dialog = await open(canvasElement)
        const buttons = dialog.getAllByRole('button').filter((b) => !/cancel|close/i.test(b.textContent ?? ''))
        await userEvent.click(buttons[buttons.length - 1])
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}

// A failed request leaves the question open.
export const LogoutFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/logout-others', 'Something went wrong')),
    play: async ({ canvasElement }) => {
        const dialog = await open(canvasElement)
        const buttons = dialog.getAllByRole('button').filter((b) => !/cancel|close/i.test(b.textContent ?? ''))
        await userEvent.click(buttons[buttons.length - 1])
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(screen.getByRole('dialog')).toBeVisible()
    },
}
