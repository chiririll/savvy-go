import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'

import { DoneFooter, DoneStep } from './DoneStep'
import type { InvitationOutcome } from './send-invitations'

const created = { ...space(), id: 9, name: 'Family' }

const outcomes: InvitationOutcome[] = [
    {
        invitation: { key: 'i1', email: 'alex@example.com', note: null, role: 'editor' },
        link: null,
        emailed: true,
        error: null,
    },
    {
        invitation: { key: 'i2', email: null, note: 'For grandma', role: 'viewer' },
        link: 'https://savvy.example.com/invite/tok123',
        emailed: false,
        error: null,
    },
    {
        invitation: { key: 'i3', email: 'sam@example.com', note: null, role: 'editor' },
        link: null,
        emailed: false,
        error: 'sam@example.com was invited a moment ago',
    },
]

const onInvite = fn()
const onLeave = fn()

// The done screen with its footer, as the modal shows them.
const meta = {
    title: 'features/spaces/create/DoneStep',
    component: DoneStep,
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => {
        onInvite.mockClear()
        onLeave.mockClear()
        msw.use(reply('get', '/invitation-methods', { email: true, link: true }))
    },
    args: { space: created, inviting: false, invitations: [], onInvitationsChange: fn(), sent: null },
    render: (args) => (
        <div className="space-y-4">
            <DoneStep {...args} />
            <div className="flex justify-end gap-2">
                <DoneFooter
                    inviting={args.inviting}
                    sent={args.sent}
                    isSending={false}
                    onInvite={onInvite}
                    onLeave={onLeave}
                />
            </div>
        </div>
    ),
} satisfies Meta<typeof DoneStep>

export default meta
type Story = StoryObj<typeof meta>

export const Created: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/"Family" is ready/)).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /invite members/i }))
        await expect(onInvite).toHaveBeenCalled()
        await userEvent.click(canvas.getByRole('button', { name: /continue/i }))
        await expect(onLeave).toHaveBeenCalled()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const Inviting: Story = {
    args: { inviting: true },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/no one added yet/i)).toBeVisible()
        await expect(canvas.queryByRole('button', { name: /invite members/i })).toBeNull()
    },
}

// Links to copy and refusals stay on screen to be read.
export const Sent: Story = {
    args: { sent: outcomes },
    beforeEach: () => {
        Object.defineProperty(navigator, 'clipboard', { value: { writeText: fn(async () => {}) }, configurable: true })
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/editor · email sent/i)).toBeVisible()
        await expect(canvas.getByText(/not sent: sam@example.com was invited a moment ago/i)).toBeVisible()
        await expect(canvas.getByText(/invite them again from the space settings/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /^done$/i })).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /copy link/i }))
        await waitFor(() =>
            expect(navigator.clipboard.writeText).toHaveBeenCalledWith('https://savvy.example.com/invite/tok123'),
        )
        await expect(await screen.findByText(/link copied/i)).toBeVisible()
    },
}

export const AllEmailed: Story = {
    args: { sent: [outcomes[0]] },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByRole('button', { name: /copy link/i })).toBeNull()
        await expect(canvas.queryByText(/invite them again/i)).toBeNull()
    },
}
