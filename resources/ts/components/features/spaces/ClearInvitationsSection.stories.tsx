import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { fail, spy } from '@/test/api-mocks'

import { ClearInvitationsSection } from './ClearInvitationsSection'

const sent: { bodies: unknown[] } = { bodies: [] }

const meta = {
    title: 'features/spaces/ClearInvitationsSection',
    component: ClearInvitationsSection,
    beforeEach: ({ msw }) => {
        const clear = spy('post', '/admin/invitations/clear', { cleared: 3 })
        sent.bodies = clear.bodies
        msw.use(clear.handler)
    },
} satisfies Meta<typeof ClearInvitationsSection>

export default meta
type Story = StoryObj<typeof meta>

export const Section: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /clear for all/i })).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const ClearedForAll: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /clear for all/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{}]))
        await expect(await screen.findByText(/cleared 3 invitations/i)).toBeVisible()
    },
}

export const ClearedForOneAddress: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const field = canvas.getByRole('textbox')
        await userEvent.type(field, ' alex@example.com ')
        await userEvent.click(canvas.getByRole('button', { name: /for this address/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ email: 'alex@example.com' }]))
        await waitFor(() => expect(field).toHaveValue(''))
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/admin/invitations/clear', 'Database is locked')),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox'), 'alex@example.com')
        await userEvent.click(canvas.getByRole('button', { name: /for this address/i }))
        await expect(await screen.findByText(/database is locked/i)).toBeVisible()
        await expect(canvas.getByRole('textbox')).toHaveValue('alex@example.com')
    },
}
