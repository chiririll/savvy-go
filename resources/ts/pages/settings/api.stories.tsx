import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { capabilities } from '@/components/features/api-tokens/story-fixtures'
import { hang, raw, reply } from '@/test/api-mocks'

import ApiSettingsPage from './api'

const token = (id: number, name: string, overrides = {}) => ({
    id,
    name,
    prefix: `sav_${id}abc`,
    kind: 'personal',
    permissions: ['transactions:read'],
    all_spaces: true,
    space_ids: [],
    expires_at: null,
    last_used_at: '2030-01-05T10:00:00Z',
    created_at: '2030-01-01T00:00:00Z',
    ...overrides,
})

const meta = {
    title: 'pages/settings/Api',
    component: ApiSettingsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/auth/api-tokens', [token(1, 'Spreadsheet'), token(2, 'Phone shortcut')]),
            reply('get', '/auth/api-token-capabilities', capabilities),
            raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }),
            reply('get', '/sync-group', null),
        ),
} satisfies Meta<typeof ApiSettingsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Spreadsheet')).toBeVisible()
        await expect(canvas.getByText('Phone shortcut')).toBeVisible()
        await expect(canvas.getByRole('link', { name: /docs/i })).toHaveAttribute('href', '/api/docs')
        await expect(canvas.getByRole('button', { name: /new token|add|create/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/api-tokens')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/api-tokens', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no .* yet|no tokens/i)).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new token|add|create/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
