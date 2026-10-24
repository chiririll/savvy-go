import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply } from '@/test/api-mocks'
import type { InviteMethods } from '@/types/spaces'

import { InviteFields } from './InviteFields'

const methods = (value: InviteMethods) => reply('get', '/invitation-methods', value)

const meta = {
    title: 'features/spaces/InviteFields',
    component: InviteFields,
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => msw.use(methods({ email: true, link: true })),
    args: { submitLabel: 'Invite', onSubmit: fn(), onEdit: fn() },
} satisfies Meta<typeof InviteFields>

export default meta
type Story = StoryObj<typeof meta>

const emailField = (canvasElement: HTMLElement) => within(canvasElement).findByPlaceholderText(/email address/i)

export const BothMethods: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /by email/i })).toHaveAttribute(
            'aria-pressed',
            'true',
        )
        await expect(await emailField(canvasElement)).toBeVisible()
    },
}

// Nothing is shown until the server says how invitations may be made.
export const WaitingForTheServer: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/invitation-methods')),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('textbox')).toBeNull()
    },
}

export const InvitedByEmail: Story = {
    play: async ({ canvasElement, args }) => {
        const field = await emailField(canvasElement)
        await userEvent.type(field, '  alex@example.com ')
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Invite' }))
        await waitFor(() =>
            expect(args.onSubmit).toHaveBeenCalledWith({ email: 'alex@example.com', note: null, role: 'editor' }),
        )
        await expect(field).toHaveValue('')
    },
}

export const EmptyAddressIsNotSent: Story = {
    play: async ({ canvasElement, args }) => {
        await emailField(canvasElement)
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Invite' }))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const TypingClearsTheError: Story = {
    args: { error: 'This address is already a member.' },
    play: async ({ canvasElement, args }) => {
        const field = await emailField(canvasElement)
        await expect(field).toHaveAttribute('aria-invalid', 'true')
        await userEvent.type(field, 'a')
        await expect(args.onEdit).toHaveBeenCalled()
    },
}

// The parent refused it: the address stays to be corrected.
export const RefusedKeepsTheAddress: Story = {
    args: { onSubmit: fn(() => false) },
    play: async ({ canvasElement }) => {
        const field = await emailField(canvasElement)
        await userEvent.type(field, 'alex@example.com{Enter}')
        await expect(field).toHaveValue('alex@example.com')
    },
}

export const LinkWithANoteAndRole: Story = {
    parameters: { a11y: overlayA11y },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /copy a link/i }))
        await userEvent.type(canvas.getByLabelText(/note/i), ' For grandma ')
        await userEvent.click(canvas.getByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: 'Viewer' }))
        await userEvent.click(canvas.getByRole('button', { name: 'Invite' }))
        await waitFor(() =>
            expect(args.onSubmit).toHaveBeenCalledWith({ email: null, note: 'For grandma', role: 'viewer' }),
        )
    },
}

export const LinkWithoutANote: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /copy a link/i }))
        await userEvent.click(canvas.getByRole('button', { name: 'Invite' }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledWith({ email: null, note: null, role: 'editor' }))
    },
}

export const Busy: Story = {
    args: { busy: true },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: 'Invite' })).toBeDisabled()
    },
}

// Mail is off: only links, with the reason why.
export const OnlyLinks: Story = {
    beforeEach: ({ msw }) => msw.use(methods({ email: false, link: true, emailReason: 'mail is not set up' })),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/cannot be emailed: mail is not set up/i)).toBeVisible()
        await expect(canvas.getByLabelText(/note/i)).toBeVisible()
        await expect(canvas.queryByRole('group')).toBeNull()
    },
}

// Verified addresses are required, so links are off.
export const OnlyEmail: Story = {
    beforeEach: ({ msw }) => msw.use(methods({ email: true, link: false })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/requires invitations by email/i)).toBeVisible()
        await expect(within(canvasElement).queryByRole('group')).toBeNull()
    },
}

export const Unavailable: Story = {
    beforeEach: ({ msw }) => msw.use(methods({ email: false, link: false, emailReason: 'mail is not set up' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/invitations are unavailable/i)).toBeVisible()
        await expect(within(canvasElement).queryByRole('textbox')).toBeNull()
    },
}
