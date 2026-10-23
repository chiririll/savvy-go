import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { pickerApi, rule } from '@/components/features/automation/fixtures'
import { hang, reply } from '@/test/api-mocks'

import AutomationPage from './index'

const meta = {
    title: 'pages/Automation',
    component: AutomationPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/automation-rules', [rule(), rule({ id: 2, name: 'Paused rule', isActive: false })]), ...pickerApi()),
} satisfies Meta<typeof AutomationPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Groceries from the shop')).toBeVisible()
        await expect(canvas.getByText('Paused rule')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new rule/i })).toBeVisible()
    },
}
export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/automation-rules')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/automation-rules', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/nothing|no .* yet|empty/i)).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new rule/i })).toBeNull())
        await expect(await within(canvasElement).findByText('Groceries from the shop')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new rule/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
