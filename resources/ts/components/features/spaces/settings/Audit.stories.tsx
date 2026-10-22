import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import type { AuditEntry } from '@/types/spaces'

import { Audit } from './Audit'

const entry = (overrides: Partial<AuditEntry> = {}): AuditEntry => ({
    id: 1,
    actor: 'Root',
    action: 'set_quota',
    targetUserId: null,
    details: '50 MB',
    createdAt: '2030-01-05T10:00:00Z',
    ...overrides,
})

const meta = {
    title: 'features/spaces/settings/Audit',
    component: Audit,
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/audit', [
                entry(),
                entry({ id: 2, action: 'assign_admin', details: 'alex@example.com', targetUserId: 2 }),
                entry({ id: 3, actor: '', action: 'restore_deleted_space', details: '' }),
            ]),
        ),
    args: { space: space() },
} satisfies Meta<typeof Audit>

export default meta
type Story = StoryObj<typeof meta>

export const Entries: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/root — changed the size limit \(50 MB\)/i)).toBeVisible()
        // No actor reads as "someone", and no details leave out the brackets.
        await expect(canvas.getByText(/^someone — restored the space$/i)).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

// An action this build does not know is shown as it came.
export const UnknownAction: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/audit', [entry({ action: 'export_space', details: '' })])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/root — export_space/i)).toBeVisible()
    },
}

export const Nothing: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/audit', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/nothing yet/i)).toBeVisible()
    },
}
