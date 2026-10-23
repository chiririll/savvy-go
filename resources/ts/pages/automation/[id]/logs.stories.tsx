import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes } from 'react-router-dom'
import { expect, within } from 'storybook/test'

import { rule } from '@/components/features/automation/fixtures'
import { hang, reply } from '@/test/api-mocks'

import AutomationLogsPage from './logs'

const log = (id: number, status: 'success' | 'error' | 'skipped', overrides = {}) => ({
    id,
    ruleId: 1,
    triggerEntityType: 'transaction',
    triggerEntityId: 40 + id,
    actionsExecuted: [{ type: 'set_category', result: null }],
    status,
    errorMessage: null,
    createdAt: '2030-01-05T10:00:00Z',
    ...overrides,
})

const meta = {
    title: 'pages/AutomationLogs',
    component: AutomationLogsPage,
    parameters: { route: '/automation/1/logs' },
    // The id comes from the address.
    render: () => (
        <Routes>
            <Route path="/automation/:id/logs" element={<AutomationLogsPage />} />
        </Routes>
    ),
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/automation-rules/1', rule()),
            reply('get', '/automation-rules/1/logs', [
                log(1, 'success'),
                log(2, 'error', { errorMessage: 'Category was deleted', actionsExecuted: null }),
                log(3, 'skipped', { triggerEntityType: null, actionsExecuted: [] }),
            ]),
        ),
} satisfies Meta<typeof AutomationLogsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Logs: Groceries from the shop')).toBeVisible()
        await expect(await canvas.findByText('Category was deleted')).toBeVisible()
        await expect(canvas.getByText('transaction #41')).toBeVisible()
        // The way back to the list.
        await expect(canvas.getByRole('link')).toHaveAttribute('href', '/automation')
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/automation-rules/1')) }

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/automation-rules/1/logs', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no logs yet/i)).toBeVisible()
    },
}
